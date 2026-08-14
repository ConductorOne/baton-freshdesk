package connector

import (
	"context"
	"testing"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"google.golang.org/protobuf/proto"
)

func hasAnno(rt *v2.ResourceType, msg proto.Message) bool {
	for _, a := range rt.GetAnnotations() {
		if a.MessageIs(msg) {
			return true
		}
	}
	return false
}

// The user type's only grants are cross-type role and group grants. Escalate to
// SkipEntitlementsAndGrants only when BOTH targets are excluded — with either
// one still synced the grants pass must run so that target's grants are emitted.
func TestUserResourceType_SkipAnnotation(t *testing.T) {
	cases := []struct {
		name                string
		skipRole, skipGroup bool
		wantSkipBoth        bool
	}{
		{"both synced", false, false, false},
		{"role filtered", true, false, false},
		{"group filtered", false, true, false},
		{"both filtered", true, true, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rt := newUserBuilder(nil, tc.skipRole, tc.skipGroup).ResourceType(context.Background())
			if hasAnno(rt, &v2.SkipEntitlementsAndGrants{}) != tc.wantSkipBoth {
				t.Fatalf("SkipEntitlementsAndGrants = %v, want %v", !tc.wantSkipBoth, tc.wantSkipBoth)
			}
			if hasAnno(rt, &v2.SkipEntitlements{}) == tc.wantSkipBoth {
				t.Fatalf("SkipEntitlements presence wrong for %s", tc.name)
			}
		})
	}

	// Both branches annotate, so a dropped proto.Clone would leak either one
	// onto the shared package-level value.
	if hasAnno(userResourceType, &v2.SkipEntitlementsAndGrants{}) || hasAnno(userResourceType, &v2.SkipEntitlements{}) {
		t.Fatal("package-level userResourceType was mutated")
	}
}

// A zero-value Connector{} is used to generate the capability set, bypassing
// New; it must report the unfiltered capabilities.
func TestZeroValueConnector_DoesNotSkipGrants(t *testing.T) {
	var found bool
	for _, s := range (&Connector{}).ResourceSyncers(context.Background()) {
		rt := s.ResourceType(context.Background())
		if rt.GetId() != userResourceType.Id {
			continue
		}
		found = true
		if hasAnno(rt, &v2.SkipEntitlementsAndGrants{}) {
			t.Fatal("zero-value Connector advertised SkipEntitlementsAndGrants")
		}
	}
	if !found {
		t.Fatal("user syncer missing from ResourceSyncers; nothing was asserted")
	}
}
