package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/tomnomnom/linkheader"
)

// Endpoints available for Freshdesk APIs.
// https://developers.freshdesk.com/api/#intro .
const (
	baseURL = "https://.freshdesk.com"

	agentsEndpoint = "/api/v2/agents"
	groupsEndpoint = "/api/v2/groups"
	rolesEndpoint  = "/api/v2/roles"
)

type FreshdeskClient struct {
	httpClient   *uhttp.BaseHttpClient
	freshdeskURL string
	domain       string
	token        string
}

type Option func(client *FreshdeskClient)

func New(ctx context.Context, opts ...Option) (*FreshdeskClient, error) {
	freshdeskClient := &FreshdeskClient{
		httpClient:   &uhttp.BaseHttpClient{},
		freshdeskURL: baseURL,
		domain:       "",
		token:        "",
	}

	for _, o := range opts {
		o(freshdeskClient)
	}

	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	if err != nil {
		return nil, err
	}

	cli, err := uhttp.NewBaseHttpClientWithContext(context.Background(), httpClient)
	if err != nil {
		return nil, err
	}

	dotIndex := strings.Index(baseURL, ".")
	if dotIndex == -1 {
		return nil, fmt.Errorf("invalid URL: %s", baseURL)
	}

	fdURL := baseURL[:dotIndex] + freshdeskClient.domain + baseURL[dotIndex:]
	if !isValidUrl(fdURL) {
		return nil, fmt.Errorf("the URL: %s is not valid", fdURL)
	}

	freshdeskClient.freshdeskURL = fdURL
	freshdeskClient.httpClient = cli

	return freshdeskClient, nil
}

func WithBearerToken(apiToken string) Option {
	return func(c *FreshdeskClient) {
		c.token = apiToken
	}
}

func WithDomain(domain string) Option {
	return func(c *FreshdeskClient) {
		c.domain = domain
	}
}

func (f *FreshdeskClient) getToken() string {
	return f.token
}

func (f *FreshdeskClient) GetDomain() string {
	return f.domain
}

func isValidUrl(urlBase string) bool {
	u, err := url.Parse(urlBase)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// ListAgents Gets all the Agents from Freshdesk and deserialized them into an Array of Agents.
func (f *FreshdeskClient) ListAgents(ctx context.Context, opts PageOptions) ([]*Agent, string, annotations.Annotations, error) {
	queryUrl, err := url.JoinPath(f.freshdeskURL, agentsEndpoint)
	if err != nil {
		return nil, "", nil, err
	}

	var res []*Agent
	nextPage, annotation, err := f.getListFromAPI(ctx, queryUrl, &res, WithPage(opts.Page), WithPageLimit(opts.PerPage))
	if err != nil {
		return nil, "", nil, err
	}

	return res, nextPage, annotation, nil
}

// GetAgentDetail Gets all the Agents from Freshdesk and deserialized them into an Array of Agents.
func (f *FreshdeskClient) GetAgentDetail(ctx context.Context, agentID string) (*Agent, annotations.Annotations, error) {
	queryUrl, err := url.JoinPath(f.freshdeskURL, agentsEndpoint, agentID)
	if err != nil {
		return nil, nil, err
	}
	var res *Agent
	_, annotation, err := f.doRequest(ctx, http.MethodGet, queryUrl, &res, nil)
	if err != nil {
		return nil, nil, err
	}

	return res, annotation, nil
}

// CreateAgent creates a new agent in Freshdesk.
func (f *FreshdeskClient) CreateAgent(ctx context.Context, payload CreateAgentPayload) (*Agent, annotations.Annotations, error) {
	queryUrl, err := url.JoinPath(f.freshdeskURL, agentsEndpoint)
	if err != nil {
		return nil, nil, err
	}

	var res Agent
	_, annotation, err := f.doRequest(ctx, http.MethodPost, queryUrl, &res, payload)
	if err != nil {
		return nil, nil, err
	}

	return &res, annotation, nil
}

// getListFromAPI sends a request to the Freshdesk API to receive a JSON with a list of entities.
func (f *FreshdeskClient) getListFromAPI(
	ctx context.Context,
	urlAddress string,
	res any,
	reqOpt ...ReqOpt,
) (string, annotations.Annotations, error) {
	header, annotation, err := f.doRequest(ctx, http.MethodGet, urlAddress, &res, nil, reqOpt...)
	if err != nil {
		return "", nil, err
	}

	var pageToken string
	pagingLinks := linkheader.Parse(header.Get("Link"))
	for _, link := range pagingLinks {
		if link.Rel == "next" {
			nextPageUrl, err := url.Parse(link.URL)
			if err != nil {
				return "", nil, err
			}
			pageToken = nextPageUrl.Query().Get("page")
			break
		}
	}

	return pageToken, annotation, nil
}

func (f *FreshdeskClient) doRequest(
	ctx context.Context,
	method string,
	endpointUrl string,
	res interface{},
	body interface{},
	reqOptions ...ReqOpt,
) (http.Header, annotations.Annotations, error) {
	var (
		resp *http.Response
		err  error
	)
	rlDesc := &v2.RateLimitDescription{}
	urlAddress, err := url.Parse(endpointUrl)
	if err != nil {
		return nil, nil, err
	}
	for _, o := range reqOptions {
		o(urlAddress)
	}

	req, err := f.httpClient.NewRequest(
		ctx,
		method,
		urlAddress,
		uhttp.WithAcceptJSONHeader(),
		uhttp.WithContentTypeJSONHeader(),
		uhttp.WithHeader("Authorization", "Basic "+basicAuth(f.getToken(), "X")),
		uhttp.WithJSONBody(body),
	)
	if err != nil {
		return nil, nil, err
	}

	var fdErr freshdeskAPIError

	doOptions := []uhttp.DoOption{
		uhttp.WithRatelimitData(rlDesc),
		uhttp.WithErrorResponse(&fdErr),
	}
	if (method == http.MethodGet || method == http.MethodPut || method == http.MethodPost) && res != nil {
		doOptions = append(doOptions, uhttp.WithResponse(&res))
	}

	resp, err = f.httpClient.Do(req, doOptions...)
	if resp != nil {
		defer resp.Body.Close()
	}

	annotation := annotations.Annotations{}
	annotation.WithRateLimiting(rlDesc)

	if err != nil {
		return nil, annotation, err
	}

	return resp.Header, annotation, nil
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func (f *FreshdeskClient) ListRoles(ctx context.Context, opts PageOptions) ([]*Role, string, annotations.Annotations, error) {
	queryUrl, err := url.JoinPath(f.freshdeskURL, rolesEndpoint)
	if err != nil {
		return nil, "", nil, err
	}

	var res []*Role
	nextPage, annotation, err := f.getListFromAPI(ctx, queryUrl, &res, WithPage(opts.Page), WithPageLimit(opts.PerPage))
	if err != nil {
		return nil, "", nil, err
	}

	return res, nextPage, annotation, nil
}

func (f *FreshdeskClient) ListGroups(ctx context.Context, opts PageOptions) ([]*Group, string, annotations.Annotations, error) {
	queryUrl, err := url.JoinPath(f.freshdeskURL, groupsEndpoint)
	if err != nil {
		return nil, "", nil, err
	}

	var res []*Group
	nextPage, annotation, err := f.getListFromAPI(ctx, queryUrl, &res, WithPage(opts.Page), WithPageLimit(opts.PerPage))
	if err != nil {
		return nil, "", nil, err
	}

	return res, nextPage, annotation, nil
}

// UpdateGroup updates a group in Freshdesk.
func (f *FreshdeskClient) UpdateGroup(ctx context.Context, groupID string, payload UpdateGroupPayload) (*Group, annotations.Annotations, error) {
	queryUrl, err := url.JoinPath(f.freshdeskURL, groupsEndpoint, groupID)
	if err != nil {
		return nil, nil, err
	}

	var res Group
	_, annotation, err := f.doRequest(ctx, http.MethodPut, queryUrl, &res, payload)
	if err != nil {
		return nil, nil, err
	}

	return &res, annotation, nil
}

// DeleteAgent soft deletes an agent in Freshdesk.
func (f *FreshdeskClient) DeleteAgent(ctx context.Context, agentID string) (annotations.Annotations, error) {
	queryUrl, err := url.JoinPath(f.freshdeskURL, agentsEndpoint, agentID)
	if err != nil {
		return nil, err
	}

	_, annotation, err := f.doRequest(ctx, http.MethodDelete, queryUrl, nil, nil)
	if err != nil {
		return nil, err
	}

	return annotation, nil
}

// GetGroup gets a group from Freshdesk.
func (f *FreshdeskClient) GetGroup(ctx context.Context, groupID string) (*Group, annotations.Annotations, error) {
	queryUrl, err := url.JoinPath(f.freshdeskURL, groupsEndpoint, groupID)
	if err != nil {
		return nil, nil, err
	}

	var res Group
	_, annotation, err := f.doRequest(ctx, http.MethodGet, queryUrl, &res, nil)
	if err != nil {
		return nil, nil, err
	}

	return &res, annotation, nil
}

func (f *FreshdeskClient) UpdateAgent(ctx context.Context, agent *Agent) (annotations.Annotations, error) {
	agentID := strconv.FormatInt(agent.ID, 10)
	queryUrl, err := url.JoinPath(f.freshdeskURL, agentsEndpoint, agentID)
	if err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"role_ids": agent.RoleIDs,
	}

	_, anno, err := f.doRequest(ctx, http.MethodPut, queryUrl, nil, body)
	if err != nil {
		return nil, err
	}

	return anno, nil
}

// define error struct.
type freshdeskAPIError struct {
	Description string `json:"description"`
	Errors      []struct {
		Field   string `json:"field"`
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"errors"`
}

func (e freshdeskAPIError) Message() string {
	return e.Description
}
