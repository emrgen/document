package service

import (
	"gorm.io/gorm"
	"net"
)

type Config struct {
	grpc net.Listener
	rest net.Listener
	db   *gorm.DB
}

func Register(cfg *Config) {

}
