package connector

import (
	"context"
	"testing"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/stretchr/testify/assert"
)

// TestUserBuilderResourceType_WillSyncResourceType asserts that userBuilder.ResourceType()
// annotates the (cloned) user resource type based on whether the cross-type grant targets
// emitted from Grants() (role, group) are actually being synced: SkipEntitlements when at
// least one target is synced (Grants() must still run), SkipEntitlementsAndGrants when
// neither is (Grants() need not run at all).
func TestUserBuilderResourceType_WillSyncResourceType(t *testing.T) {
	cases := []struct {
		name           string
		rolesExcluded  bool
		groupsExcluded bool
		wantSkip       string // "entitlements" or "entitlements_and_grants"
	}{
		{name: "both synced", rolesExcluded: false, groupsExcluded: false, wantSkip: "entitlements"},
		{name: "role filtered out only", rolesExcluded: true, groupsExcluded: false, wantSkip: "entitlements"},
		{name: "group filtered out only", rolesExcluded: false, groupsExcluded: true, wantSkip: "entitlements"},
		{name: "both filtered out", rolesExcluded: true, groupsExcluded: true, wantSkip: "entitlements_and_grants"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			u := newUserBuilder(nil, tt.rolesExcluded, tt.groupsExcluded)

			rt := u.ResourceType(context.Background())
			anno := annotations.Annotations(rt.GetAnnotations())

			switch tt.wantSkip {
			case "entitlements":
				assert.True(t, anno.Contains(&v2.SkipEntitlements{}), "expected SkipEntitlements")
				assert.False(t, anno.Contains(&v2.SkipEntitlementsAndGrants{}), "did not expect SkipEntitlementsAndGrants")
			case "entitlements_and_grants":
				assert.True(t, anno.Contains(&v2.SkipEntitlementsAndGrants{}), "expected SkipEntitlementsAndGrants")
			}

			// The package-level resource type must never be mutated by ResourceType().
			assert.Empty(t, userResourceType.GetAnnotations(), "package-level userResourceType must not be mutated")
		})
	}
}

// TestUserBuilderResourceType_ZeroValueDefaultsToSyncAll guards against a regression where the
// zero-value userBuilder (as constructed by connectorrunner.WithDefaultCapabilitiesConnectorBuilderV2,
// which builds a bare &Connector{} for the `capabilities` CLI command and bypasses New()/opts
// entirely) would report the connector as excluding role/group sync by default. The zero value of
// rolesExcluded/groupsExcluded must mean "included" so the generated capabilities metadata matches
// the actual default (sync everything) behavior.
func TestUserBuilderResourceType_ZeroValueDefaultsToSyncAll(t *testing.T) {
	var u userBuilder
	u.resourceType = userResourceType

	rt := u.ResourceType(context.Background())
	anno := annotations.Annotations(rt.GetAnnotations())

	assert.True(t, anno.Contains(&v2.SkipEntitlements{}), "zero-value userBuilder should behave as sync-all (SkipEntitlements)")
	assert.False(t, anno.Contains(&v2.SkipEntitlementsAndGrants{}), "zero-value userBuilder must not report SkipEntitlementsAndGrants")
}
