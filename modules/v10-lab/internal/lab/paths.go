package lab

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type GedixPaths struct {
	GedixRoot    string
	CfgPath      string
	EnvName      string
	EnvPath      string
	AppName      string
	AppPath      string
	FrontExePath string
	AppExePath   string
}

type DebugTargetKind string

const (
	DebugTargetService   DebugTargetKind = "service"
	DebugTargetConnector DebugTargetKind = "connector"
)

type DebugTarget struct {
	Name    string
	Kind    DebugTargetKind
	WorkDir string
	ExePath string
}

func DetectGedixPaths(config Config) (GedixPaths, error) {
	ApplyDefaults(&config)
	root := ResolveMaquetteTargetPath(config)
	paths := GedixPaths{
		GedixRoot:    root,
		FrontExePath: filepath.Join(root, "gx-front.exe"),
		AppName:      config.Maquette.AppName,
	}
	if _, err := os.Stat(paths.FrontExePath); err != nil {
		return paths, fmt.Errorf("gx-front.exe introuvable dans %s: %w", root, err)
	}
	envName, envPath, err := detectEnv(root, "env_"+config.Maquette.EnvName)
	if err != nil {
		return paths, err
	}
	paths.EnvName = envName
	paths.EnvPath = envPath
	paths.AppPath = filepath.Join(envPath, "app_"+paths.AppName)
	paths.AppExePath = filepath.Join(paths.AppPath, "gx-app.exe")
	if _, err := os.Stat(paths.AppPath); err != nil {
		return paths, fmt.Errorf("application app_%s introuvable dans %s: %w", paths.AppName, envPath, err)
	}
	cfgPath, err := detectOrCreateCfg(root, paths.FrontExePath)
	if err != nil {
		return paths, err
	}
	paths.CfgPath = cfgPath
	return paths, nil
}

func detectEnv(root string, configured string) (string, string, error) {
	if strings.TrimSpace(configured) != "" {
		path := filepath.Join(root, configured)
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			if err == nil {
				err = fmt.Errorf("not a directory")
			}
			return "", "", fmt.Errorf("env configuré %q introuvable: %w", configured, err)
		}
		return configured, path, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", "", err
	}
	matches := []string{}
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(strings.ToLower(entry.Name()), "env_") {
			matches = append(matches, entry.Name())
		}
	}
	sort.Strings(matches)
	if len(matches) == 0 {
		return "", "", fmt.Errorf("aucun dossier env_* trouvé dans %s", root)
	}
	return matches[0], filepath.Join(root, matches[0]), nil
}

func detectOrCreateCfg(root string, frontExePath string) (string, error) {
	cfgs, err := filepath.Glob(filepath.Join(root, "*.cfg"))
	if err != nil {
		return "", err
	}
	preferred := filepath.Join(root, "gedix.cfg")
	for _, cfg := range cfgs {
		if strings.EqualFold(filepath.Base(cfg), "gedix.cfg") {
			return cfg, nil
		}
	}
	if len(cfgs) == 1 {
		return cfgs[0], nil
	}
	if len(cfgs) > 1 {
		return "", fmt.Errorf("plusieurs fichiers .cfg trouvés dans %s; conservez gedix.cfg ou corrigez la maquette", root)
	}
	if err := runCommand(root, frontExePath, "config", "write"); err != nil {
		return "", fmt.Errorf("génération gedix.cfg: %w", err)
	}
	templatePath := filepath.Join(root, "gedix.cfg.templ")
	if _, err := os.Stat(templatePath); err != nil {
		return "", fmt.Errorf("gedix.cfg.templ introuvable après gx-front.exe config write: %w", err)
	}
	if err := os.Rename(templatePath, preferred); err != nil {
		return "", err
	}
	return preferred, nil
}

func DetectDebugTarget(paths GedixPaths, target string) (DebugTarget, error) {
	serviceExe := filepath.Join(paths.AppPath, "gx-"+target+".exe")
	if info, err := os.Stat(serviceExe); err == nil && !info.IsDir() {
		return DebugTarget{Name: target, Kind: DebugTargetService, WorkDir: paths.AppPath, ExePath: serviceExe}, nil
	}
	connectorExe := filepath.Join(paths.AppPath, target, "gx-connector.exe")
	if info, err := os.Stat(connectorExe); err == nil && !info.IsDir() {
		return DebugTarget{Name: target, Kind: DebugTargetConnector, WorkDir: filepath.Join(paths.AppPath, target), ExePath: connectorExe}, nil
	}
	return DebugTarget{}, fmt.Errorf("cible debug %q introuvable: ni service gx-%s.exe ni connector %s/gx-connector.exe", target, target, target)
}

func runCommand(dir string, exe string, args ...string) error {
	cmd := exec.Command(exe, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w: %s", exe, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}
