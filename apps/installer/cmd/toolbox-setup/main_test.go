package main

import (
	"strings"
	"testing"
)

func TestStartScriptUsesToolboxConfigForURL(t *testing.T) {
	content := startScriptContent()
	for _, expected := range []string{
		`api-toolbox.exe" server --config "%~dp0toolbox.cfg"`,
		`web-server-toolbox.exe" start --config "%~dp0toolbox.cfg"`,
		`set "TOOLBOX_URL=http://localhost:20251"`,
		`Join-Path (Get-Location) 'toolbox.cfg'`,
		`$scheme + '://' + $fqdn + ':' + $port`,
		`start "" "%TOOLBOX_URL%"`,
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected start script to contain %q, got:\n%s", expected, content)
		}
	}
}
