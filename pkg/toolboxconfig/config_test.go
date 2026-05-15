package toolboxconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultDerivation(t *testing.T) {
	clearToolboxEnv(t)
	cfg, err := Load("", Overrides{})
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(t, cfg.Web.Addr, ":20251")
	assertEqual(t, cfg.Web.PublicURL, "http://localhost:20251")
	assertEqual(t, cfg.API.Addr, "127.0.0.1:20250")
	assertEqual(t, cfg.API.Target, "http://127.0.0.1:20250")
}

func TestPlatformAndAPIHostDerivation(t *testing.T) {
	clearToolboxEnv(t)
	path := writeConfig(t, `[platform]
fqdn = "192.168.1.50"
port = 20251
tls = false
bind = "0.0.0.0"

[services.api]
host = "127.0.0.1:20250"
`)

	cfg, err := Load(path, Overrides{})
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(t, cfg.Web.Addr, "0.0.0.0:20251")
	assertEqual(t, cfg.Web.PublicURL, "http://192.168.1.50:20251")
	assertEqual(t, cfg.API.Addr, "127.0.0.1:20250")
	assertEqual(t, cfg.API.Target, "http://127.0.0.1:20250")
}

func TestEnvDerivation(t *testing.T) {
	clearToolboxEnv(t)
	t.Setenv("TOOLBOX_FQDN", "toolbox.local")
	t.Setenv("TOOLBOX_PORT", "20443")
	t.Setenv("TOOLBOX_TLS", "true")
	t.Setenv("TOOLBOX_BIND", "0.0.0.0")
	t.Setenv("TOOLBOX_API_HOST", "127.0.0.1:20250")

	cfg, err := Load("", Overrides{})
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(t, cfg.Web.Addr, "0.0.0.0:20443")
	assertEqual(t, cfg.Web.PublicURL, "https://toolbox.local:20443")
	assertEqual(t, cfg.API.Target, "http://127.0.0.1:20250")
}

func TestLegacyOverridesStillWork(t *testing.T) {
	clearToolboxEnv(t)
	path := writeConfig(t, `[platform]
fqdn = "localhost"
port = 20251

[services.api]
host = "127.0.0.1:20250"

[web]
addr = "0.0.0.0:30000"
public_url = "http://legacy.local:30000"

[api]
target = "http://legacy-api:20250"
`)

	cfg, err := Load(path, Overrides{})
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(t, cfg.Web.Addr, "0.0.0.0:30000")
	assertEqual(t, cfg.Web.PublicURL, "http://legacy.local:30000")
	assertEqual(t, cfg.API.Target, "http://legacy-api:20250")
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "toolbox.cfg")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func clearToolboxEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"TOOLBOX_FQDN",
		"TOOLBOX_PORT",
		"TOOLBOX_TLS",
		"TOOLBOX_BIND",
		"TOOLBOX_API_HOST",
		"TOOLBOX_CORS_ORIGINS",
		"TOOLBOX_WEB_ADDR",
		"TOOLBOX_WEB_PUBLIC_URL",
		"TOOLBOX_API_ADDR",
		"TOOLBOX_API_TARGET",
	} {
		t.Setenv(name, "")
	}
}

func assertEqual(t *testing.T, actual, expected string) {
	t.Helper()
	if actual != expected {
		t.Fatalf("expected %q, got %q", expected, actual)
	}
}
