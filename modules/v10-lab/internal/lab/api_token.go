package lab

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type maquetteSecrets struct {
	APIToken string `json:"apiToken"`
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
