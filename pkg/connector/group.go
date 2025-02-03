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

	for _, group := range *groups {
		groupCopy := group
		userResource, err := parseIntoGroupResource(ctx, &groupCopy, parentResourceID)
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

		if slices.Contains(agentDetail.GroupIDs, value) {
			userResource, _ := parseIntoUserResource(&agentDetail, nil)

			membershipGrant := grant.NewGrant(resource, permissionName, userResource.Id)
			rv = append(rv, membershipGrant)
		}

	}
	return rv, "", nil, nil
}

func newGroupBuilder(c *client.FreshdeskClient) *groupBuilder {
	return &groupBuilder{
		resourceType: groupResourceType,
		client:       c,
	}
}

// This function parses a group from Freshdesk into a Group Resource
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

	if g.agentsDetails != nil && len(g.agentsDetails) > 0 {
		return nil
	}

	paginationToken := pagination.Token{50, ""}
	IDs, err := g.GetAllAgentsIDs(ctx, &paginationToken)
	if err != nil {
		return err
	}

	if len(IDs) == 0 {
		return errors.New("no agents found")
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

func (g *groupBuilder) GetAllAgentsIDs(ctx context.Context, pToken *pagination.Token) ([]string, error) {
	var rv []string

	for {
		bag, pageToken, err := getToken(pToken, userResourceType)

		agents, nextPageToken, _, err := g.client.ListAgents(ctx, client.PageOptions{
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
