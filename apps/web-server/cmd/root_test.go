package cmd

import (
	"runtime"
	"testing"
)

func TestOpenBrowserCommandUsesConfiguredURL(t *testing.T) {
	rawURL := "http://PORTLB:80"
	name, args := openBrowserCommand(rawURL)
	switch runtime.GOOS {
	case "windows":
		if name != "cmd" || len(args) != 4 || args[0] != "/C" || args[1] != "start" || args[3] != rawURL {
			t.Fatalf("unexpected Windows open command: %s %#v", name, args)
		}
	case "darwin":
		if name != "open" || len(args) != 1 || args[0] != rawURL {
			t.Fatalf("unexpected macOS open command: %s %#v", name, args)
		}
	case "linux":
		if name != "xdg-open" || len(args) != 1 || args[0] != rawURL {
			t.Fatalf("unexpected Linux open command: %s %#v", name, args)
		}
	}
}
