package connector

import (
	"context"
	"errors"
	"github.com/conductorone/baton-freshdesk/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"slices"
	"strconv"
	"sync"
)

type roleBuilder struct {
	resourceType      *v2.ResourceType
	agentsDetails     []client.Agent
	agentDetailsMutex sync.RWMutex
	client            *client.FreshdeskClient
}

func (r *roleBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return r.resourceType
}

func (r *roleBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var rv []*v2.Resource
	bag, pageToken, err := getToken(pToken, roleResourceType)

	roles, nextPageToken, annotation, err := r.client.ListRoles(ctx, client.PageOptions{
		Page:    pageToken,
		PerPage: pToken.Size,
	})
	if err != nil {
		return nil, "", nil, err
	}

	err = bag.Next(nextPageToken)
	if err != nil {
		return nil, "", nil, err
	}

	for _, role := range *roles {
		roleCopy := role
		roleResource, err := parseIntoRoleResource(ctx, &roleCopy, parentResourceID)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, roleResource)
	}

	nextPageToken, err = bag.Marshal()
	if err != nil {
		return nil, "", nil, err
	}

	return rv, nextPageToken, annotation, nil
}

func (r *roleBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement
	permissionName := "assigned"

	assigmentOptions := []entitlement.EntitlementOption{
		entitlement.WithGrantableTo(userResourceType),
		entitlement.WithDescription(resource.Description),
		entitlement.WithDisplayName(resource.DisplayName),
	}

	rv = append(rv, entitlement.NewPermissionEntitlement(resource, permissionName, assigmentOptions...))

	return rv, "", nil, nil
}

func (r *roleBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var rv []*v2.Grant
	err := r.GetAgentsDetails(ctx)
	if err != nil {
		return nil, "", nil, err
	}

	for _, agentDetail := range r.agentsDetails {
		const permissionName = "assigned"

		value, err := strconv.Atoi(resource.Id.Resource)
		if err != nil {
			return nil, "", nil, err
		}

		if slices.Contains(agentDetail.RoleIDs, value) {
			userResource, _ := parseIntoUserResource(&agentDetail, nil)

			membershipGrant := grant.NewGrant(resource, permissionName, userResource.Id)
			rv = append(rv, membershipGrant)
		}

	}

	return rv, "", nil, nil
}

func (r *roleBuilder) GetAllAgentsIDs(ctx context.Context, pToken *pagination.Token) ([]string, error) {
	var rv []string

	for {
		bag, pageToken, err := getToken(pToken, userResourceType)

		agents, nextPageToken, _, err := r.client.ListAgents(ctx, client.PageOptions{
			Page:    pageToken,
			PerPage: pToken.Size,
		})
		if err != nil {
			return nil, err
		}

		err = bag.Next(nextPageToken)
		if err != nil {
			return nil, err
		}

		for _, agent := range *agents {
			agentID := strconv.Itoa(agent.ID)

			rv = append(rv, agentID)
		}

		nextPageToken, err = bag.Marshal()
		if err != nil {
			return nil, err
		}

		if nextPageToken == "" {
			break
		}
		pToken.Token = nextPageToken
	}

	return rv, nil
}

func (r *roleBuilder) GetAgentsDetails(ctx context.Context) error {
	r.agentDetailsMutex.Lock()
	defer r.agentDetailsMutex.Unlock()

	if r.agentsDetails != nil && len(r.agentsDetails) > 0 {
		return nil
	}

	paginationToken := pagination.Token{50, ""}
	IDs, err := r.GetAllAgentsIDs(ctx, &paginationToken)
	if err != nil {
		return err
	}

	if len(IDs) == 0 {
		return errors.New("no agents found")
	}

	for _, id := range IDs {
		agentDetail, _, err := r.client.GetAgentDetail(ctx, id)
		if err != nil {
			return err
		}

		r.agentsDetails = append(r.agentsDetails, *agentDetail)
	}

	return nil
}

func newRoleBuilder(c *client.FreshdeskClient) *roleBuilder {
	return &roleBuilder{
		resourceType: roleResourceType,
		client:       c,
	}
}

// This function parses a role from Freshdesk into a Role Resource
func parseIntoRoleResource(ctx context.Context, role *client.Role, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"id":          role.ID,
		"name":        role.Name,
		"description": role.Description,
	}

	roleTraits := []rs.RoleTraitOption{
		rs.WithRoleProfile(profile),
	}

	ret, err := rs.NewRoleResource(role.Name, roleResourceType, role.ID, roleTraits)
	if err != nil {
		return nil, err
	}

	return ret, nil
}
