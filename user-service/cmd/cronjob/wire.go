// +build wireinject

package main

import (
	"user-service/internal/server"
	"user-service/internal/service"
	"user-service/internal/tasks"
	eagle "github.com/go-eagle/eagle/pkg/app"
	"github.com/google/wire"
)

func InitApp(cfg *eagle.Config, config *eagle.ServerConfig, taskCfg *tasks.Config) (*eagle.App, error) {
	wire.Build(server.ProviderSet, service.ProviderSet, newApp)
	return &eagle.App{}, nil
}
