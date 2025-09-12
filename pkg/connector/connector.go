package connector

import (
	"context"
	"io"

	"google.golang.org/protobuf/proto"

	"github.com/conductorone/baton-freshdesk/pkg/client"
	cfg "github.com/conductorone/baton-freshdesk/pkg/config"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
)

type Connector struct {
	client *client.FreshdeskClient
}

// ResourceSyncers returns a ResourceSyncer for each resource type that should be synced from the upstream service.
func (d *Connector) ResourceSyncers(_ context.Context) []connectorbuilder.ResourceSyncer {
	return []connectorbuilder.ResourceSyncer{
		newUserBuilder(d.client),
		newRoleBuilder(d.client),
		newGroupBuilder(d.client),
	}
}

// Asset takes an input AssetRef and attempts to fetch it using the connector's authenticated http client
// It streams a response, always starting with a metadata object, following by chunked payloads for the asset.
func (d *Connector) Asset(ctx context.Context, asset *v2.AssetRef) (string, io.ReadCloser, error) {
	return "", nil, nil
}

// Metadata returns metadata about the connector.
func (d *Connector) Metadata(_ context.Context) (*v2.ConnectorMetadata, error) {
	return &v2.ConnectorMetadata{
		DisplayName: "Freshdesk Connector",
		Description: "Connector to obtain data from Freshdesk.",
		AccountCreationSchema: &v2.ConnectorAccountCreationSchema{
			FieldMap: map[string]*v2.ConnectorAccountCreationSchema_Field{
				"name": {
					DisplayName: "Name",
					Required:    false,
					Field: &v2.ConnectorAccountCreationSchema_Field_StringField{
						StringField: &v2.ConnectorAccountCreationSchema_StringField{},
					},
					Order: 1,
				},
				"occasional": {
					DisplayName: "Occasional",
					Required:    false,
					Field: &v2.ConnectorAccountCreationSchema_Field_BoolField{
						BoolField: &v2.ConnectorAccountCreationSchema_BoolField{},
					},
					Order: 2,
				},
				"email": {
					DisplayName: "Email",
					Required:    true,
					Field: &v2.ConnectorAccountCreationSchema_Field_StringField{
						StringField: &v2.ConnectorAccountCreationSchema_StringField{},
					},
					Order: 3,
				},
				"language": {
					DisplayName: "Language",
					Required:    false,
					Field: &v2.ConnectorAccountCreationSchema_Field_StringField{
						StringField: &v2.ConnectorAccountCreationSchema_StringField{},
					},
					Order: 4,
				},
				"ticketScope": {
					DisplayName: "Ticket Scope",
					Required:    true,
					Description: "Ticket permission of the agent (1 -> Global Access, 2 -> Group Access, 3 -> Restricted Access). ",
					Field: &v2.ConnectorAccountCreationSchema_Field_IntField{
						IntField: &v2.ConnectorAccountCreationSchema_IntField{
							DefaultValue: proto.Int32(1),
						},
					},
					Order: 5,
				},
			},
		},
	}, nil
}

// Validate is called to ensure that the connector is properly configured. It should exercise any API credentials
// to be sure that they are valid. // TODO Apply validations.
func (d *Connector) Validate(ctx context.Context) (annotations.Annotations, error) {
	return nil, nil
}

// New returns a new instance of the connector.
func New(ctx context.Context, cfg *cfg.Freshdesk) (*Connector, error) {
	freshdeskClient, err := client.New(
		ctx,
		client.WithDomain(cfg.Domain),
		client.WithBearerToken(cfg.ApiKey),
	)

	if err != nil {
		return nil, err
	}

	return &Connector{
		client: freshdeskClient,
	}, nil
}
