package config

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

const (
	apiKey = "api-key" //nolint:gosec // Not a credential
	domain = "domain"
)

var (
	apiKeyField = field.StringField(
		apiKey,
		field.WithDisplayName("API Key"),
		field.WithIsSecret(true),
		field.WithRequired(true),
		field.WithDescription("Freshdesk account API key"),
	)
	domainField = field.StringField(
		domain,
		field.WithDisplayName("Domain"),
		field.WithRequired(true),
		field.WithDescription("Freshdesk account domain"),
	)

	ConfigurationFields = []field.SchemaField{apiKeyField, domainField}

	FieldRelationships = []field.SchemaFieldRelationship{}
)

//go:generate go run ./gen
var Configuration = field.NewConfiguration(
	ConfigurationFields,
	field.WithConstraints(FieldRelationships...),
	field.WithConnectorDisplayName("Freshdesk"),
	field.WithHelpUrl("/docs/baton/freshdesk"),
	field.WithIconUrl("/static/app-icons/freshdesk.svg"),
)
