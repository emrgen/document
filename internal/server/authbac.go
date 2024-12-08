package server

import (
	"context"
	authv1 "github.com/emrgen/authbac/apis/v1"
	"github.com/emrgen/authbac/spicedb"
	"github.com/sirupsen/logrus"
)

var _ spicedb.PermissionService = (*TokenService)(nil)

type TokenService struct {
	auth authv1.AccessTokenServiceClient
}

func NewTokenService(auth authv1.AccessTokenServiceClient) *TokenService {
	return &TokenService{
		auth: auth,
	}
}

func (t TokenService) VerifyProjectAccess(ctx context.Context, token string) (bool, error) {
	_, err := t.auth.VerifyAccessToken(ctx, &authv1.VerifyAccessTokenRequest{
		AccessToken: token,
	})
	if err != nil {
		logrus.Errorf("failed to verify project access: %v", err)
		return false, err
	}

	return true, nil
}

type NullTokenService struct{}

var _ spicedb.PermissionService = NullTokenService{}

func NewNullTokenService() *NullTokenService {
	return &NullTokenService{}
}

func (t NullTokenService) VerifyProjectAccess(ctx context.Context, token string) (bool, error) {
	logrus.Infof("null token service: %v", token)
	return true, nil
}
