package lab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Convention V10 Lab API Gedix:
// - api_client.go contains only the generic HTTP client plumbing.
// - api_token.go contains only API token management.
// - api_actions_<domain>.go contains Gedix business actions by domain.
// - actions.go remains the action catalog exposed to the Plan d'actions.
// To add an API action: create/complete api_actions_<domain>.go, add Execute..., then reference it in actions.go.
const (
	defaultAPIAuthHeader  = "Authorization"
	defaultAPIAuthPrefix  = "Bearer"
	defaultAPIContentType = "application/json; charset=UTF-8"
)

type GedixAPIClient struct {
	httpClient *http.Client
	baseURL    string
	token      string
	writer     io.Writer
}

type GedixAPIRequest struct {
	Name              string
	Method            string
	Path              string
	Query             map[string]string
	Headers           map[string]string
	Body              any
	BodyJSONParam     string
	ExpectedStatuses  []int
	PrintResponseBody bool
}

func NewGedixAPIClient(config Config, writer io.Writer) (*GedixAPIClient, error) {
	token, err := LoadAPIToken(config.Name)
	if err != nil {
		return nil, err
	}
	if token == "" {
		return nil, fmt.Errorf("Token API non renseigné pour cette maquette.")
	}
	baseURL, err := gedixAPIBaseURL(config)
	if err != nil {
		return nil, err
	}
	if writer == nil {
		writer = io.Discard
	}
	return &GedixAPIClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
		token:      token,
		writer:     writer,
	}, nil
}

func ExecuteGedixAPIRequests(requests []GedixAPIRequest) ActionExecute {
	return func(ctx ActionContext, params map[string]any) error {
		return executeGedixAPIRequests(ctx, params, requests)
	}
}

func executeGedixAPIRequests(ctx ActionContext, params map[string]any, requests []GedixAPIRequest) error {
	client, err := NewGedixAPIClient(ctx.Config, ctx.Writer)
	if err != nil {
		return err
	}
	for _, request := range requests {
		body, err := gedixAPIRequestBodyValue(request, params)
		if err != nil {
			return err
		}
		request.Body = body
		if err := client.DoJSON(request); err != nil {
			return err
		}
	}
	return nil
}

func (c *GedixAPIClient) PostJSON(apiPath string, body any, expectedStatuses ...int) error {
	return c.DoJSON(GedixAPIRequest{
		Method:           http.MethodPost,
		Path:             apiPath,
		Body:             body,
		ExpectedStatuses: expectedStatuses,
	})
}

func (c *GedixAPIClient) DoJSON(apiRequest GedixAPIRequest) error {
	return executeGedixAPIRequest(c.httpClient, c.writer, c.baseURL, c.token, apiRequest)
}

func executeGedixAPIRequest(client *http.Client, writer io.Writer, baseURL string, token string, apiRequest GedixAPIRequest) error {
	method := strings.ToUpper(strings.TrimSpace(apiRequest.Method))
	if method == "" {
		method = http.MethodGet
	}
	targetURL, err := gedixAPIRequestURL(baseURL, apiRequest.Path, apiRequest.Query)
	if err != nil {
		return err
	}
	body, err := gedixAPIRequestBody(apiRequest.Body)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(apiRequest.Name)
	if name != "" {
		fmt.Fprintf(writer, "[API] %s\n", name)
	}
	fmt.Fprintf(writer, "[API] %s %s\n", method, gedixAPIRequestLogPath(targetURL, apiRequest))
	request, err := http.NewRequest(method, targetURL, body)
	if err != nil {
		return err
	}
	request.Header.Set(defaultAPIAuthHeader, strings.TrimSpace(defaultAPIAuthPrefix+" "+token))
	request.Header.Set("Content-Type", defaultAPIContentType)
	for key, value := range apiRequest.Headers {
		if strings.TrimSpace(key) != "" {
			request.Header.Set(key, value)
		}
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return err
	}
	safeBody := redactToken(string(responseBody), token)
	fmt.Fprintf(writer, "[API] Status HTTP : %d\n", response.StatusCode)
	if apiRequest.PrintResponseBody && strings.TrimSpace(safeBody) != "" {
		fmt.Fprintf(writer, "[API] Réponse :\n%s\n", safeBody)
	}
	if !expectedHTTPStatus(response.StatusCode, apiRequest.ExpectedStatuses) {
		if strings.TrimSpace(safeBody) != "" {
			fmt.Fprintf(writer, "[API] Erreur Gedix : %s\n", safeBody)
		}
		if strings.TrimSpace(safeBody) == "" {
			return fmt.Errorf("requête API Gedix échouée: status HTTP %d", response.StatusCode)
		}
		return fmt.Errorf("requête API Gedix échouée: status HTTP %d: %s", response.StatusCode, safeBody)
	}
	return nil
}

func (r GedixAPIRequest) LogPath() string {
	value := strings.TrimSpace(r.Path)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err == nil && parsed.IsAbs() {
		return parsed.Redacted()
	}
	return value
}

func gedixAPIRequestLogPath(targetURL string, apiRequest GedixAPIRequest) string {
	parsed, err := url.Parse(targetURL)
	if err != nil || parsed.Path == "" {
		return apiRequest.LogPath()
	}
	value := parsed.EscapedPath()
	if parsed.RawQuery != "" {
		value += "?" + parsed.RawQuery
	}
	return value
}

func gedixAPIBaseURL(config Config) (string, error) {
	return GedixAPIBaseURL(config)
}

func gedixAPIRequestURL(baseURL string, requestPath string, query map[string]string) (string, error) {
	requestPath = strings.TrimSpace(requestPath)
	if requestPath == "" {
		return "", fmt.Errorf("chemin API requis")
	}
	parsed, err := url.Parse(requestPath)
	if err != nil {
		return "", err
	}
	if parsed.IsAbs() && (parsed.Scheme == "http" || parsed.Scheme == "https") {
		addQuery(parsed, query)
		return parsed.String(), nil
	}
	base, err := url.Parse(strings.TrimRight(baseURL, "/") + "/")
	if err != nil {
		return "", err
	}
	relative, err := url.Parse(strings.TrimLeft(requestPath, "/"))
	if err != nil {
		return "", err
	}
	resolved := base.ResolveReference(relative)
	addQuery(resolved, query)
	return resolved.String(), nil
}

func gedixAPIRequestBody(body any) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(payload), nil
}

func gedixAPIRequestBodyValue(apiRequest GedixAPIRequest, params map[string]any) (any, error) {
	if strings.TrimSpace(apiRequest.BodyJSONParam) == "" {
		return apiRequest.Body, nil
	}
	raw := strings.TrimSpace(stringParam(params, apiRequest.BodyJSONParam))
	if raw == "" {
		return apiRequest.Body, nil
	}
	var parsed any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("%s: JSON invalide: %w", apiRequest.BodyJSONParam, err)
	}
	return parsed, nil
}

func addQuery(target *url.URL, query map[string]string) {
	values := target.Query()
	for key, value := range query {
		values.Set(key, value)
	}
	target.RawQuery = values.Encode()
}

func expectedHTTPStatus(status int, expected []int) bool {
	if len(expected) == 0 {
		return status >= 200 && status < 300
	}
	for _, item := range expected {
		if status == item {
			return true
		}
	}
	return false
}

func redactToken(value string, token string) string {
	if token == "" {
		return value
	}
	return strings.ReplaceAll(value, token, "[REDACTED]")
}
