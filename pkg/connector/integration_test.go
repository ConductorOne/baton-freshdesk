package connector

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/conductorone/baton-freshdesk/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/stretchr/testify/assert"
)

var (
	ctx              = context.Background()
	domain, _        = os.LookupEnv("FRESHDESK_DOMAIN")
	apikey, _        = os.LookupEnv("FRESHDESK_TOKEN")
	parentResourceID = &v2.ResourceId{}
	attrs            = rs.SyncOpAttrs{
		Session:   nil,
		PageToken: pagination.Token{Size: 50, Token: ""},
	}
)

func TestUserBuilderList(t *testing.T) {
	if apikey == "" || domain == "" {
		message := fmt.Sprintf("params not found. apikey: %s - domain: %s", apikey, domain)
		t.Skip(message)
	}

	c, err := client.New(
		ctx,
		client.WithDomain(domain),
		client.WithBearerToken(apikey),
	)
	if err != nil {
		t.Errorf("ERROR: Failed to create client: %v", err)
	}

	u := newUserBuilder(c, true, true)
	res, _, err := u.List(ctx, parentResourceID, attrs)
	assert.Nil(t, err)
	assert.NotNil(t, res)

	message := fmt.Sprintf("Amount of users obtained: %d", len(res))
	t.Log(message)
}

func TestRoleBuilderList(t *testing.T) {
	if apikey == "" || domain == "" {
		message := fmt.Sprintf("params not founc. apikey: %s - domain: %s", apikey, domain)
		t.Skip(message)
	}

	c, err := client.New(
		ctx,
		client.WithDomain(domain),
		client.WithBearerToken(apikey),
	)
	if err != nil {
		t.Errorf("ERROR: Failed to create client: %v", err)
	}

	r := newRoleBuilder(c)

	res, _, err := r.List(ctx, parentResourceID, attrs)
	assert.Nil(t, err)
	assert.NotNil(t, res)

	message := fmt.Sprintf("Amount of roles obtained: %d", len(res))
	t.Log(message)
}

func TestGroupBuilderList(t *testing.T) {
	if apikey == "" || domain == "" {
		message := fmt.Sprintf("params not founc. apikey: %s - domain: %s", apikey, domain)
		t.Skip(message)
	}

	c, err := client.New(
		ctx,
		client.WithDomain(domain),
		client.WithBearerToken(apikey),
	)
	if err != nil {
		t.Errorf("ERROR: Failed to create client: %v", err)
	}

	g := newGroupBuilder(c)
	res, _, err := g.List(ctx, parentResourceID, attrs)
	assert.Nil(t, err)
	assert.NotNil(t, res)

	message := fmt.Sprintf("Amount of groups obtained: %d", len(res))
	t.Log(message)
}
