package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const moduleLine = "module toolBox"

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
	fmt.Println("Usage: go run ./tools/build [--root <path>] [api|web|web-server|module <name>|modules|all|help]")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run ./tools/build all")
	fmt.Println("  go run ./tools/build api")
	fmt.Println("  go run ./tools/build web")
	fmt.Println("  go run ./tools/build web-server")
	fmt.Println("  go run ./tools/build module test-sheet")
	fmt.Println("  go run ./tools/build modules")
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
	for _, name := range []string{"test-sheet", "test-env"} {
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
	args := []string{"build", "-o", output, "./" + filepath.ToSlash(packageRel)}
	env := []string{}
	if cgo {
		env = append(env, "CGO_ENABLED=1")
	}
	if err := runCommand(b.root, env, "go", args...); err != nil {
		return fmt.Errorf("%s build failed: %w", label, err)
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

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, "Build failed:", err)
	os.Exit(1)
}
