package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/tomnomnom/linkheader"
)

// Endpoints available for Freshdesk APIs.
const (
	baseURL = "https://.freshdesk.com"

	// GET endpoints.
	allAgents = "/api/v2/agents"
	allGrous  = "/api/v2/groups"
	allRoles  = "/api/v2/roles"

	getAgentDetail = "/api/v2/agents" // Must indicate the agent ID: /[id].

	// PUT endpoints.
	updateAgent = "/api/v2/agents" // Must indicate the agent ID: /[id].
)

type FreshdeskClient struct {
	httpClient   *uhttp.BaseHttpClient
	freshdeskURL string
	domain       string
	token        string
}

func New(ctx context.Context, freshdeskClient *FreshdeskClient) (*FreshdeskClient, error) {
	var (
		clientToken  = freshdeskClient.getToken()
		clientDomain = freshdeskClient.GetDomain()
	)

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

	fdClient := FreshdeskClient{
		httpClient:   cli,
		freshdeskURL: fdURL,
		domain:       clientDomain,
		token:        clientToken,
	}

	return &fdClient, nil
}

func NewClient() *FreshdeskClient {
	return &FreshdeskClient{
		httpClient:   &uhttp.BaseHttpClient{},
		freshdeskURL: baseURL,
		domain:       "",
		token:        "",
	}
}

func (f *FreshdeskClient) WithBearerToken(apiToken string) *FreshdeskClient {
	f.token = apiToken
	return f
}

func (f *FreshdeskClient) WithDomain(domain string) *FreshdeskClient {
	f.domain = domain
	return f
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
func (f *FreshdeskClient) ListAgents(ctx context.Context, opts PageOptions) (*[]Agent, string, annotations.Annotations, error) {
	queryUrl, err := url.JoinPath(f.freshdeskURL, allAgents)
	if err != nil {
		return nil, "", nil, err
	}

	var res *[]Agent
	nextPage, annotation, err := f.getListFromAPI(ctx, queryUrl, &res, WithPage(opts.Page), WithPageLimit(opts.PerPage))
	if err != nil {
		return nil, "", nil, err
	}

	return res, nextPage, annotation, nil
}

// GetAgentDetail Gets all the Agents from Freshdesk and deserialized them into an Array of Agents.
func (f *FreshdeskClient) GetAgentDetail(ctx context.Context, agentID string) (*Agent, annotations.Annotations, error) {
	queryUrl, err := url.JoinPath(f.freshdeskURL, getAgentDetail, agentID)
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

	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPost:
		doOptions := []uhttp.DoOption{}
		if res != nil {
			doOptions = append(doOptions, uhttp.WithResponse(&res))
		}
		resp, err = f.httpClient.Do(req, doOptions...)
		if resp != nil {
			defer resp.Body.Close()
		}
	case http.MethodDelete:
		resp, err = f.httpClient.Do(req)
		if resp != nil {
			defer resp.Body.Close()
		}
	}

	if err != nil {
		return nil, nil, err
	}

	annotation := annotations.Annotations{}

	return resp.Header, annotation, nil
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func (f *FreshdeskClient) ListRoles(ctx context.Context, opts PageOptions) (*[]Role, string, annotations.Annotations, error) {
	queryUrl, err := url.JoinPath(f.freshdeskURL, allRoles)
	if err != nil {
		return nil, "", nil, err
	}

	var res *[]Role
	nextPage, annotation, err := f.getListFromAPI(ctx, queryUrl, &res, WithPage(opts.Page), WithPageLimit(opts.PerPage))
	if err != nil {
		return nil, "", nil, err
	}

	return res, nextPage, annotation, nil
}

func (f *FreshdeskClient) ListGroups(ctx context.Context, opts PageOptions) (*[]Group, string, annotations.Annotations, error) {
	queryUrl, err := url.JoinPath(f.freshdeskURL, allGrous)
	if err != nil {
		return nil, "", nil, err
	}

	var res *[]Group
	nextPage, annotation, err := f.getListFromAPI(ctx, queryUrl, &res, WithPage(opts.Page), WithPageLimit(opts.PerPage))
	if err != nil {
		return nil, "", nil, err
	}

	return res, nextPage, annotation, nil
}

func (f *FreshdeskClient) UpdateAgent(ctx context.Context, agent *Agent) (annotations.Annotations, error) {
	agentID := strconv.FormatInt(agent.ID, 10)
	queryUrl, err := url.JoinPath(f.freshdeskURL, updateAgent, "/", agentID)
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
