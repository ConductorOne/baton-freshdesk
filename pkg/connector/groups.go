package connector

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"sync"

	"github.com/conductorone/baton-freshdesk/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.uber.org/zap"
)

type groupBuilder struct {
	resourceType     *v2.ResourceType
	client           *client.FreshdeskClient
	agentsDetails    []client.Agent
	agentDetailMutex sync.RWMutex
}

func (g *groupBuilder) ResourceType(_ context.Context) *v2.ResourceType {
	return g.resourceType
}

func (g *groupBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var rv []*v2.Resource
	bag, pageToken, err := getToken(pToken, roleResourceType)
	if err != nil {
		return nil, "", nil, err
	}

	groups, nextPageToken, annotation, err := g.client.ListGroups(ctx, client.PageOptions{
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

	for _, group := range groups {
		userResource, err := parseIntoGroupResource(ctx, group, parentResourceID)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, userResource)
	}

	nextPageToken, err = bag.Marshal()
	if err != nil {
		return nil, "", nil, err
	}

	return rv, nextPageToken, annotation, nil
}

func (g *groupBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement
	const permissionName = "member"

	assigmentOptions := []entitlement.EntitlementOption{
		entitlement.WithGrantableTo(userResourceType),
		entitlement.WithDescription(resource.Description),
		entitlement.WithDisplayName(resource.DisplayName),
	}

	rv = append(rv, entitlement.NewPermissionEntitlement(resource, permissionName, assigmentOptions...))

	return rv, "", nil, nil
}

func (g *groupBuilder) Grants(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var rv []*v2.Grant
	err := g.GetAgentsDetails(ctx)
	if err != nil {
		return nil, "", nil, err
	}

	for _, agentDetail := range g.agentsDetails {
		const permissionName = "member"

		value, err := strconv.Atoi(resource.Id.Resource)
		if err != nil {
			return nil, "", nil, err
		}

		if slices.Contains(agentDetail.GroupIDs, int64(value)) {
			userResource, _ := parseIntoUserResource(&agentDetail, nil)

			membershipGrant := grant.NewGrant(resource, permissionName, userResource.Id)
			rv = append(rv, membershipGrant)
		}
	}
	return rv, "", nil, nil
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
		return nil, status.Errorf(codes.Internal, "failed to parse agent ID: %v", err)
	}
	groupID := entitlement.Resource.Id.Resource

	group, _, err := g.client.GetGroup(ctx, groupID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get group details: %v", err)
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
		return nil, status.Errorf(codes.Internal, "failed to update group members: %v", err)
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
		return nil, status.Errorf(codes.Internal, "failed to parse agent ID: %v", err)
	}
	groupID := entitlement.Resource.Id.Resource

	group, _, err := g.client.GetGroup(ctx, groupID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get group details: %v", err)
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
		return nil, status.Errorf(codes.Internal, "failed to update group members: %v", err)
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

	groupTraits := []rs.GroupTraitOption{
		rs.WithGroupProfile(profile),
	}

	ret, err := rs.NewGroupResource(
		group.Name,
		groupResourceType,
		group.ID,
		groupTraits,
		rs.WithParentResourceID(parentResourceID),
	)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (g *groupBuilder) GetAgentsDetails(ctx context.Context) error {
	g.agentDetailMutex.Lock()
	defer g.agentDetailMutex.Unlock()

	if g.agentsDetails != nil || len(g.agentsDetails) > 0 {
		return nil
	}

	IDs, err := g.GetAllAgentsIDs(ctx)
	if err != nil {
		return err
	}

	if len(IDs) == 0 {
		return fmt.Errorf("no agents found")
	}

	for _, id := range IDs {
		agentDetail, _, err := g.client.GetAgentDetail(ctx, id)
		if err != nil {
			return err
		}

		g.agentsDetails = append(g.agentsDetails, *agentDetail)
	}

	return nil
}

func (g *groupBuilder) GetAllAgentsIDs(ctx context.Context) ([]string, error) {
	var rv []string
	paginationToken := pagination.Token{Size: 50, Token: ""}

	for {
		bag, pageToken, err := getToken(&paginationToken, userResourceType)
		if err != nil {
			return nil, err
		}

		agents, nextPageToken, _, err := g.client.ListAgents(ctx, client.PageOptions{
			Page:    pageToken,
			PerPage: paginationToken.Size,
		})
		if err != nil {
			return nil, err
		}

		err = bag.Next(nextPageToken)
		if err != nil {
			return nil, err
		}

		for _, agent := range agents {
			agentID := strconv.FormatInt(agent.ID, 10)

			rv = append(rv, agentID)
		}

		nextPageToken, err = bag.Marshal()
		if err != nil {
			return nil, err
		}

		if nextPageToken == "" {
			break
		}
		paginationToken.Token = nextPageToken
	}

	return rv, nil
}
