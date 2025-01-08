package server

import (
	"context"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"time"
)

//func CheckPermissionInterceptor() grpc.UnaryServerInterceptor {
//	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
//		permission, err := authx.GetProjectPermission(ctx)
//		if err != nil {
//			return nil, err
//		}
//
//		switch info.FullMethod {
//		case v1.DocumentService_CreateDocument_FullMethodName,
//			v1.DocumentService_UpdateDocument_FullMethodName,
//			v1.DocumentService_DeleteDocument_FullMethodName,
//			v1.DocumentBackupService_RestoreDocumentBackup_FullMethodName:
//			// check if the user has permission to write
//			if permission >= authbasev1.Permission_ADMIN {
//				return handler(ctx, req)
//			}
//
//		case v1.DocumentService_ListDocuments_FullMethodName,
//			v1.DocumentService_GetDocument_FullMethodName,
//			v1.DocumentBackupService_ListDocumentBackups_FullMethodName,
//			v1.DocumentBackupService_GetDocumentBackup_FullMethodName:
//			// check if the user has permission to read
//			if permission >= authbasev1.Permission_READ {
//				return handler(ctx, req)
//			}
//
//		default:
//			return nil, errors.New("unknown method")
//		}
//
//		return nil, errors.New("permission denied")
//	}
//}

func UnaryGrpcRequestTimeInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		reqTime := time.Since(start)
		logrus.Infof("request time: %v: %v", info.FullMethod, reqTime)
		return resp, err
	}
}

func UnaryRequestTimeInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req interface{},
		reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		reqTime := time.Since(start)
		logrus.Infof("request time: %v: %v", method, reqTime)
		return err
	}
}
