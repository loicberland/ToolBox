package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const moduleLine = "module toolBox"
const packageOutputName = "toolbox-package"

type builder struct {
	root string
}

func main() {
	args, rootFlag, err := parseArgs(os.Args[1:])
	if err != nil {
		exitErr(err)
	}

	root, err := resolveRoot(rootFlag)
	if err != nil {
		exitErr(err)
	}

	b := builder{root: root}
	target := "help"
	if len(args) > 0 {
		target = args[0]
	}

	switch target {
	case "api":
		err = b.buildAPI()
	case "web":
		err = b.buildWeb()
	case "web-server":
		err = b.buildWebServer()
	case "module":
		if len(args) < 2 {
			err = errors.New("missing module name")
			break
		}
		err = b.buildModule(args[1])
	case "modules":
		err = b.buildModules()
	case "installer", "package":
		err = b.buildInstaller()
	case "all":
		err = b.buildAll()
	case "help", "-h", "--help":
		printHelp()
	default:
		err = fmt.Errorf("unknown target %q", target)
		printHelp()
	}
	if err != nil {
		exitErr(err)
	}
}

func parseArgs(args []string) ([]string, string, error) {
	clean := make([]string, 0, len(args))
	var root string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--root":
			if i+1 >= len(args) {
				return nil, "", errors.New("--root requires a path")
			}
			root = args[i+1]
			i++
		case strings.HasPrefix(arg, "--root="):
			root = strings.TrimPrefix(arg, "--root=")
			if root == "" {
				return nil, "", errors.New("--root requires a path")
			}
		default:
			clean = append(clean, arg)
		}
	}
	return clean, root, nil
}

func printHelp() {
	fmt.Println("Usage: go run ./tools/build [--root <path>] [api|web|web-server|module <name>|modules|installer|package|all|help]")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run ./tools/build all")
	fmt.Println("  go run ./tools/build api")
	fmt.Println("  go run ./tools/build web")
	fmt.Println("  go run ./tools/build web-server")
	fmt.Println("  go run ./tools/build module test-sheet")
	fmt.Println("  go run ./tools/build modules")
	fmt.Println("  go run ./tools/build installer")
}

func resolveRoot(rootFlag string) (string, error) {
	if rootFlag != "" {
		return validateRoot(rootFlag)
	}
	if envRoot := os.Getenv("TOOLBOX_ROOT"); envRoot != "" {
		return validateRoot(envRoot)
	}
	if cwd, err := os.Getwd(); err == nil {
		if root, err := findRootUpward(cwd); err == nil {
			return root, nil
		}
	}
	if exe, err := os.Executable(); err == nil {
		if root, err := findRootUpward(filepath.Dir(exe)); err == nil {
			return root, nil
		}
	}
	return "", errors.New("could not find project root containing go.mod with module toolBox")
}

func validateRoot(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("invalid root %s: %w", abs, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("root is not a directory: %s", abs)
	}
	goMod := filepath.Join(abs, "go.mod")
	content, err := os.ReadFile(goMod)
	if err != nil {
		return "", fmt.Errorf("root %s does not contain go.mod: %w", abs, err)
	}
	for _, line := range strings.Split(string(content), "\n") {
		if strings.TrimSpace(line) == moduleLine {
			return abs, nil
		}
	}
	return "", fmt.Errorf("go.mod in %s does not declare %q", abs, moduleLine)
}

func findRootUpward(start string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if root, err := validateRoot(current); err == nil {
			return root, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", errors.New("project root not found")
}

func (b builder) buildAll() error {
	if err := b.buildAPI(); err != nil {
		return err
	}
	if err := b.buildWebServer(); err != nil {
		return err
	}
	return b.buildModules()
}

func (b builder) buildInstaller() error {
	if err := b.buildWebServer(); err != nil {
		return err
	}
	if err := b.buildAPI(); err != nil {
		return err
	}
	if err := b.buildModules(); err != nil {
		return err
	}

	payloadDir := filepath.Join(b.root, "apps", "installer", "cmd", "toolbox-setup", "payload")
	payloadRoot := filepath.Join(payloadDir, "ToolBox")
	payloadZip := filepath.Join(payloadDir, "payload.zip")
	if err := resetInstallerPayload(payloadDir); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(b.root, "_build", executableName("api-toolbox")), filepath.Join(payloadRoot, executableName("api-toolbox"))); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(b.root, "_build", executableName("web-server-toolbox")), filepath.Join(payloadRoot, executableName("web-server-toolbox"))); err != nil {
		return err
	}
	for _, name := range []string{"test-sheet", "v10-lab"} {
		if err := copyFile(
			filepath.Join(b.root, "_build", executableName(name)),
			filepath.Join(payloadRoot, "modules", name, executableName(name)),
		); err != nil {
			return err
		}
	}

	if err := verifyInstallerPayload(payloadRoot); err != nil {
		return err
	}
	if err := zipDir(payloadRoot, payloadZip); err != nil {
		return err
	}
	if err := b.goBuildInstaller(); err != nil {
		return err
	}
	if err := appendInstallerPayload(filepath.Join(b.root, "_build", executableName(packageOutputName)), payloadZip); err != nil {
		return err
	}
	if err := resetInstallerPayload(payloadDir); err != nil {
		return err
	}
	for _, name := range []string{"api-toolbox", "web-server-toolbox", "test-sheet", "v10-lab"} {
		if err := os.Remove(filepath.Join(b.root, "_build", executableName(name))); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return cleanBuildDirForInstaller(filepath.Join(b.root, "_build"), executableName(packageOutputName))
}

func (b builder) buildAPI() error {
	return b.goBuild("api", "api-toolbox", filepath.Join("apps", "api", "cmd", "api"), true)
}

func (b builder) buildWeb() error {
	webDir := filepath.Join(b.root, "apps", "web")
	if _, err := os.Stat(filepath.Join(webDir, "node_modules")); os.IsNotExist(err) {
		if err := runCommand(webDir, nil, npmCommand(), "install"); err != nil {
			return fmt.Errorf("web npm install failed: %w", err)
		}
	}
	if err := runCommand(webDir, nil, npmCommand(), "run", "build"); err != nil {
		return fmt.Errorf("web build failed: %w", err)
	}
	return nil
}

func (b builder) buildWebServer() error {
	if err := b.buildWeb(); err != nil {
		return err
	}
	source := filepath.Join(b.root, "apps", "web", "dist")
	if info, err := os.Stat(source); err != nil || !info.IsDir() {
		if err == nil {
			err = errors.New("not a directory")
		}
		return fmt.Errorf("web dist not found at %s: %w", source, err)
	}
	target := filepath.Join(b.root, "apps", "web-server", "cmd", "dist")
	fmt.Printf("Removing %s\n", target)
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("remove web-server dist failed: %w", err)
	}
	fmt.Printf("Copying %s -> %s\n", source, target)
	if err := copyDir(source, target); err != nil {
		return fmt.Errorf("copy web dist failed: %w", err)
	}
	return b.goBuild("web-server", "web-server-toolbox", filepath.Join("apps", "web-server", "cmd", "web-server"), false)
}

func (b builder) buildModules() error {
	for _, name := range []string{"test-sheet", "v10-lab"} {
		if err := b.buildModule(name); err != nil {
			return err
		}
	}
	return nil
}

func (b builder) buildModule(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("module name is required")
	}
	moduleDir := filepath.Join(b.root, "modules", name)
	if info, err := os.Stat(moduleDir); err != nil || !info.IsDir() {
		if err == nil {
			err = errors.New("not a directory")
		}
		return fmt.Errorf("module %q not found at %s: %w", name, moduleDir, err)
	}
	cmdRel := filepath.Join("modules", name, "cmd", name)
	cmdDir := filepath.Join(b.root, cmdRel)
	if info, err := os.Stat(cmdDir); err != nil || !info.IsDir() {
		if err == nil {
			err = errors.New("not a directory")
		}
		return fmt.Errorf("module command for %q not found at %s: %w", name, cmdDir, err)
	}
	return b.goBuild("module "+name, name, cmdRel, true)
}

func (b builder) goBuild(label, outputName, packageRel string, cgo bool) error {
	if err := os.MkdirAll(filepath.Join(b.root, "_build"), 0755); err != nil {
		return fmt.Errorf("create _build failed: %w", err)
	}
	output := filepath.Join(b.root, "_build", executableName(outputName))
	args := []string{"build", "-ldflags", buildLDFlags(b.root), "-o", output, "./" + filepath.ToSlash(packageRel)}
	env := []string{}
	if cgo {
		env = append(env, "CGO_ENABLED=1")
	}
	if err := runCommand(b.root, env, "go", args...); err != nil {
		return fmt.Errorf("%s build failed: %w", label, err)
	}
	return nil
}

func buildLDFlags(root string) string {
	return fmt.Sprintf(
		"-X toolBox/pkg/toolboxversion.Commit=%s -X toolBox/pkg/toolboxversion.BuildDate=%s",
		gitCommit(root),
		buildDate(),
	)
}

func gitCommit(root string) string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	commit := strings.TrimSpace(string(output))
	if commit == "" {
		return "unknown"
	}
	return commit
}

func buildDate() string {
	return time.Now().Format("2006-01-02T15:04:05")
}

func (b builder) goBuildInstaller() error {
	if err := os.MkdirAll(filepath.Join(b.root, "_build"), 0755); err != nil {
		return fmt.Errorf("create _build failed: %w", err)
	}
	output := filepath.Join(b.root, "_build", executableName(packageOutputName))
	args := []string{"build", "-a", "-o", output, "./" + filepath.ToSlash(filepath.Join("apps", "installer", "cmd", "toolbox-setup"))}
	if err := runCommand(b.root, []string{"CGO_ENABLED=0"}, "go", args...); err != nil {
		return fmt.Errorf("installer build failed: %w", err)
	}
	return nil
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func npmCommand() string {
	if runtime.GOOS == "windows" {
		return "npm.cmd"
	}
	return "npm"
}

func runCommand(dir string, extraEnv []string, name string, args ...string) error {
	fmt.Printf("Running in %s: %s %s\n", dir, name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, extraEnv...)
	return cmd.Run()
}

func copyDir(source, target string) error {
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

func copyFile(source, target string) error {
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

func resetInstallerPayload(payloadDir string) error {
	if err := os.RemoveAll(filepath.Join(payloadDir, "ToolBox")); err != nil {
		return err
	}
	if err := os.MkdirAll(payloadDir, 0755); err != nil {
		return err
	}
	return createEmptyZip(filepath.Join(payloadDir, "payload.zip"))
}

func verifyInstallerPayload(payloadRoot string) error {
	for _, rel := range []string{
		executableName("api-toolbox"),
		executableName("web-server-toolbox"),
		filepath.Join("modules", "test-sheet", executableName("test-sheet")),
		filepath.Join("modules", "v10-lab", executableName("v10-lab")),
	} {
		path := filepath.Join(payloadRoot, rel)
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("installer payload missing %s: %w", rel, err)
		}
		if info.Size() == 0 {
			return fmt.Errorf("installer payload file is empty: %s", rel)
		}
	}
	return nil
}

func zipDir(sourceDir, targetZip string) error {
	if err := os.MkdirAll(filepath.Dir(targetZip), 0755); err != nil {
		return err
	}
	out, err := os.Create(targetZip)
	if err != nil {
		return err
	}
	defer out.Close()

	archive := zip.NewWriter(out)
	defer archive.Close()

	return filepath.WalkDir(sourceDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		header.Method = zip.Store
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(writer, in)
		return err
	})
}

func createEmptyZip(path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	archive := zip.NewWriter(out)
	if err := archive.Close(); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func appendInstallerPayload(installerPath, payloadZip string) error {
	payload, err := os.ReadFile(payloadZip)
	if err != nil {
		return err
	}
	out, err := os.OpenFile(installerPath, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := out.Write([]byte("TOOLBOX_PACKAGE_PAYLOAD_V1")); err != nil {
		return err
	}
	_, err = out.Write(payload)
	return err
}

func cleanBuildDirForInstaller(buildDir, installerName string) error {
	entries, err := os.ReadDir(buildDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() == installerName {
			continue
		}
		if err := os.RemoveAll(filepath.Join(buildDir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, "Build failed:", err)
	os.Exit(1)
}
