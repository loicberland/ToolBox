package lab

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func CreateEnv(ctx ActionContext, params map[string]any) error {
	config := ctx.Config
	zipPath := firstNonEmpty(stringParam(params, "zipPath"), config.Release.ZipPath)
	workDir := firstNonEmpty(stringParam(params, "workDir"), config.Release.WorkDir)
	overwrite := config.Release.Overwrite || boolParam(params, "overwrite")
	if strings.TrimSpace(zipPath) == "" {
		return fmt.Errorf("release.zipPath est requis")
	}
	if _, err := os.Stat(zipPath); err != nil {
		return fmt.Errorf("ZIP release introuvable %s: %w", zipPath, err)
	}
	if workDir == "" {
		temp, err := os.MkdirTemp("", "v10-lab-*")
		if err != nil {
			return err
		}
		workDir = temp
	} else if err := os.MkdirAll(workDir, 0755); err != nil {
		return err
	}
	extractDir := filepath.Join(workDir, "release-"+safeDirName(config.Name))
	if err := os.RemoveAll(extractDir); err != nil {
		return err
	}
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Writer, "Décompression release: %s\n", zipPath)
	if err := unzip(zipPath, extractDir); err != nil {
		return err
	}
	releaseRoot, err := findReleaseRoot(extractDir)
	if err != nil {
		return err
	}
	gxPath := filepath.Join(releaseRoot, "gx.exe")
	fmt.Fprintf(ctx.Writer, "Installation Gedix: %s install --write-config\n", gxPath)
	if err := runInstallCommand(releaseRoot, gxPath); err != nil {
		return err
	}
	gedixDir, err := findGedixDirectory(releaseRoot)
	if err != nil {
		return err
	}
	target := ResolveMaquetteTargetPath(config)
	if err := prepareTargetDirectory(target, overwrite); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Writer, "Copie maquette: %s -> %s\n", gedixDir, target)
	if err := copyDir(gedixDir, target); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Writer, "Maquette créée: %s\n", target)
	return nil
}

func StartMaquette(config Config, writer io.Writer) error {
	paths, err := DetectGedixPaths(config)
	if err != nil {
		return err
	}
	fmt.Fprintf(writer, "Démarrage gx-front dans %s\n", paths.GedixRoot)
	if err := openConsole(paths.GedixRoot, "V10 Lab gx-front", paths.FrontExePath, "listen"); err != nil {
		return err
	}
	appArgs := []string{"run"}
	if len(config.Runtime.DebugTargets) > 0 {
		appArgs = append(appArgs, "-e")
		appArgs = append(appArgs, config.Runtime.DebugTargets...)
	}
	fmt.Fprintf(writer, "Démarrage gx-app dans %s: gx-app.exe %s\n", paths.AppPath, strings.Join(appArgs, " "))
	if err := openConsole(paths.AppPath, "V10 Lab gx-app", paths.AppExePath, appArgs...); err != nil {
		return err
	}
	for _, target := range config.Runtime.DebugTargets {
		debugTarget, err := DetectDebugTarget(paths, target)
		if err != nil {
			return err
		}
		fmt.Fprintf(writer, "Démarrage debug %s (%s)\n", debugTarget.Name, debugTarget.Kind)
		if err := openConsole(debugTarget.WorkDir, "V10 Lab debug "+debugTarget.Name, debugTarget.ExePath, "listen", "--debug", "-v2"); err != nil {
			return err
		}
	}
	return nil
}

func KillGXProcesses(writer io.Writer, force bool, interactive bool) error {
	fmt.Fprintln(writer, "WARNING: cette commande tue tous les processus gx-* avec taskkill.")
	if !force && interactive {
		fmt.Fprint(writer, "Confirmer ? tapez OUI: ")
		var answer string
		_, _ = fmt.Scanln(&answer)
		if answer != "OUI" {
			fmt.Fprintln(writer, "Annulé.")
			return nil
		}
	}
	if !force && !interactive {
		return fmt.Errorf("confirmation requise: relancez avec --force")
	}
	if runtime.GOOS != "windows" {
		return fmt.Errorf("kill-gx-processes est disponible uniquement sur Windows")
	}
	cmd := exec.Command("taskkill", "-f", "-t", "-im", "gx-*")
	output, err := cmd.CombinedOutput()
	if len(output) > 0 {
		fmt.Fprintln(writer, strings.TrimSpace(string(output)))
	}
	return err
}

func runInstallCommand(dir string, gxPath string) error {
	cmd := exec.Command(gxPath, "install", "--write-config")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func openConsole(dir string, title string, exe string, args ...string) error {
	if runtime.GOOS != "windows" {
		fmt.Printf("[DRY-RUN non-windows] cd %s && %s %s\n", dir, exe, strings.Join(args, " "))
		return nil
	}
	commandLine := quoteCmdArg(exe)
	for _, arg := range args {
		commandLine += " " + quoteCmdArg(arg)
	}
	cmd := exec.Command("cmd", "/C", "start", title, "/D", dir, "cmd", "/K", commandLine)
	return cmd.Start()
}

func quoteCmdArg(value string) string {
	if strings.ContainsAny(value, " \t&()[]{}^=;!'+,`~") {
		return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
	}
	return value
}

func unzip(zipPath string, targetDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, file := range reader.File {
		target := filepath.Join(targetDir, filepath.FromSlash(file.Name))
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(targetDir)+string(os.PathSeparator)) {
			return fmt.Errorf("chemin ZIP dangereux: %s", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		source, err := file.Open()
		if err != nil {
			return err
		}
		destination, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.FileInfo().Mode())
		if err != nil {
			_ = source.Close()
			return err
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
	}
	return nil
}

func findReleaseRoot(extractDir string) (string, error) {
	if _, err := os.Stat(filepath.Join(extractDir, "gx.exe")); err == nil {
		return extractDir, nil
	}
	matches := []string{}
	if err := filepath.WalkDir(extractDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if _, err := os.Stat(filepath.Join(path, "gx.exe")); err == nil {
				matches = append(matches, path)
			}
		}
		return nil
	}); err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("gx.exe introuvable dans la release dézippée")
	}
	return matches[0], nil
}

func findGedixDirectory(releaseRoot string) (string, error) {
	candidates := []string{}
	entries, err := os.ReadDir(releaseRoot)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(releaseRoot, entry.Name())
		if _, err := os.Stat(filepath.Join(path, "gx-front.exe")); err == nil {
			candidates = append(candidates, path)
		}
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("dossier Gedix créé introuvable après gx.exe install --write-config")
	}
	return candidates[0], nil
}

func prepareTargetDirectory(target string, overwrite bool) error {
	if strings.TrimSpace(target) == "" {
		return fmt.Errorf("targetPath vide")
	}
	if !overwrite {
		if _, err := os.Stat(target); err == nil {
			return fmt.Errorf("le dossier cible existe déjà: %s (overwrite=false)", target)
		} else if !os.IsNotExist(err) {
			return err
		}
		return os.MkdirAll(filepath.Dir(target), 0755)
	}
	if err := ensureSafeDeletePath(target); err != nil {
		return err
	}
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	return os.MkdirAll(filepath.Dir(target), 0755)
}

func ensureSafeDeletePath(target string) error {
	clean, err := filepath.Abs(filepath.Clean(target))
	if err != nil {
		return err
	}
	volume := filepath.VolumeName(clean)
	root := volume + string(os.PathSeparator)
	if clean == root || clean == volume || clean == string(os.PathSeparator) {
		return fmt.Errorf("refus de supprimer un chemin racine: %s", target)
	}
	if len(strings.Trim(clean, `\/ `)) < 6 {
		return fmt.Errorf("refus de supprimer un chemin trop court: %s", target)
	}
	return nil
}

func copyDir(source string, target string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		return copyFile(path, dest)
	})
}

func copyFile(source string, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
