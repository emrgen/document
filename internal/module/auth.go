package module

import (
	"context"
	"errors"
	"github.com/emrgen/authbac/spicedb"
	"github.com/emrgen/authbac/token"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	authorization = "authorization"
)

func UnaryServerAuthTokenInterceptor(verifyToken spicedb.TokenService) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		accessToken, err := accessTokenFromHeader(ctx, authorization)
		if err != nil {
			return nil, err
		}

		_, _, err = token.DecodeProjectToken(accessToken)
		if err != nil {
			return nil, err
		}

		ok, err := verifyToken.VerifyProjectAccess(ctx, accessToken)
		if err != nil {
			return nil, err
		}

		if !ok {
			return nil, errors.New("access token verification failed")
		}

		return handler(ctx, req)
	}
}

func accessTokenFromHeader(ctx context.Context, header string) (string, error) {
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("metadata not found")
	}

	val, ok := headers[header]
	if !ok {
		return "", errors.New("header not found")
	}

	authToken := val[0]
	if authToken == "" {
		return "", errors.New("authToken not found")
	}

	// remove prefix Bearer
	return authToken[7:], nil
}
