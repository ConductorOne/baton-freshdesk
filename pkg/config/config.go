package config

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

const (
	apiKey  = "api-key"
	domain  = "domain"
	baseURL = "base-url"
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
	BaseURLField = field.StringField(
		baseURL,
		field.WithDisplayName("Base URL"),
		field.WithDescription("Override the Freshdesk API URL (for testing)"),
		field.WithHidden(true),
		field.WithExportTarget(field.ExportTargetCLIOnly),
	)

	ConfigurationFields = []field.SchemaField{apiKeyField, domainField, BaseURLField}

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
