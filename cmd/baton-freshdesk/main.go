package main

import (
	"context"

	cfg "github.com/conductorone/baton-freshdesk/pkg/config"
	"github.com/conductorone/baton-freshdesk/pkg/connector"
	"github.com/conductorone/baton-sdk/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/connectorrunner"
)

var version = "dev"

func main() {
	ctx := context.Background()
	config.RunConnector(
		ctx,
		"baton-freshdesk",
		version,
		cfg.Configuration,
		connector.New,
		connectorrunner.WithSessionStoreEnabled(),
		connectorrunner.WithProvisioningEnabled(),
		connectorrunner.WithDefaultCapabilitiesConnectorBuilderV2(&connector.Connector{}),
	)
}
