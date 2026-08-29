package connector

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/conductorone/baton-freshdesk/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/types/entitlement"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.uber.org/zap"
)

type groupBuilder struct {
	resourceType *v2.ResourceType
	client       *client.FreshdeskClient
}

func (g *groupBuilder) ResourceType(_ context.Context) *v2.ResourceType {
	return g.resourceType
}

func (g *groupBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, attrs rs.SyncOpAttrs) ([]*v2.Resource, *rs.SyncOpResults, error) {
	var rv []*v2.Resource
	bag, pageToken, err := getToken(&attrs.PageToken, groupResourceType)
	if err != nil {
		return nil, nil, err
	}

	groups, nextPageToken, annotation, err := g.client.ListGroups(ctx, client.PageOptions{
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

	for _, group := range groups {
		userResource, err := parseIntoGroupResource(ctx, group, parentResourceID)
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

func (g *groupBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Entitlement, *rs.SyncOpResults, error) {
	var rv []*v2.Entitlement
	const permissionName = "member"

	assigmentOptions := []entitlement.EntitlementOption{
		entitlement.WithGrantableTo(userResourceType),
		entitlement.WithDescription(resource.Description),
		entitlement.WithDisplayName(resource.DisplayName),
	}

	rv = append(rv, entitlement.NewPermissionEntitlement(resource, permissionName, assigmentOptions...))

	return rv, &rs.SyncOpResults{}, nil
}

// Grants are emitted from userBuilder.Grants() in users.go instead.
func (g *groupBuilder) Grants(_ context.Context, _ *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Grant, *rs.SyncOpResults, error) {
	return nil, &rs.SyncOpResults{}, nil
}

func (g *groupBuilder) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	if principal.Id.ResourceType != userResourceType.Id {
		l.Warn(
			"baton-freshdesk: only users can be granted group membership",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, status.Error(codes.InvalidArgument, "baton-freshdesk: only users can be granted group membership")
	}

	agentID, err := strconv.ParseInt(principal.Id.Resource, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("freshdesk-connector: failed to parse agent ID: %w", err)
	}
	groupID := entitlement.Resource.Id.Resource

	group, _, err := g.client.GetGroup(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("freshdesk-connector: failed to get group details: %w", err)
	}

	if slices.Contains(group.AgentIDs, agentID) {
		return annotations.New(&v2.GrantAlreadyExists{}), nil
	}

	group.AgentIDs = append(group.AgentIDs, agentID)

	payload := client.UpdateGroupPayload{
		AgentIDs: group.AgentIDs,
	}
	_, annos, err := g.client.UpdateGroup(ctx, groupID, payload)
	if err != nil {
		return nil, fmt.Errorf("freshdesk-connector: failed to update group members: %w", err)
	}

	return annos, nil
}

func (g *groupBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	entitlement := grant.Entitlement
	principal := grant.Principal

	if principal.Id.ResourceType != userResourceType.Id {
		l.Warn(
			"baton-freshdesk: only users can have group membership revoked",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, status.Error(codes.InvalidArgument, "baton-freshdesk: only users can have group membership revoked")
	}

	agentID, err := strconv.ParseInt(principal.Id.Resource, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("freshdesk-connector: failed to parse agent ID: %w", err)
	}
	groupID := entitlement.Resource.Id.Resource

	group, _, err := g.client.GetGroup(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("freshdesk-connector: failed to get group details: %w", err)
	}

	if !slices.Contains(group.AgentIDs, agentID) {
		return annotations.New(&v2.GrantAlreadyRevoked{}), nil
	}

	agentIDs := make([]int64, 0, len(group.AgentIDs))
	for _, id := range group.AgentIDs {
		if id != agentID {
			agentIDs = append(agentIDs, id)
		}
	}

	payload := client.UpdateGroupPayload{
		AgentIDs: agentIDs,
	}
	_, annos, err := g.client.UpdateGroup(ctx, groupID, payload)
	if err != nil {
		return nil, fmt.Errorf("freshdesk-connector: failed to update group members: %w", err)
	}

	return annos, nil
}

func newGroupBuilder(c *client.FreshdeskClient) *groupBuilder {
	return &groupBuilder{
		resourceType: groupResourceType,
		client:       c,
	}
}

// This function parses a group from Freshdesk into a Group Resource.
func parseIntoGroupResource(_ context.Context, group *client.Group, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"group_id":   group.ID,
		"group_name": group.Name,
	}

	groupTraits := []rs.GroupTraitOption{}

	ret, err := rs.NewGroupResource(
		group.Name,
		groupResourceType,
		group.ID,
		groupTraits,
		rs.WithResourceProfile(profile),
		rs.WithParentResourceID(parentResourceID),
	)
	if err != nil {
		return nil, err
	}

	return ret, nil
}
