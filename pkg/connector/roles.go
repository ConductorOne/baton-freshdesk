package connector

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/conductorone/baton-freshdesk/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/types/entitlement"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

type roleBuilder struct {
	resourceType *v2.ResourceType
	client       *client.FreshdeskClient
}

func (r *roleBuilder) ResourceType(_ context.Context) *v2.ResourceType {
	return r.resourceType
}

func (r *roleBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, attrs rs.SyncOpAttrs) ([]*v2.Resource, *rs.SyncOpResults, error) {
	var rv []*v2.Resource
	bag, pageToken, err := getToken(&attrs.PageToken, roleResourceType)
	if err != nil {
		return nil, nil, err
	}

	roles, nextPageToken, annotation, err := r.client.ListRoles(ctx, client.PageOptions{
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

	for _, role := range roles {
		roleResource, err := parseIntoRoleResource(ctx, role, parentResourceID)
		if err != nil {
			return nil, nil, err
		}
		rv = append(rv, roleResource)
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

func (r *roleBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Entitlement, *rs.SyncOpResults, error) {
	var rv []*v2.Entitlement
	permissionName := "assigned"

	assigmentOptions := []entitlement.EntitlementOption{
		entitlement.WithGrantableTo(userResourceType),
		entitlement.WithDescription(resource.Description),
		entitlement.WithDisplayName(resource.DisplayName),
	}

	rv = append(rv, entitlement.NewPermissionEntitlement(resource, permissionName, assigmentOptions...))

	return rv, &rs.SyncOpResults{}, nil
}

func (r *roleBuilder) Grants(_ context.Context, _ *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Grant, *rs.SyncOpResults, error) {
	return nil, &rs.SyncOpResults{}, nil
}

func (r *roleBuilder) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)
	if principal.Id.ResourceType != userResourceType.Id {
		l.Warn("freshdesk-connector: only users can be granted with role membership",
			zap.String("principal_id", principal.Id.Resource),
			zap.String("principal_type", principal.Id.ResourceType))
		return nil, fmt.Errorf("freshdesk-connector: only users can be granted with role membership")
	}

	userID := principal.Id.Resource
	roleID, err := strconv.ParseInt(entitlement.Resource.Id.Resource, 10, 64)
	if err != nil {
		return nil, err
	}

	agent, _, err := r.client.GetAgentDetail(ctx, userID)
	if err != nil {
		return nil, err
	}

	if slices.Contains(agent.RoleIDs, roleID) {
		return annotations.New(&v2.GrantAlreadyExists{}), nil
	}

	agent.RoleIDs = append(agent.RoleIDs, roleID)

	anno, err := r.client.UpdateAgent(ctx, agent)
	if err != nil {
		return nil, err
	}

	return anno, nil
}

func (r *roleBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	userID := grant.Principal.Id.Resource
	roleID, err := ExtractRoleIDFromEntitlement(grant.Entitlement.Id)
	if err != nil {
		return nil, fmt.Errorf("freshdesk-connector: failed to parse role ID from entitlement: %w", err)
	}

	agent, _, err := r.client.GetAgentDetail(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("freshdesk-connector: failed to get agent: %w", err)
	}

	if !slices.Contains(agent.RoleIDs, roleID) {
		return annotations.New(&v2.GrantAlreadyRevoked{}), nil
	}

	var remainingRoles []int64
	for _, assignedRole := range agent.RoleIDs {
		if assignedRole != roleID {
			remainingRoles = append(remainingRoles, assignedRole)
		}
	}

	if len(remainingRoles) == 0 {
		return nil, fmt.Errorf("freshdesk-connector: cannot revoke last remaining role from agent")
	}

	agent.RoleIDs = remainingRoles

	anno, err := r.client.UpdateAgent(ctx, agent)
	if err != nil {
		return nil, fmt.Errorf("freshdesk-connector: failed to update agent roles: %w", err)
	}

	return anno, nil
}


func newRoleBuilder(c *client.FreshdeskClient) *roleBuilder {
	return &roleBuilder{
		resourceType: roleResourceType,
		client:       c,
	}
}

// This function parses a role from Freshdesk into a Role Resource.
func parseIntoRoleResource(_ context.Context, role *client.Role, _ *v2.ResourceId) (*v2.Resource, error) {
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

func ExtractRoleIDFromEntitlement(entitlementID string) (int64, error) {
	segments := strings.Split(entitlementID, ":")
	if len(segments) != 3 {
		return 0, fmt.Errorf("baton-freshdesk: invalid entitlement ID %s", entitlementID)
	}

	roleID, err := strconv.ParseInt(segments[1], 10, 64)
	if err != nil {
		return 0, err
	}

	return roleID, nil
}
