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
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
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
	tempRoot := ""
	if workDir == "" {
		temp, err := os.MkdirTemp("", "v10-lab-")
		if err != nil {
			return err
		}
		tempRoot = temp
	} else if err := os.MkdirAll(workDir, 0755); err != nil {
		return err
	} else {
		tempRoot = filepath.Join(workDir, safeDirName(config.Name)+"-"+time.Now().Format("20060102-150405"))
	}
	target := ResolveMaquetteTargetPath(config)
	if err := os.MkdirAll(tempRoot, 0755); err != nil {
		return err
	}
	success := false
	defer func() {
		if success {
			fmt.Fprintf(ctx.Writer, "[INFO] Nettoyage du répertoire temporaire : %s\n", tempRoot)
			if err := safeRemoveTempDir(tempRoot, workDir, target); err != nil {
				fmt.Fprintf(ctx.Writer, "[WARN] Nettoyage impossible : %v\n", err)
			}
			return
		}
		fmt.Fprintf(ctx.Writer, "[WARN] Création interrompue, répertoire temporaire conservé pour diagnostic : %s\n", tempRoot)
	}()
	extractDir := filepath.Join(tempRoot, "release")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Writer, "[INFO] Décompression de la release : %s\n", zipPath)
	if err := unzip(zipPath, extractDir); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Writer, "[INFO] Release dézippée dans : %s\n", extractDir)
	releaseRoot, err := findReleaseRoot(extractDir)
	if err != nil {
		return err
	}
	gxPath := filepath.Join(releaseRoot, "gx.exe")
	fmt.Fprintf(ctx.Writer, "[INFO] Lancement : %s install --write-config\n", gxPath)
	fmt.Fprintf(ctx.Writer, "[INFO] Répertoire de travail : %s\n", releaseRoot)
	fmt.Fprintln(ctx.Writer, "[INFO] Installation Gedix en cours, cette étape peut durer plusieurs minutes...")
	if err := runInstallCommand(releaseRoot, gxPath, true, ctx.Writer); err != nil {
		return err
	}
	fmt.Fprintln(ctx.Writer, "[INFO] Installation terminée.")
	gedixDir, err := findGedixDirectory(releaseRoot)
	if err != nil {
		return err
	}
	if err := prepareTargetDirectory(target, overwrite); err != nil {
		return err
	}
	fmt.Fprintf(ctx.Writer, "[INFO] Copie du dossier Gedix vers : %s\n", target)
	fmt.Fprintf(ctx.Writer, "[INFO] Source Gedix : %s\n", gedixDir)
	if err := copyDir(gedixDir, target); err != nil {
		return err
	}
	fmt.Fprintln(ctx.Writer, "[INFO] Copie terminée.")
	fmt.Fprintf(ctx.Writer, "Maquette créée: %s\n", target)
	success = true
	return nil
}

func UpdateEnv(ctx ActionContext, params map[string]any) error {
	config := ctx.Config
	ApplyDefaults(&config)
	zipPath := firstNonEmpty(stringParam(params, "zipPath"), config.Release.ZipPath)
	workDir := config.Release.WorkDir
	target := ResolveMaquetteTargetPath(config)
	if err := validateUpdateEnvInputs(zipPath, target); err != nil {
		return err
	}
	tempRoot, err := makeUpdateTempDir(workDir, config.Name)
	if err != nil {
		return err
	}
	success := false
	defer func() {
		if success {
			fmt.Fprintln(ctx.Writer, "[INFO] Nettoyage du dossier temporaire.")
			if err := safeRemoveTempDir(tempRoot, workDir, target); err != nil {
				fmt.Fprintf(ctx.Writer, "[WARN] Nettoyage impossible : %v\n", err)
			}
			fmt.Fprintln(ctx.Writer, "[INFO] Mise à jour terminée.")
			return
		}
		fmt.Fprintf(ctx.Writer, "[WARN] Mise à jour interrompue, dossier temporaire conservé pour diagnostic : %s\n", tempRoot)
	}()
	fmt.Fprintf(ctx.Writer, "[INFO] Préparation de la mise à jour de la maquette %s.\n", config.Name)
	fmt.Fprintf(ctx.Writer, "[INFO] Release utilisée : %s\n", zipPath)
	fmt.Fprintf(ctx.Writer, "[INFO] Dossier cible : %s\n", target)
	extractDir := filepath.Join(tempRoot, "release")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return err
	}
	fmt.Fprintln(ctx.Writer, "[INFO] Décompression de la release...")
	if err := unzip(zipPath, extractDir); err != nil {
		return err
	}
	fmt.Fprintln(ctx.Writer, "[INFO] Décompression terminée.")
	releaseRoot, err := findReleaseRoot(extractDir)
	if err != nil {
		return err
	}
	gxPath := filepath.Join(releaseRoot, "gx.exe")
	fmt.Fprintln(ctx.Writer, "[INFO] Lancement : gx.exe install")
	fmt.Fprintln(ctx.Writer, "[INFO] Installation Gedix en cours, cette étape peut durer plusieurs minutes...")
	if err := runInstallCommand(releaseRoot, gxPath, false, ctx.Writer); err != nil {
		return err
	}
	fmt.Fprintln(ctx.Writer, "[INFO] Installation terminée.")
	gedixDir, err := findGedixDirectory(releaseRoot)
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.Writer, "[INFO] Copie des fichiers applicatifs vers la maquette...")
	fmt.Fprintln(ctx.Writer, "[INFO] Exclusions : gedix.cfg")
	if err := copyDirForUpdate(gedixDir, target); err != nil {
		return err
	}
	fmt.Fprintln(ctx.Writer, "[INFO] Copie terminée.")
	success = true
	return nil
}

func StartMaquette(config Config, writer io.Writer) error {
	paths, err := DetectGedixPaths(config)
	if err != nil {
		return err
	}
	if len(config.Runtime.DebugTargets) > 0 {
		fmt.Fprintf(writer, "[INFO] Cibles debug : %s\n", strings.Join(config.Runtime.DebugTargets, ", "))
	}
	fmt.Fprintf(writer, "Démarrage gx-front : %s\n", consoleCommandLine(paths.FrontExePath, "listen"))
	if err := openConsole(paths.GedixRoot, "V10 Lab gx-front", paths.FrontExePath, "listen"); err != nil {
		return err
	}
	appArgs := []string{"run"}
	if len(config.Runtime.DebugTargets) > 0 {
		appArgs = append(appArgs, "-e")
		appArgs = append(appArgs, debugExclusionArg(config.Runtime.DebugTargets))
		fmt.Fprintf(writer, "[INFO] Lancement gx-app avec exclusions : gx-app.exe %s\n", strings.Join(appArgs, " "))
	}
	fmt.Fprintf(writer, "Démarrage gx-app : %s\n", consoleCommandLine(paths.AppExePath, appArgs...))
	if err := openConsole(paths.AppPath, "V10 Lab gx-app", paths.AppExePath, appArgs...); err != nil {
		return err
	}
	for _, target := range config.Runtime.DebugTargets {
		debugTarget, err := DetectDebugTarget(paths, target)
		if err != nil {
			return err
		}
		if debugTarget.Kind == DebugTargetConnector {
			fmt.Fprintf(writer, "[INFO] Lancement connecteur debug %s : gx-connector.exe listen --debug -v2\n", debugTarget.Name)
		} else {
			fmt.Fprintf(writer, "[INFO] Lancement service debug %s : %s listen --debug -v2\n", debugTarget.Name, filepath.Base(debugTarget.ExePath))
		}
		fmt.Fprintf(writer, "Démarrage debug %s (%s) : %s\n", debugTarget.Name, debugTarget.Kind, consoleCommandLine(debugTarget.ExePath, "listen", "--debug", "-v2"))
		if err := openConsole(debugTarget.WorkDir, "V10 Lab debug "+debugTarget.Name, debugTarget.ExePath, "listen", "--debug", "-v2"); err != nil {
			return err
		}
	}
	return nil
}

func debugExclusionArg(targets []string) string {
	return strings.Join(targets, ",")
}

func KillGXProcesses(writer io.Writer, force bool, interactive bool) error {
	fmt.Fprintln(writer, "[INFO] Coupure des services GX demandée.")
	fmt.Fprintln(writer, "[INFO] Commande exécutée : taskkill -f -t -im gx-*")
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
	cmd := exec.Command("powershell", "-NoProfile", "-Command", `[Console]::OutputEncoding=[System.Text.UTF8Encoding]::new(); taskkill /f /t /im gx-* 2>&1`)
	output, err := cmd.CombinedOutput()
	outputText := decodeCommandOutput(output)
	if len(output) > 0 {
		fmt.Fprintln(writer, strings.TrimSpace(outputText))
	}
	if err != nil && strings.Contains(strings.ToLower(outputText), "not found") {
		fmt.Fprintln(writer, "Aucun service GX à couper.")
		return nil
	}
	if err != nil && strings.Contains(strings.ToLower(outputText), "introuvable") {
		fmt.Fprintln(writer, "Aucun service GX à couper.")
		return nil
	}
	return err
}

func runInstallCommand(dir string, gxPath string, writeConfig bool, writer io.Writer) error {
	args := []string{"install"}
	if writeConfig {
		args = append(args, "--write-config")
	}
	cmd := exec.Command(gxPath, args...)
	cmd.Dir = dir
	cmd.Stdout = prefixedWriter{writer: writer, prefix: "[GX] "}
	cmd.Stderr = prefixedWriter{writer: writer, prefix: "[GX] "}
	return cmd.Run()
}

type prefixedWriter struct {
	writer io.Writer
	prefix string
}

func (w prefixedWriter) Write(payload []byte) (int, error) {
	text := strings.TrimRight(decodeCommandOutput(payload), "\r\n")
	if text != "" {
		for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
			if strings.TrimSpace(line) != "" {
				fmt.Fprintln(w.writer, w.prefix+line)
			}
		}
	}
	return len(payload), nil
}

func decodeCommandOutput(payload []byte) string {
	if utf8.Valid(payload) {
		return string(payload)
	}
	for _, decoder := range []*charmap.Charmap{
		charmap.CodePage850,
		charmap.CodePage437,
		charmap.Windows1252,
	} {
		text, err := decoder.NewDecoder().String(string(payload))
		if err == nil {
			return text
		}
	}
	return string(payload)
}

func openConsole(dir string, title string, exe string, args ...string) error {
	commandLine := consoleCommandLine(exe, args...)
	if runtime.GOOS != "windows" {
		fmt.Printf("[DRY-RUN non-windows] cd %s && %s\n", quoteCmdArg(dir), commandLine)
		return nil
	}
	cmd := exec.Command("cmd", openConsoleArgs(dir, title, commandLine)...)
	return cmd.Start()
}

func openConsoleArgs(dir string, title string, commandLine string) []string {
	return []string{"/C", "start", title, "/D", dir, "cmd", "/K", commandLine}
}

func consoleCommandLine(exe string, args ...string) string {
	parts := []string{quoteWindowsPath(exe)}
	for _, arg := range args {
		parts = append(parts, quoteCmdArg(arg))
	}
	return strings.Join(parts, " ")
}

func quoteWindowsPath(path string) string {
	if strings.HasPrefix(path, `"`) && strings.HasSuffix(path, `"`) {
		return path
	}
	return `"` + strings.ReplaceAll(path, `"`, `\"`) + `"`
}

func quoteCmdArg(value string) string {
	if strings.ContainsAny(value, " \t&()[]{}^=;!'`~") {
		return quoteWindowsPath(value)
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
		return "", fmt.Errorf("dossier Gedix créé introuvable après gx.exe install")
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

func validateUpdateEnvInputs(zipPath string, target string) error {
	if strings.TrimSpace(zipPath) == "" {
		return fmt.Errorf("release.zipPath est requis pour mettre à jour la maquette")
	}
	info, err := os.Stat(zipPath)
	if err != nil {
		return fmt.Errorf("ZIP release introuvable %s: %w", zipPath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("release.zipPath doit pointer vers un fichier ZIP: %s", zipPath)
	}
	if !strings.EqualFold(filepath.Ext(zipPath), ".zip") {
		return fmt.Errorf("release.zipPath doit pointer vers un fichier .zip: %s", zipPath)
	}
	if err := ensureSafeExistingTargetPath(target); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(target, "gx-front.exe")); err != nil {
		return fmt.Errorf("maquette Gedix invalide: gx-front.exe introuvable dans %s: %w", target, err)
	}
	if !hasEnvDirectory(target) {
		return fmt.Errorf("maquette Gedix invalide: aucun dossier env_* trouvé dans %s", target)
	}
	return nil
}

func ensureSafeExistingTargetPath(target string) error {
	if strings.TrimSpace(target) == "" {
		return fmt.Errorf("dossier cible de maquette requis")
	}
	clean, err := filepath.Abs(filepath.Clean(target))
	if err != nil {
		return err
	}
	volume := filepath.VolumeName(clean)
	root := volume + string(os.PathSeparator)
	if clean == root || clean == volume || clean == string(os.PathSeparator) {
		return fmt.Errorf("chemin cible dangereux refusé: %s", target)
	}
	if len(strings.Trim(clean, `\/ `)) < 6 {
		return fmt.Errorf("chemin cible dangereux refusé: %s", target)
	}
	info, err := os.Stat(clean)
	if err != nil {
		return fmt.Errorf("dossier cible de maquette introuvable %s: %w", target, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("dossier cible de maquette invalide %s: ce n'est pas un dossier", target)
	}
	return nil
}

func hasEnvDirectory(target string) bool {
	entries, err := os.ReadDir(target)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(strings.ToLower(entry.Name()), "env_") {
			return true
		}
	}
	return false
}

func makeUpdateTempDir(workDir string, maquetteName string) (string, error) {
	prefix := "v10-lab-update-" + safeDirName(maquetteName) + "-" + time.Now().Format("20060102-150405")
	if strings.TrimSpace(workDir) == "" {
		return os.MkdirTemp("", prefix+"-")
	}
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return "", err
	}
	return os.MkdirTemp(workDir, prefix+"-")
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

func safeRemoveTempDir(path string, protectedPaths ...string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("refus de supprimer un chemin vide")
	}
	clean, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return err
	}
	if err := ensureSafeDeletePath(clean); err != nil {
		return err
	}
	for _, protected := range protectedPaths {
		if strings.TrimSpace(protected) == "" {
			continue
		}
		protectedClean, err := filepath.Abs(filepath.Clean(protected))
		if err != nil {
			return err
		}
		if strings.EqualFold(clean, protectedClean) {
			return fmt.Errorf("refus de supprimer un chemin protégé: %s", path)
		}
	}
	return os.RemoveAll(clean)
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

func copyDirForUpdate(source string, target string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if entry.IsDir() && isUpdateExcludedDir(entry.Name()) {
			return filepath.SkipDir
		}
		if !entry.IsDir() && isUpdateExcludedFile(entry.Name()) {
			return nil
		}
		dest := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		return copyFile(path, dest)
	})
}

func isUpdateExcludedDir(name string) bool {
	return strings.EqualFold(name, "log")
}

func isUpdateExcludedFile(name string) bool {
	return strings.EqualFold(name, "gedix.cfg")
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
