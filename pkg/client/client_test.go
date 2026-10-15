package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const testToken = "test-api-key"

type stubResponse struct {
	status      int
	contentType string
	body        string
	headers     map[string]string
}

func newTestClient(t *testing.T, stub stubResponse) *FreshdeskClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		for k, v := range stub.headers {
			w.Header().Set(k, v)
		}
		if stub.contentType != "" {
			w.Header().Set("Content-Type", stub.contentType)
		}
		w.WriteHeader(stub.status)
		_, _ = w.Write([]byte(stub.body))
	}))
	t.Cleanup(srv.Close)

	c, err := New(context.Background(), WithBaseURL(srv.URL), WithBearerToken(testToken))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func assertContains(t *testing.T, got string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("%q does not contain %q", got, w)
		}
	}
}

func TestListErrors(t *testing.T) {
	const (
		documentedBody = `{"description":"Validation failed","errors":[{"field":"email","message":"It should be a valid email","code":"invalid_value"}]}`
		deniedBody     = `{"errors":[{"message":"You are not authorized to perform this action.","code":"access_denied"}]}`
		authBody       = `{"description":"Authentication failure","errors":[{"message":"Incorrect or missing API credentials.","code":"invalid_credentials"}]}`
	)

	cases := []struct {
		name     string
		stub     stubResponse
		wantCode codes.Code
		want     []string
	}{
		{
			name:     "403 documented body",
			stub:     stubResponse{status: http.StatusForbidden, contentType: "application/json", body: documentedBody},
			wantCode: codes.PermissionDenied,
			want:     []string{"Validation failed", "email: It should be a valid email (invalid_value)"},
		},
		{
			name:     "403 errors without description",
			stub:     stubResponse{status: http.StatusForbidden, contentType: "application/json", body: deniedBody},
			wantCode: codes.PermissionDenied,
			want:     []string{"You are not authorized to perform this action. (access_denied)"},
		},
		{
			name:     "403 empty body",
			stub:     stubResponse{status: http.StatusForbidden, contentType: "application/json", body: `{}`},
			wantCode: codes.PermissionDenied,
			want:     []string{"no error detail in response body"},
		},
		{
			name:     "403 html body",
			stub:     stubResponse{status: http.StatusForbidden, contentType: "text/html", body: "<html>Forbidden</html>"},
			wantCode: codes.PermissionDenied,
			want:     []string{"unexpected content type"},
		},
		{
			name:     "401 invalid credentials",
			stub:     stubResponse{status: http.StatusUnauthorized, contentType: "application/json", body: authBody},
			wantCode: codes.Unauthenticated,
			want:     []string{"Authentication failure", "invalid_credentials"},
		},
		{
			name: "403 with request id",
			stub: stubResponse{
				status: http.StatusForbidden, contentType: "application/json", body: deniedBody,
				headers: map[string]string{"X-Request-Id": "abc123"},
			},
			wantCode: codes.PermissionDenied,
			want:     []string{"x-request-id=abc123"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestClient(t, tc.stub)
			calls := map[string]func() error{
				"roles": func() error {
					_, _, _, err := c.ListRoles(context.Background(), PageOptions{})
					return err
				},
				"groups": func() error {
					_, _, _, err := c.ListGroups(context.Background(), PageOptions{})
					return err
				},
				"agents": func() error {
					_, _, _, err := c.ListAgents(context.Background(), PageOptions{})
					return err
				},
			}
			for name, call := range calls {
				err := call()
				if err == nil {
					t.Fatalf("%s: expected error", name)
				}
				if got := status.Code(err); got != tc.wantCode {
					t.Errorf("%s: status code = %v, want %v", name, got, tc.wantCode)
				}
				assertContains(t, err.Error(), tc.want...)
				assertContains(t, err.Error(), "baton-freshdesk: GET /api/v2/"+name)
				if strings.Contains(err.Error(), testToken) || strings.Contains(err.Error(), "Authorization") {
					t.Errorf("%s: error leaks credentials: %v", name, err)
				}
			}
		})
	}
}

func TestListRolesSuccess(t *testing.T) {
	c := newTestClient(t, stubResponse{
		status:      http.StatusOK,
		contentType: "application/json",
		body:        `[{"id":1,"name":"Administrator"},{"id":2,"name":"Agent"}]`,
		headers:     map[string]string{"Link": `<https://example.freshdesk.com/api/v2/roles?page=2>; rel="next"`},
	})

	roles, next, _, err := c.ListRoles(context.Background(), PageOptions{})
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	if len(roles) != 2 || roles[0].ID != 1 || roles[1].Name != "Agent" {
		t.Errorf("unexpected roles: %+v", roles)
	}
	if next != "2" {
		t.Errorf("next page = %q, want %q", next, "2")
	}
}

func TestListRolesRateLimited(t *testing.T) {
	c := newTestClient(t, stubResponse{
		status:      http.StatusTooManyRequests,
		contentType: "application/json",
		body:        `{}`,
		headers:     map[string]string{"Retry-After": "30", "X-Ratelimit-Total": "100", "X-Ratelimit-Remaining": "0"},
	})

	var res []*Role
	_, annos, err := c.doRequest(context.Background(), http.MethodGet, c.freshdeskURL+rolesEndpoint, &res, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	rl := &v2.RateLimitDescription{}
	ok, annoErr := annos.Pick(rl)
	if annoErr != nil || !ok {
		t.Fatalf("missing rate limit annotation: %v", annoErr)
	}
	if rl.GetStatus() != v2.RateLimitDescription_STATUS_OVERLIMIT {
		t.Errorf("rate limit status = %v, want OVERLIMIT", rl.GetStatus())
	}
}

func TestFreshdeskAPIErrorMessage(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		want    []string
		exactly string
	}{
		{
			name: "documented body",
			body: `{"description":"Validation failed","errors":[{"field":"email","message":"It should be a valid email","code":"invalid_value"}]}`,
			want: []string{"Validation failed", "email: It should be a valid email (invalid_value)"},
		},
		{
			name:    "errors without description",
			body:    `{"description":"","errors":[{"message":"Feature not enabled","code":"require_feature"}]}`,
			exactly: "Feature not enabled (require_feature)",
		},
		{
			name:    "empty body",
			body:    `{}`,
			exactly: "no error detail in response body",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var e freshdeskAPIError
			if err := json.Unmarshal([]byte(tc.body), &e); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			got := e.Message()
			if got == "" || strings.HasPrefix(got, ": ") {
				t.Fatalf("bad message %q", got)
			}
			if tc.exactly != "" && got != tc.exactly {
				t.Errorf("Message() = %q, want %q", got, tc.exactly)
			}
			assertContains(t, got, tc.want...)
		})
	}
}
