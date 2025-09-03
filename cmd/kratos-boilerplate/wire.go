//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"kratos-boilerplate/internal/biz"
	"kratos-boilerplate/internal/conf"
	"kratos-boilerplate/internal/data"
	"kratos-boilerplate/internal/pkg/plugin"
	"kratos-boilerplate/internal/server"
	"kratos-boilerplate/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, *conf.Auth, log.Logger) (*kratos.App, func(), error) {
	wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, plugin.ProviderSet, newApp)
	return nil, nil, nil
}

