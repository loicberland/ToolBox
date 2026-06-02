package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"toolBox/pkg/toolboxconfig"
)

var payloadMarker = []byte("TOOLBOX_PACKAGE_PAYLOAD_V1")

const accessDeniedMessage = "Accès refusé au dossier cible. Choisis un dossier utilisateur avec --dir ou relance l’installeur en administrateur si tu veux installer dans un dossier protégé."

func main() {
	parentDir := flag.String("dir", ".", "parent directory where ToolBox will be installed")
	forceConfig := flag.Bool("force-config", false, "overwrite ToolBox/toolbox.cfg with defaults")
	cleanExe := flag.Bool("clean-exe", false, "remove known ToolBox executables before reinstalling them")
	flag.Parse()

	if err := install(*parentDir, *forceConfig, *cleanExe); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func install(parentDir string, forceConfig, cleanExe bool) error {
	parent, err := filepath.Abs(parentDir)
	if err != nil {
		return err
	}
	root := filepath.Join(parent, "ToolBox")
	if err := os.MkdirAll(root, 0755); err != nil {
		return targetAccessError(root, err)
	}

	if cleanExe {
		for _, rel := range []string{
			"api-toolbox.exe",
			"web-server-toolbox.exe",
			filepath.Join("modules", "test-sheet", "test-sheet.exe"),
			filepath.Join("modules", "v10-lab", "v10-lab.exe"),
		} {
			if err := os.Remove(filepath.Join(root, rel)); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}

	if err := extractPayload(root); err != nil {
		return err
	}
	if err := ensureRuntimeDirs(root); err != nil {
		return err
	}
	if err := ensureConfig(root, forceConfig); err != nil {
		return err
	}
	if err := ensureStartScript(root); err != nil {
		return err
	}

	fmt.Printf("ToolBox installed in %s\n", root)
	return nil
}

func extractPayload(root string) error {
	data, err := readAppendedPayload()
	if err != nil {
		return err
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, file := range archive.File {
		if file.FileInfo().IsDir() || !isSafeZipPath(file.Name) {
			continue
		}
		target := filepath.Join(root, filepath.FromSlash(file.Name))
		writeTarget := target
		if strings.HasSuffix(strings.ToLower(target), ".exe") {
			writeTarget = target + ".tmp"
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return targetAccessError(filepath.Dir(target), err)
		}
		source, err := file.Open()
		if err != nil {
			return err
		}
		destination, err := os.OpenFile(writeTarget, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.FileInfo().Mode())
		if err != nil {
			_ = source.Close()
			return targetAccessError(writeTarget, err)
		}
		_, copyErr := io.Copy(destination, source)
		closeErr := destination.Close()
		_ = source.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if strings.HasSuffix(strings.ToLower(target), ".exe") {
			_ = os.Chmod(writeTarget, 0755)
			if err := os.Rename(writeTarget, target); err != nil {
				_ = os.Remove(writeTarget)
				return targetAccessError(target, err)
			}
		}
	}
	return nil
}

func readAppendedPayload() ([]byte, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(executable)
	if err != nil {
		return nil, err
	}
	index := bytes.LastIndex(data, payloadMarker)
	if index < 0 {
		return nil, fmt.Errorf("package payload not found")
	}
	return data[index+len(payloadMarker):], nil
}

func isSafeZipPath(path string) bool {
	if path == "" || strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") {
		return false
	}
	for _, part := range strings.FieldsFunc(path, func(r rune) bool { return r == '/' || r == '\\' }) {
		if part == ".." {
			return false
		}
	}
	return true
}

func ensureRuntimeDirs(root string) error {
	for _, dir := range []string{
		filepath.Join(root, "modules", "test-sheet", "data"),
		filepath.Join(root, "modules", "test-sheet", "files", "documents"),
		filepath.Join(root, "modules", "test-sheet", "files", "runs"),
		filepath.Join(root, "modules", "v10-lab", "data", "maquettes"),
		filepath.Join(root, "modules", "v10-lab", "files"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return targetAccessError(dir, err)
		}
	}
	return nil
}

func ensureConfig(root string, force bool) error {
	path := filepath.Join(root, "toolbox.cfg")
	if !force {
		if _, err := os.Stat(path); err == nil {
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	return targetAccessError(path, os.WriteFile(path, []byte(toolboxconfig.DefaultConfigFile), 0644))
}

func ensureStartScript(root string) error {
	if err := os.WriteFile(filepath.Join(root, "ToolBox Start.bat"), []byte(startScriptContent()), 0644); err != nil {
		return targetAccessError(filepath.Join(root, "ToolBox Start.bat"), err)
	}
	if err := os.Remove(filepath.Join(root, "ToolBox Url.ps1")); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func targetAccessError(path string, err error) error {
	if err == nil {
		return nil
	}
	if os.IsPermission(err) || errors.Is(err, os.ErrPermission) {
		return fmt.Errorf("%s Chemin: %s. Erreur originale: %w", accessDeniedMessage, path, err)
	}
	return err
}

func startScriptContent() string {
	return "@echo off\r\n" +
		"cd /d \"%~dp0\"\r\n" +
		"\r\n" +
		"start \"ToolBox api\" cmd /k \"\"%~dp0api-toolbox.exe\" server --config \"%~dp0toolbox.cfg\"\"\r\n" +
		"start \"ToolBox front\" cmd /k \"\"%~dp0web-server-toolbox.exe\" start --config \"%~dp0toolbox.cfg\" --open\"\r\n" +
		"\r\n" +
		"exit\r\n"
}
