package lab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultAPIAuthHeader  = "Authorization"
	defaultAPIAuthPrefix  = "Bearer"
	defaultAPIContentType = "application/json; charset=UTF-8"
)

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

type maquetteSecrets struct {
	APIToken string `json:"apiToken"`
}

func ExecuteGedixAPIRequests(requests []GedixAPIRequest) ActionExecute {
	return func(ctx ActionContext, params map[string]any) error {
		return executeGedixAPIRequests(ctx, params, requests)
	}
}

func ExecuteCreatePlant() ActionExecute {
	return func(ctx ActionContext, params map[string]any) error {
		request := GedixAPIRequest{
			Name:             "Créer une usine",
			Method:           "POST",
			Path:             "/entreprise/api/v1/plants",
			Body:             createPlantPayload(params),
			ExpectedStatuses: []int{http.StatusOK},
		}
		if err := executeGedixAPIRequests(ctx, params, []GedixAPIRequest{request}); err != nil {
			return err
		}
		fmt.Fprintf(ctx.Writer, "[API] Usine créée avec succès : %s\n", stringParam(params, "entity_name"))
		return nil
	}
}

func SaveAPIToken(maquetteName string, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("token API requis")
	}
	path := apiTokenPath(maquetteName)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(maquetteSecrets{APIToken: token}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(payload, '\n'), 0600)
}

func DeleteAPIToken(maquetteName string) error {
	path := apiTokenPath(maquetteName)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func LoadAPIToken(maquetteName string) (string, error) {
	data, err := os.ReadFile(apiTokenPath(maquetteName))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var secrets maquetteSecrets
	if err := json.Unmarshal(data, &secrets); err != nil {
		return "", err
	}
	return strings.TrimSpace(secrets.APIToken), nil
}

func HasAPIToken(maquetteName string) (bool, error) {
	token, err := LoadAPIToken(maquetteName)
	return token != "", err
}

func apiTokenPath(maquetteName string) string {
	return filepath.Join(MaquettesDir(), safeDirName(maquetteName), "data", "secrets.json")
}

func executeGedixAPIRequests(ctx ActionContext, params map[string]any, requests []GedixAPIRequest) error {
	token, err := LoadAPIToken(ctx.Config.Name)
	if err != nil {
		return err
	}
	if token == "" {
		return fmt.Errorf("Token API non renseigné pour cette maquette.")
	}
	client := &http.Client{Timeout: 30 * time.Second}
	baseURL, err := gedixAPIBaseURL(ctx.Config)
	if err != nil {
		return err
	}
	for _, apiRequest := range requests {
		if err := executeGedixAPIRequest(client, ctx.Writer, baseURL, token, apiRequest, params); err != nil {
			return err
		}
	}
	return nil
}

func executeGedixAPIRequest(client *http.Client, writer io.Writer, baseURL string, token string, apiRequest GedixAPIRequest, params map[string]any) error {
	method := strings.ToUpper(strings.TrimSpace(apiRequest.Method))
	if method == "" {
		method = "GET"
	}
	targetURL, err := gedixAPIRequestURL(baseURL, apiRequest.Path, apiRequest.Query)
	if err != nil {
		return err
	}
	body, err := gedixAPIRequestBody(apiRequest, params)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(apiRequest.Name)
	if name != "" {
		fmt.Fprintf(writer, "[API] %s\n", name)
	}
	fmt.Fprintf(writer, "[API] %s %s\n", method, apiRequest.LogPath())
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

func gedixAPIBaseURL(config Config) (string, error) {
	if strings.TrimSpace(config.API.BaseURL) != "" {
		return strings.TrimRight(strings.TrimSpace(config.API.BaseURL), "/"), nil
	}
	fqdn := strings.TrimSpace(config.GedixConfig.FQDN)
	if fqdn == "" {
		return "", fmt.Errorf("FQDN Gedix requis pour construire l'URL API")
	}
	host := fqdn
	if config.GedixConfig.Port > 0 {
		host = fmt.Sprintf("%s:%d", fqdn, config.GedixConfig.Port)
	}
	base := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   path.Join("env_"+config.Maquette.EnvName, "app_"+config.Maquette.AppName),
	}
	return strings.TrimRight(base.String(), "/"), nil
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

func gedixAPIRequestBody(apiRequest GedixAPIRequest, params map[string]any) (io.Reader, error) {
	var body any
	if strings.TrimSpace(apiRequest.BodyJSONParam) != "" {
		raw := strings.TrimSpace(stringParam(params, apiRequest.BodyJSONParam))
		if raw != "" {
			var parsed any
			if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
				return nil, fmt.Errorf("%s: JSON invalide: %w", apiRequest.BodyJSONParam, err)
			}
			body = parsed
		}
	}
	if body == nil {
		body = apiRequest.Body
	}
	if body == nil {
		return nil, nil
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(payload), nil
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

func createPlantPayload(params map[string]any) map[string]any {
	return map[string]any{
		"entity_name":        stringParam(params, "entity_name"),
		"description":        stringParam(params, "description"),
		"address_name":       stringParam(params, "address_name"),
		"address_street":     stringParam(params, "address_street"),
		"address_postalcode": stringParam(params, "address_postalcode"),
		"address_town":       stringParam(params, "address_town"),
		"address_country":    stringParam(params, "address_country"),
		"created_by":         numberParam(params, "created_by"),
	}
}

func numberParam(params map[string]any, key string) any {
	switch value := params[key].(type) {
	case int:
		return value
	case int64:
		return value
	case float64:
		if value == float64(int64(value)) {
			return int64(value)
		}
		return value
	case json.Number:
		if integer, err := value.Int64(); err == nil {
			return integer
		}
		if decimal, err := value.Float64(); err == nil {
			return decimal
		}
	}
	return params[key]
}
