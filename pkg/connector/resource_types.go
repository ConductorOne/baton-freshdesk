package connector

import (
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
)

// The user resource type is for all user objects from the database.
var (
	userResourceType = &v2.ResourceType{
		Id:          "user",
		DisplayName: "User",
		Description: "The Agents are the users for Freshdesk",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_USER},
	}

	roleResourceType = &v2.ResourceType{
		Id:          "role",
		DisplayName: "Role",
		Description: "The Roles allow you to create special privileges and specify what an agent can see and do within your Freshdesk support portal",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_ROLE},
	}

	groupResourceType = &v2.ResourceType{
		Id:          "group",
		DisplayName: "Group",
		Description: "The Agents can be organized into different groups. It's useful for the organization of users.",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_GROUP},
	}
)
