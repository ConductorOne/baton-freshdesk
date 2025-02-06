package main

import (
	"context"
	"fmt"
	"os"

	"github.com/conductorone/baton-freshdesk/pkg/client"
	"github.com/conductorone/baton-freshdesk/pkg/connector"
	"github.com/conductorone/baton-sdk/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/field"
	"github.com/conductorone/baton-sdk/pkg/types"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var version = "dev"

func main() {
	ctx := context.Background()

	_, cmd, err := config.DefineConfiguration(
		ctx,
		"baton-freshdesk",
		getConnector,
		field.Configuration{
			Fields: ConfigurationFields,
		},
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	cmd.Version = version

	err = cmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func getConnector(ctx context.Context, v *viper.Viper) (types.ConnectorServer, error) {
	fdClient := client.NewClient()

	// Get params from Viper
	fdApiKey := v.GetString(apiKey)
	fdDomain := v.GetString(domain)

	l := ctxzap.Extract(ctx)

	fdClient = fdClient.WithBearerToken(fdApiKey).WithDomain(fdDomain)

	if err := ValidateConfig(v); err != nil {
		return nil, err
	}

	cb, err := connector.New(ctx, fdApiKey, fdDomain, fdClient)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}

	opts := make([]connectorbuilder.Opt, 0)

	c, err := connectorbuilder.NewConnector(ctx, cb, opts...)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}

	return c, nil
}
