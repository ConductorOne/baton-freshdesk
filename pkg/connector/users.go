package connector

import (
	"context"
	"fmt"
	"strconv"

	"github.com/conductorone/baton-freshdesk/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

type userBuilder struct {
	resourceType *v2.ResourceType
	client       *client.FreshdeskClient
}

var _ connectorbuilder.AccountManagerV2 = &userBuilder{}

func (u *userBuilder) ResourceType(_ context.Context) *v2.ResourceType {
	return userResourceType
}

// List returns all the users from the database as resource objects.
// Users include a UserTrait because they are the 'shape' of a standard user.
func (u *userBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, attrs rs.SyncOpAttrs) ([]*v2.Resource, *rs.SyncOpResults, error) {
	var rv []*v2.Resource

	bag, pageToken, err := getToken(&attrs.PageToken, userResourceType)
	if err != nil {
		return nil, nil, err
	}

	agents, nextPageToken, annotation, err := u.client.ListAgents(ctx, client.PageOptions{
		Page:    pageToken,
		PerPage: attrs.PageToken.Size,
	})
	if err != nil {
		return nil, nil, err
	}

	err = bag.Next(nextPageToken)
	if err != nil {
		return nil, nil, err
	}

	for _, agent := range agents {
		userResource, err := parseIntoUserResource(agent, parentResourceID)
		if err != nil {
			return nil, nil, err
		}
		rv = append(rv, userResource)
	}

	nextPageToken, err = bag.Marshal()
	if err != nil {
		return nil, nil, err
	}

	return rv, &rs.SyncOpResults{
		NextPageToken: nextPageToken,
		Annotations:   annotation,
	}, nil
}

// parseIntoUserResource - This function parses an Agent (users from Freshdesk) into a User Resource.
func parseIntoUserResource(agent *client.Agent, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	var userStatus = v2.UserTrait_Status_STATUS_ENABLED

	profile := map[string]interface{}{
		"user_id":    agent.ID,
		"login":      agent.Contact.Email,
		"first_name": agent.Contact.Name,
		"last_name":  agent.Contact.Name,
		"email":      agent.Contact.Email,
		"is_agent":   true,
	}

	userTraits := []rs.UserTraitOption{
		rs.WithUserProfile(profile),
		rs.WithStatus(userStatus),
		rs.WithUserLogin(agent.Contact.Email),
		rs.WithEmail(agent.Contact.Email, true),
	}

	displayName := agent.Contact.Name
	if displayName == "" {
		displayName = agent.Contact.Email
	}

	ret, err := rs.NewUserResource(
		displayName,
		userResourceType,
		agent.ID,
		userTraits,
		rs.WithParentResourceID(parentResourceID),
	)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// Entitlements always returns an empty slice for users.
func (u *userBuilder) Entitlements(_ context.Context, _ *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Entitlement, *rs.SyncOpResults, error) {
	return nil, &rs.SyncOpResults{}, nil
}

// Grants returns role and group grants for the given user by fetching their agent detail.
func (u *userBuilder) Grants(ctx context.Context, resource *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Grant, *rs.SyncOpResults, error) {
	agentDetail, _, err := u.client.GetAgentDetail(ctx, resource.Id.Resource)
	if err != nil {
		return nil, nil, fmt.Errorf("freshdesk-connector: failed to get agent detail: %w", err)
	}

	var rv []*v2.Grant

	for _, roleID := range agentDetail.RoleIDs {
		roleGrant := grant.NewGrant(&v2.Resource{Id: &v2.ResourceId{
			ResourceType: roleResourceType.Id,
			Resource:     strconv.FormatInt(roleID, 10),
		}}, "assigned", resource.Id)
		rv = append(rv, roleGrant)
	}

	for _, groupID := range agentDetail.GroupIDs {
		groupGrant := grant.NewGrant(&v2.Resource{Id: &v2.ResourceId{
			ResourceType: groupResourceType.Id,
			Resource:     strconv.FormatInt(groupID, 10),
		}}, "member", resource.Id)
		rv = append(rv, groupGrant)
	}

	return rv, &rs.SyncOpResults{}, nil
}

// CreateAccountCapabilityDetails returns the account provisioning capabilities of this connector.
// In this case, only account creation without password is supported.
func (u *userBuilder) CreateAccountCapabilityDetails(
	_ context.Context,
) (*v2.CredentialDetailsAccountProvisioning, annotations.Annotations, error) {
	return &v2.CredentialDetailsAccountProvisioning{
		SupportedCredentialOptions: []v2.CapabilityDetailCredentialOption{
			v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD,
		},
		PreferredCredentialOption: v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD,
	}, nil, nil
}

func (o *userBuilder) CreateAccount(
	ctx context.Context,
	accountInfo *v2.AccountInfo,
	credentialOptions *v2.LocalCredentialOptions,
) (
	connectorbuilder.CreateAccountResponse,
	[]*v2.PlaintextData,
	annotations.Annotations,
	error,
) {
	// Extract fields from the profile
	profile := accountInfo.GetProfile().AsMap()

	var (
		email       string
		occasional  bool
		ticketScope int
		language    string
		name        string
	)

	if v, ok := profile["email"].(string); ok {
		email = v
	}

	if v, ok := profile["occasional"].(bool); ok {
		occasional = v
	}

	if v, ok := profile["ticketScope"]; ok {
		switch t := v.(type) {
		case int:
			ticketScope = t
		case float64:
			ticketScope = int(t)
		}
	}

	if v, ok := profile["language"].(string); ok {
		language = v
	}

	if v, ok := profile["name"].(string); ok {
		name = v
	}

	payload := client.CreateAgentPayload{
		Email:       email,
		TicketScope: ticketScope,
		Occasional:  occasional,
	}

	if language != "" {
		payload.Language = language
	}
	if name != "" {
		payload.Name = name
	}

	// Create a new user in Freshdesk
	agent, annotation, err := o.client.CreateAgent(ctx, payload)
	if err != nil {
		return nil, nil, annotation, err
	}

	userResource, err := parseIntoUserResource(agent, nil)
	if err != nil {
		return nil, nil, annotation, err
	}

	resp := &v2.CreateAccountResponse_SuccessResult{Resource: userResource, IsCreateAccountResult: true}

	return resp, nil, annotation, nil
}

func (u *userBuilder) Delete(ctx context.Context, resourceId *v2.ResourceId) (annotations.Annotations, error) {
	profile, err := u.client.DeleteAgent(ctx, resourceId.Resource)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

func newUserBuilder(c *client.FreshdeskClient) *userBuilder {
	return &userBuilder{
		resourceType: userResourceType,
		client:       c,
	}
}
