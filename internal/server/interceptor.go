package server

import (
	"context"
	"errors"
	v1 "github.com/emrgen/document/apis/v1"
	tinysv1 "github.com/emrgen/tinys/apis/v1"
	"google.golang.org/grpc"
)

func CheckPermissionInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		permission, ok := ctx.Value("projectPermission").(tinysv1.MemberPermission)
		if !ok {
			return nil, errors.New("missing project permission, check if the user is a member of the project")
		}

		switch info.FullMethod {
		case v1.DocumentService_CreateDocument_FullMethodName,
			v1.DocumentService_UpdateDocument_FullMethodName,
			v1.DocumentService_DeleteDocument_FullMethodName:
			// check if the user has permission to write
			if permission >= tinysv1.MemberPermission_MEMBER_WRITE {
				return handler(ctx, req)
			}

		case v1.DocumentService_ListDocuments_FullMethodName,
			v1.DocumentService_GetDocument_FullMethodName:
			// check if the user has permission to read
			if permission >= tinysv1.MemberPermission_MEMBER_READ {
				return handler(ctx, req)
			}

		default:
			return nil, errors.New("unknown method")
		}

		return nil, errors.New("permission denied")
	}

}
