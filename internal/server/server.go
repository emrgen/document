package server

import (
	"context"
	"errors"
	"fmt"
	gatewayfile "github.com/black-06/grpc-gateway-file"
	v1 "github.com/emrgen/document/apis/v1"
	"github.com/emrgen/document/internal/cache"
	"github.com/emrgen/document/internal/config"
	"github.com/emrgen/document/internal/service"
	gopackv1 "github.com/emrgen/gopack/apis/v1"
	"github.com/emrgen/gopack/token"
	tinysv1 "github.com/emrgen/tinys/apis/v1"
	authz "github.com/emrgen/tinys/aythz"
	"github.com/gobuffalo/packr"
	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcvalidator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

func Start(grpcPort, httpPort string) error {
	var err error

	grpcPort = ":" + grpcPort
	httpPort = ":" + httpPort

	cnf := config.LoadConfig()
	rdb := config.GetDb(cnf)

	gl, err := net.Listen("tcp", grpcPort)
	if err != nil {
		return err
	}

	rl, err := net.Listen("tcp", httpPort)
	if err != nil {
		return err
	}

	authConn, err := grpc.NewClient(":4000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer authConn.Close()
	// authClient provides the token service
	authClient := gopackv1.NewTokenServiceClient(authConn)

	tinyConn, err := grpc.NewClient(":4010", grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer tinyConn.Close()
	// tinyClient provides the membership service
	tinyClient := tinysv1.NewMembershipServiceClient(tinyConn)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpcmiddleware.ChainUnaryServer(
			grpcvalidator.UnaryServerInterceptor(),
			// verify the token
			token.VerifyTokenInterceptor(authClient),
			// get the project permissions
			authz.InjectPermissionInterceptor(tinyClient),
			// check the project permissions
			CheckPermissionInterceptor(),
			UnaryGrpcRequestTimeInterceptor(),
		)),
	)

	// connect the rest gateway to the grpc server
	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.HTTPBodyMarshaler{
			Marshaler: &runtime.JSONPb{
				MarshalOptions: protojson.MarshalOptions{
					EmitUnpopulated: true,
				},
				UnmarshalOptions: protojson.UnmarshalOptions{
					DiscardUnknown: true,
				},
			},
		}),
		gatewayfile.WithHTTPBodyMarshaler(),
	)

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(UnaryRequestTimeInterceptor()),
	}
	endpoint := "localhost" + grpcPort

	// create token service authClient
	redis, err := cache.NewRedis()
	if err != nil {
		return err
	}

	// Register the grpc server
	v1.RegisterDocumentServiceServer(grpcServer, service.NewDocumentService(rdb, redis))

	// Register the rest gateway
	if err = v1.RegisterDocumentServiceHandlerFromEndpoint(context.TODO(), mux, endpoint, opts); err != nil {
		return err
	}

	apiMux := http.NewServeMux()
	openapiDocs := packr.NewBox("../../docs/v1")
	docsPath := "/v1/docs/"
	apiMux.Handle(docsPath, http.StripPrefix(docsPath, http.FileServer(openapiDocs)))
	apiMux.Handle("/", mux)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // All origins are allowed
		AllowedMethods:   []string{"GET", "POST", "DELETE", "PUT"},
		AllowedHeaders:   []string{"Authorization"},
		AllowCredentials: true,
	})

	restServer := &http.Server{
		Addr:    httpPort,
		Handler: c.Handler(apiMux),
	}

	// make sure to wait for the servers to stop before exiting
	var wg sync.WaitGroup

	wg.Add(1)
	// Start the grpc server
	go func() {
		defer wg.Done()
		logrus.Info("starting rest gateway on: ", httpPort)
		logrus.Info("click on the following link to view the API documentation: http://localhost", httpPort, "/v1/docs/")
		if err := restServer.Serve(rl); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				logrus.Errorf("error starting rest gateway: %v", err)
			}
		}
		logrus.Infof("rest gateway stopped")
	}()

	// Start the rest gateway
	wg.Add(1)
	go func() {
		defer wg.Done()
		logrus.Info("starting grpc server on: ", grpcPort)
		if err := grpcServer.Serve(gl); err != nil {
			logrus.Infof("grpc failed to start: %v", err)
		}
		logrus.Infof("grpc server stopped")
	}()

	time.Sleep(1 * time.Second)
	logrus.Infof("Press Ctrl+C to stop the server")

	// listen for interrupt signal to gracefully shut down the server
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, unix.SIGTERM, unix.SIGINT, unix.SIGTSTP)
	<-sigs
	// clean Ctrl+C output
	fmt.Println()

	grpcServer.Stop()
	err = restServer.Shutdown(context.Background())
	if err != nil {
		logrus.Errorf("error stopping rest gateway: %v", err)
	}

	wg.Wait()

	return nil
}
