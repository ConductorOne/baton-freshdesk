package connector

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/conductorone/baton-freshdesk/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestFreshdeskClient spins up an httptest.Server that returns a fixed agent detail
// payload for GET /api/v2/agents/{id}, and returns a client pointed at it.
func newTestFreshdeskClient(t *testing.T, agent client.Agent) (*client.FreshdeskClient, func()) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(agent)
		require.NoError(t, err)
	}))

	c, err := client.New(
		context.Background(),
		client.WithDomain("test"),
		client.WithBearerToken("token"),
		client.WithBaseURL(srv.URL),
	)
	require.NoError(t, err)

	return c, srv.Close
}

// TestUserBuilderGrants_WillSyncResourceType asserts that cross-type role/group grant emission
// in userBuilder.Grants() is gated on whether the target resource type is actually being synced:
// when a target type is filtered out of the sync, no grants for that type should be emitted.
func TestUserBuilderGrants_WillSyncResourceType(t *testing.T) {
	agent := client.Agent{
		ID:       1,
		RoleIDs:  []int64{10, 11},
		GroupIDs: []int64{20},
	}

	principal := &v2.Resource{Id: &v2.ResourceId{ResourceType: UserResourceTypeID, Resource: "1"}}

	tests := []struct {
		name       string
		syncRoles  bool
		syncGroups bool
		wantRole   bool
		wantGroup  bool
	}{
		{name: "both synced", syncRoles: true, syncGroups: true, wantRole: true, wantGroup: true},
		{name: "role filtered out", syncRoles: false, syncGroups: true, wantRole: false, wantGroup: true},
		{name: "group filtered out", syncRoles: true, syncGroups: false, wantRole: true, wantGroup: false},
		{name: "both filtered out", syncRoles: false, syncGroups: false, wantRole: false, wantGroup: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, closeFn := newTestFreshdeskClient(t, agent)
			defer closeFn()

			u := newUserBuilder(c, tt.syncRoles, tt.syncGroups)

			grants, _, err := u.Grants(context.Background(), principal, rs.SyncOpAttrs{})
			assert.NoError(t, err)

			var gotRole, gotGroup bool
			for _, g := range grants {
				switch g.Entitlement.Resource.Id.ResourceType {
				case RoleResourceTypeID:
					gotRole = true
				case GroupResourceTypeID:
					gotGroup = true
				}
			}

			assert.Equal(t, tt.wantRole, gotRole, "role grant emission")
			assert.Equal(t, tt.wantGroup, gotGroup, "group grant emission")

			if !tt.syncRoles && !tt.syncGroups {
				assert.Nil(t, grants, "no HTTP call/grants should be produced when both targets are filtered out")
			}
		})
	}
}
