package lab

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
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
	if err := ValidateReleaseZipPath(zipPath); err != nil {
		return err
	}
	zipPath = strings.TrimSpace(zipPath)
	target := ResolveMaquetteTargetPath(config)
	fmt.Fprintf(ctx.Writer, "[INFO] Vérification du dossier cible : %s\n", target)
	if err := checkCreateTargetAvailable(target, overwrite); err != nil {
		return err
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
	zipPath = strings.TrimSpace(zipPath)
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
	ApplyDefaults(&config)
	product, err := ProductDefinitionByID(config.Product)
	if err != nil {
		return err
	}
	paths, err := DetectGedixPaths(config)
	if err != nil {
		return err
	}
	debugTargets := RuntimeDebugLaunchTargets(config.Runtime)
	if len(debugTargets) > 0 {
		fmt.Fprintf(writer, "[INFO] Cibles debug : %s\n", strings.Join(debugTargets, ", "))
	}
	fmt.Fprintf(writer, "Démarrage gx-front : %s\n", consoleCommandLine(paths.FrontExePath, "listen"))
	if err := openConsole(paths.GedixRoot, "V10 Lab gx-front", paths.FrontExePath, "listen"); err != nil {
		return err
	}
	appArgs := []string{"run"}
	if len(debugTargets) > 0 {
		appArgs = append(appArgs, "-e")
		appArgs = append(appArgs, debugExclusionArg(debugTargets))
		fmt.Fprintf(writer, "[INFO] Lancement gx-app avec exclusions : gx-app.exe %s\n", strings.Join(appArgs, " "))
	}
	fmt.Fprintf(writer, "Démarrage gx-app : %s\n", consoleCommandLine(paths.AppExePath, appArgs...))
	if err := openConsole(paths.AppPath, "V10 Lab gx-app", paths.AppExePath, appArgs...); err != nil {
		return err
	}
	for _, target := range debugTargets {
		debugTarget, err := DetectDebugTargetForProductConfig(paths, target, product, config)
		if err != nil {
			return err
		}
		debugArgs, err := debugArgsForTarget(config.Runtime, target)
		if err != nil {
			return err
		}
		if debugTarget.Kind == DebugTargetConnector || debugTarget.Kind == DebugTargetAgent || debugTarget.Kind == DebugTargetAdaptor {
			fmt.Fprintf(writer, "[INFO] Lancement %s debug %s : %s %s\n", debugTarget.Kind, debugTarget.Name, filepath.Base(debugTarget.ExePath), strings.Join(debugArgs, " "))
		} else {
			fmt.Fprintf(writer, "[INFO] Lancement service debug %s : %s %s\n", debugTarget.Name, filepath.Base(debugTarget.ExePath), strings.Join(debugArgs, " "))
		}
		fmt.Fprintf(writer, "Démarrage debug %s (%s) : %s\n", debugTarget.Name, debugTarget.Kind, consoleCommandLine(debugTarget.ExePath, debugArgs...))
		if err := openConsole(debugTarget.WorkDir, "V10 Lab debug "+debugTarget.Name, debugTarget.ExePath, debugArgs...); err != nil {
			return err
		}
	}
	return nil
}

type ModuleCommandRequest struct {
	UnitName string
	Command  string
}

type ExecutableCommandTargetKind string

const (
	ExecutableCommandTargetRoot      ExecutableCommandTargetKind = "root"
	ExecutableCommandTargetService   ExecutableCommandTargetKind = "service"
	ExecutableCommandTargetConnector ExecutableCommandTargetKind = "connector"
	ExecutableCommandTargetAgent     ExecutableCommandTargetKind = "agent"
	ExecutableCommandTargetAdaptor   ExecutableCommandTargetKind = "adaptor"
)

type ExecutableCommandRequest struct {
	TargetKind ExecutableCommandTargetKind
	TargetName string
	Command    string
}

type ExecutableCommandTarget struct {
	Kind    ExecutableCommandTargetKind
	Name    string
	ExePath string
	WorkDir string
}

func RunModuleCommand(config Config, request ModuleCommandRequest, writer io.Writer) error {
	return RunExecutableCommand(config, ExecutableCommandRequest{
		TargetKind: ExecutableCommandTargetConnector,
		TargetName: request.UnitName,
		Command:    request.Command,
	}, writer)
}

func RunExecutableCommand(config Config, request ExecutableCommandRequest, writer io.Writer) error {
	ApplyDefaults(&config)
	product, err := ProductDefinitionByID(config.Product)
	if err != nil {
		return err
	}
	targetKind := ExecutableCommandTargetKind(strings.TrimSpace(string(request.TargetKind)))
	targetName := strings.TrimSpace(request.TargetName)
	command := strings.TrimSpace(request.Command)
	if targetKind == "" {
		return fmt.Errorf("type de cible requis")
	}
	if targetName == "" {
		return fmt.Errorf("cible requise")
	}
	if command == "" {
		return fmt.Errorf("commande exécutable requise")
	}
	if _, err := splitCommandLine(command); err != nil {
		return err
	}
	paths, err := DetectGedixPaths(config)
	if err != nil {
		return err
	}
	target, err := ResolveExecutableCommandTarget(paths, product, config, targetKind, targetName)
	if err != nil {
		return err
	}
	fmt.Fprintln(writer, "[INFO] Lancement commande exécutable.")
	fmt.Fprintf(writer, "[INFO] Type : %s\n", target.Kind)
	fmt.Fprintf(writer, "[INFO] Cible : %s\n", target.Name)
	fmt.Fprintf(writer, "[INFO] Exécutable : %s\n", target.ExePath)
	fmt.Fprintf(writer, "[INFO] Répertoire de travail : %s\n", target.WorkDir)
	fmt.Fprintf(writer, "[INFO] Commande : %s\n", command)
	if target.Kind == ExecutableCommandTargetConnector || target.Kind == ExecutableCommandTargetAgent || target.Kind == ExecutableCommandTargetAdaptor {
		if err := ensureGXFrontListening(paths, writer); err != nil {
			return err
		}
	}
	if err := openConsoleRaw(target.WorkDir, "V10 Lab - "+string(target.Kind)+" "+target.Name, target.ExePath, command); err != nil {
		return err
	}
	fmt.Fprintln(writer, "[INFO] Console de commande exécutable ouverte.")
	return nil
}

func ResolveExecutableCommandTarget(paths GedixPaths, product ProductDefinition, config Config, kind ExecutableCommandTargetKind, name string) (ExecutableCommandTarget, error) {
	name = strings.TrimSpace(name)
	switch kind {
	case ExecutableCommandTargetRoot:
		return resolveRootExecutableCommandTarget(paths, name)
	case ExecutableCommandTargetService:
		return resolveServiceExecutableCommandTarget(paths, product, config, name)
	case ExecutableCommandTargetConnector, ExecutableCommandTargetAgent, ExecutableCommandTargetAdaptor:
		return resolveUnitExecutableCommandTarget(paths, product, config, kind, name)
	default:
		return ExecutableCommandTarget{}, fmt.Errorf("type de cible inconnu: %s", kind)
	}
}

func resolveRootExecutableCommandTarget(paths GedixPaths, name string) (ExecutableCommandTarget, error) {
	var exePath string
	switch {
	case strings.EqualFold(name, "gx.exe"):
		name = "gx.exe"
		exePath = paths.GxExePath
	case strings.EqualFold(name, "gx-front.exe"):
		name = "gx-front.exe"
		exePath = paths.FrontExePath
	default:
		return ExecutableCommandTarget{}, fmt.Errorf("exécutable général inconnu: %s", name)
	}
	if err := ensureExecutableFile(exePath); err != nil {
		return ExecutableCommandTarget{}, err
	}
	return ExecutableCommandTarget{Kind: ExecutableCommandTargetRoot, Name: name, ExePath: exePath, WorkDir: paths.GedixRoot}, nil
}

func resolveServiceExecutableCommandTarget(paths GedixPaths, product ProductDefinition, config Config, name string) (ExecutableCommandTarget, error) {
	if _, ok := product.Service(name); !ok {
		return ExecutableCommandTarget{}, fmt.Errorf("service %s introuvable pour le produit %s", name, product.ID)
	}
	if _, ok := config.GedixConfig.Services[name]; !ok {
		return ExecutableCommandTarget{}, fmt.Errorf("service %s absent de la configuration", name)
	}
	debugTarget, err := DetectDebugTargetForProductConfig(paths, name, product, config)
	if err != nil {
		return ExecutableCommandTarget{}, err
	}
	if debugTarget.Kind != DebugTargetService {
		return ExecutableCommandTarget{}, fmt.Errorf("cible %s n'est pas un service", name)
	}
	if err := ensureExecutableFile(debugTarget.ExePath); err != nil {
		return ExecutableCommandTarget{}, err
	}
	return ExecutableCommandTarget{Kind: ExecutableCommandTargetService, Name: debugTarget.Name, ExePath: debugTarget.ExePath, WorkDir: debugTarget.WorkDir}, nil
}

func resolveUnitExecutableCommandTarget(paths GedixPaths, product ProductDefinition, config Config, kind ExecutableCommandTargetKind, name string) (ExecutableCommandTarget, error) {
	family, unit, ok := ProductUnitFamilyByName(config, name)
	if !ok {
		return ExecutableCommandTarget{}, fmt.Errorf("unite %s introuvable dans la configuration", name)
	}
	if executableKindForUnitKind(family.Definition.Kind) != kind {
		return ExecutableCommandTarget{}, fmt.Errorf("%s n'est pas une cible %s", name, kind)
	}
	moduleName := NormalizeModuleType(unit.Module)
	if moduleName == "" {
		return ExecutableCommandTarget{}, fmt.Errorf("module requis pour %s", name)
	}
	module, err := DetectModuleCommandTarget(paths, product, family.Definition, name, moduleName)
	if err != nil {
		return ExecutableCommandTarget{}, err
	}
	if err := ensureExecutableFile(module.ExePath); err != nil {
		return ExecutableCommandTarget{}, err
	}
	return ExecutableCommandTarget{Kind: kind, Name: module.Name, ExePath: module.ExePath, WorkDir: module.WorkDir}, nil
}

func executableKindForUnitKind(kind UnitKind) ExecutableCommandTargetKind {
	switch kind {
	case UnitKindConnector:
		return ExecutableCommandTargetConnector
	case UnitKindAgent:
		return ExecutableCommandTargetAgent
	case UnitKindAdaptor:
		return ExecutableCommandTargetAdaptor
	default:
		return ""
	}
}

func ensureExecutableFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("exécutable introuvable: %s", path)
	}
	if info.IsDir() {
		return fmt.Errorf("le chemin de l'exécutable désigne un dossier: %s", path)
	}
	return nil
}

func runModuleCommandLegacy(config Config, request ModuleCommandRequest, writer io.Writer) error {
	ApplyDefaults(&config)
	product, err := ProductDefinitionByID(config.Product)
	if err != nil {
		return err
	}
	if !product.SupportsModuleCommand() {
		return fmt.Errorf("Ce produit ne supporte pas les commandes de module connector/agent.")
	}
	unitName := strings.TrimSpace(request.UnitName)
	command := strings.TrimSpace(request.Command)
	if unitName == "" {
		return fmt.Errorf("%s requis", product.PrimaryUnitDefinition().SingularLabel)
	}
	if command == "" {
		return fmt.Errorf("commande module requise")
	}
	if !isSafeModuleCommand(command) {
		return fmt.Errorf("la commande contient des caracteres non autorises")
	}
	family, unit, ok := ProductUnitFamilyByName(config, unitName)
	if !ok {
		return fmt.Errorf("unite %s introuvable dans la configuration", unitName)
	}
	if strings.TrimSpace(family.Definition.ModuleExecutablePattern) == "" {
		return fmt.Errorf("%s %s ne supporte pas les commandes de module", family.Definition.SingularLabel, unitName)
	}
	moduleName := NormalizeModuleType(unit.Module)
	if moduleName == "" {
		return fmt.Errorf("Le module %s %s n’est pas renseigné. Scannez le cfg ou renseignez le module manuellement.", productUnitArticle(product), unitName)
	}
	paths, err := DetectGedixPaths(config)
	if err != nil {
		return err
	}
	module, err := DetectModuleCommandTarget(paths, product, family.Definition, unitName, moduleName)
	if err != nil {
		return err
	}
	fmt.Fprintln(writer, "[INFO] Lancement commande module.")
	fmt.Fprintf(writer, "[INFO] %s : %s\n", titleLabel(family.Definition.SingularLabel), unitName)
	fmt.Fprintf(writer, "[INFO] Module : %s\n", moduleName)
	fmt.Fprintf(writer, "[INFO] Exécutable : %s\n", module.ExePath)
	fmt.Fprintf(writer, "[INFO] Commande : %s\n", command)
	if err := ensureGXFrontListening(paths, writer); err != nil {
		return err
	}
	if err := openConsoleRaw(module.WorkDir, "V10 Lab - module "+unitName, module.ExePath, command); err != nil {
		return err
	}
	fmt.Fprintln(writer, "[INFO] Console ouverte.")
	return nil
}

func productUnitArticle(product ProductDefinition) string {
	if product.UnitKind == UnitKindAgent {
		return "de l’agent"
	}
	return "du connector"
}

func RuntimeDebugLaunchTargets(runtime RuntimeConfig) []string {
	seen := map[string]bool{}
	targets := []string{}
	for _, target := range runtime.DebugTargets {
		target = strings.TrimSpace(target)
		key := strings.ToLower(target)
		if target == "" || seen[key] {
			continue
		}
		seen[key] = true
		targets = append(targets, target)
	}
	for target, arguments := range runtime.DebugTargetFlags {
		target = strings.TrimSpace(target)
		key := strings.ToLower(target)
		if target == "" || customArgumentsForTarget(arguments) == "" || seen[key] {
			continue
		}
		seen[key] = true
		targets = append(targets, target)
	}
	sort.SliceStable(targets, func(i, j int) bool {
		return strings.ToLower(targets[i]) < strings.ToLower(targets[j])
	})
	return targets
}

func debugArgsForTarget(runtime RuntimeConfig, target string) ([]string, error) {
	return buildListenArguments(customArgumentsForTarget(debugTargetFlagsForTarget(runtime, target)), runtimeHasDebugTarget(runtime, target))
}

// buildListenArguments preserves custom arguments while placing the automatic
// debug flags after them. Command-line parsing is only used for process launch.
func buildListenArguments(customArguments string, debug bool) ([]string, error) {
	args := []string{"listen"}
	customArguments = strings.TrimSpace(customArguments)
	if customArguments != "" {
		customArgs, err := splitCommandLine(customArguments)
		if err != nil {
			return nil, err
		}
		args = append(args, customArgs...)
	}
	if debug {
		args = append(args, "--debug", "-v2")
	}
	return args, nil
}

func runtimeHasDebugTarget(runtime RuntimeConfig, target string) bool {
	for _, item := range runtime.DebugTargets {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}

func debugTargetFlagsForTarget(runtime RuntimeConfig, target string) []string {
	for key, flags := range runtime.DebugTargetFlags {
		if strings.EqualFold(strings.TrimSpace(key), strings.TrimSpace(target)) {
			return flags
		}
	}
	return nil
}

func customArgumentsForTarget(arguments []string) string {
	parts := make([]string, 0, len(arguments))
	for _, arguments := range arguments {
		arguments = strings.TrimSpace(arguments)
		if arguments != "" {
			parts = append(parts, arguments)
		}
	}
	return strings.Join(parts, " ")
}

func DetectModuleCommandTarget(paths GedixPaths, product ProductDefinition, definition ProductUnitDefinition, unitName string, moduleName string) (DebugTarget, error) {
	moduleName = NormalizeModuleType(moduleName)
	if moduleName == "" {
		return DebugTarget{}, fmt.Errorf("module requis pour %s", unitName)
	}
	moduleExe := filepath.Join(paths.AppPath, unitName, ResolveUnitModuleExecutable(product, definition, unitName, ProductUnitConfig{Module: moduleName}))
	if info, err := os.Stat(moduleExe); err == nil && !info.IsDir() {
		kind := DebugTargetConnector
		if definition.Kind == UnitKindAgent {
			kind = DebugTargetAgent
		} else if definition.Kind == UnitKindAdaptor {
			kind = DebugTargetAdaptor
		}
		return DebugTarget{Name: unitName, Kind: kind, WorkDir: filepath.Join(paths.AppPath, unitName), ExePath: moduleExe}, nil
	}
	return DebugTarget{}, fmt.Errorf("Module introuvable : %s", moduleExe)
}

func isSafeModuleCommand(command string) bool {
	command = strings.TrimSpace(command)
	return command != "" && !strings.ContainsAny(command, "&|><")
}

func ensureGXFrontListening(paths GedixPaths, writer io.Writer) error {
	fmt.Fprintln(writer, "[INFO] Vérification gx-front pour la maquette.")
	running, err := isGXFrontRunning(paths.FrontExePath)
	if err != nil {
		fmt.Fprintf(writer, "[WARN] Détection gx-front impossible : %v\n", err)
	}
	if running {
		fmt.Fprintln(writer, "[INFO] gx-front déjà lancé.")
		return nil
	}
	fmt.Fprintln(writer, "[INFO] gx-front non détecté, démarrage de gx-front.")
	fmt.Fprintf(writer, "[INFO] Démarrage gx-front : %s\n", consoleCommandLine(paths.FrontExePath, "listen"))
	if err := openConsole(paths.GedixRoot, "V10 Lab gx-front", paths.FrontExePath, "listen"); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	return nil
}

func isGXFrontRunning(frontExePath string) (bool, error) {
	if runtime.GOOS != "windows" {
		return false, nil
	}
	script := gxFrontDetectionPowerShell(frontExePath)
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(string(output)), "true"), nil
}

func gxFrontDetectionPowerShell(frontExePath string) string {
	escaped := strings.ReplaceAll(frontExePath, "'", "''")
	return fmt.Sprintf(`$target = '%s'; $found = $false; Get-CimInstance Win32_Process -Filter "Name = 'gx-front.exe'" | ForEach-Object { if ($_.ExecutablePath -eq $target -or ($_.CommandLine -like ('*' + $target + '*'))) { $found = $true } }; if ($found) { 'true' } else { 'false' }`, escaped)
}

func titleLabel(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
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
	cmd := newConsoleCommand(dir, title, exe, args...)
	return cmd.Start()
}

func openConsoleRaw(dir string, title string, exe string, rawArgs string) error {
	commandLine := consoleCommandLineRaw(exe, rawArgs)
	if runtime.GOOS != "windows" {
		fmt.Printf("[DRY-RUN non-windows] cd %s && %s\n", quoteCmdArg(dir), commandLine)
		return nil
	}
	args, err := splitCommandLine(rawArgs)
	if err != nil {
		return err
	}
	return openConsole(dir, title, exe, args...)
}

func newConsoleCommand(dir string, title string, exe string, args ...string) *exec.Cmd {
	if strings.TrimSpace(title) == "" {
		title = "V10 Lab"
	}
	cmdArgs := []string{
		"/D",
		"/C",
		"start",
		title,
		"/D",
		dir,
		"cmd.exe",
		"/D",
		"/K",
		"call",
		exe,
	}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command("cmd.exe", cmdArgs...)
	cmd.Dir = dir
	return cmd
}

func splitCommandLine(value string) ([]string, error) {
	args := []string{}
	var builder strings.Builder
	inQuotes := false
	for _, char := range value {
		switch {
		case char == '"':
			inQuotes = !inQuotes
		case (char == ' ' || char == '\t') && !inQuotes:
			if builder.Len() > 0 {
				args = append(args, builder.String())
				builder.Reset()
			}
		default:
			builder.WriteRune(char)
		}
	}
	if inQuotes {
		return nil, fmt.Errorf("commande exécutable invalide: guillemet non fermé")
	}
	if builder.Len() > 0 {
		args = append(args, builder.String())
	}
	return args, nil
}

func consoleCommandLine(exe string, args ...string) string {
	parts := []string{quoteBatchPath(exe)}
	for _, arg := range args {
		parts = append(parts, quoteBatchArg(arg))
	}
	return strings.Join(parts, " ")
}

func consoleCommandLineRaw(exe string, rawArgs string) string {
	commandLine := quoteBatchPath(exe)
	rawArgs = strings.TrimSpace(rawArgs)
	if rawArgs != "" {
		commandLine += " " + rawArgs
	}
	return commandLine
}

func quoteBatchPath(path string) string {
	if strings.HasPrefix(path, `"`) && strings.HasSuffix(path, `"`) {
		return path
	}
	return `"` + strings.ReplaceAll(path, `"`, `""`) + `"`
}

func quoteCmdArg(value string) string {
	return quoteBatchArg(value)
}

func quoteBatchArg(value string) string {
	if strings.ContainsAny(value, " \t&|<>()[]{}^=;!'`~") {
		return quoteBatchPath(value)
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

func checkCreateTargetAvailable(target string, overwrite bool) error {
	if strings.TrimSpace(target) == "" {
		return fmt.Errorf("targetPath vide")
	}
	if overwrite {
		return nil
	}
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("le dossier cible existe déjà: %s (overwrite=false)", target)
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func ValidateReleaseZipPath(zipPath string) error {
	zipPath = strings.TrimSpace(zipPath)
	if zipPath == "" {
		return fmt.Errorf("Sélectionnez un ZIP de release avant de créer ou mettre à jour la maquette.")
	}
	info, err := os.Stat(zipPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("Le ZIP de release est introuvable : %q.", zipPath)
		}
		return fmt.Errorf("Le ZIP de release est inaccessible : %q: %w", zipPath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("Le chemin du ZIP de release désigne un dossier : %q.", zipPath)
	}
	if !strings.EqualFold(filepath.Ext(zipPath), ".zip") {
		return fmt.Errorf("Le fichier sélectionné doit être un ZIP avec l'extension .zip.")
	}
	return nil
}

func validateUpdateEnvInputs(zipPath string, target string) error {
	if err := ValidateReleaseZipPath(zipPath); err != nil {
		return err
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
