package connector

import (
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
)

// Exported so callers (e.g. cmd/main.go, tests) can reference the target
// resource type IDs for cross-type grant filtering without hardcoding strings.
const (
	UserResourceTypeID  = "user"
	RoleResourceTypeID  = "role"
	GroupResourceTypeID = "group"
)

// The user resource type is for all user objects from the database.
var (
	userResourceType = &v2.ResourceType{
		Id:          UserResourceTypeID,
		DisplayName: "User",
		Description: "The Agents are the users for Freshdesk",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_USER},
	}

	roleResourceType = &v2.ResourceType{
		Id:          RoleResourceTypeID,
		DisplayName: "Role",
		Description: "The Roles allow you to create special privileges and specify what an agent can see and do within your Freshdesk support portal",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_ROLE},
	}

	groupResourceType = &v2.ResourceType{
		Id:          GroupResourceTypeID,
		DisplayName: "Group",
		Description: "The Agents can be organized into different groups. It's useful for the organization of users.",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_GROUP},
	}
)
