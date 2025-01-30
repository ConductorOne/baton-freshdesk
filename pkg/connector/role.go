package connector

import (
	"context"
	"github.com/conductorone/baton-freshdesk/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

type roleBuilder struct {
	resourceType *v2.ResourceType
	client       *client.FreshdeskClient
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
		roleResource, err := parseIntoRoleResource(ctx, &roleCopy, nil)
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

// Entitlements always returns an empty slice for users. //TODO Analyze the case for the Roles
func (o *roleBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// Grants always returns an empty slice for users since they don't have any entitlements. //TODO Analyze the case for the Roles
func (o *roleBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
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
		"role_type":   1, //role.RoleType,  //TODO Analyze from where does this value come from. What does it represents?
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
