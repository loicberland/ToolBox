package toolboxruntime

import (
	"os"
	"path/filepath"
)

const (
	EnvRoot           = "TOOLBOX_ROOT"
	EnvModuleID       = "TOOLBOX_MODULE_ID"
	EnvModuleDir      = "TOOLBOX_MODULE_DIR"
	EnvModuleDataDir  = "TOOLBOX_MODULE_DATA_DIR"
	EnvModuleFilesDir = "TOOLBOX_MODULE_FILES_DIR"
)

type Layout struct {
	RootDir string
}

type ModuleLayout struct {
	RootDir  string
	ID       string
	Dir      string
	Exe      string
	DataDir  string
	FilesDir string
}

func ForApp(configPath string) (Layout, error) {
	if configPath != "" {
		abs, err := filepath.Abs(configPath)
		if err != nil {
			return Layout{}, err
		}
		return Layout{RootDir: filepath.Dir(abs)}, nil
	}

	exe, err := os.Executable()
	if err != nil {
		return Layout{}, err
	}
	return Layout{RootDir: filepath.Dir(exe)}, nil
}

func ForModule(moduleID string) (ModuleLayout, error) {
	if root := os.Getenv(EnvRoot); root != "" {
		layout := Layout{RootDir: root}.Module(moduleID)
		if value := os.Getenv(EnvModuleDir); value != "" {
			layout.Dir = value
		}
		if value := os.Getenv(EnvModuleDataDir); value != "" {
			layout.DataDir = value
		}
		if value := os.Getenv(EnvModuleFilesDir); value != "" {
			layout.FilesDir = value
		}
		return layout, nil
	}

	exe, err := os.Executable()
	if err != nil {
		return ModuleLayout{}, err
	}
	moduleDir := filepath.Dir(exe)
	rootDir := filepath.Dir(filepath.Dir(moduleDir))
	return ModuleLayout{
		RootDir:  rootDir,
		ID:       moduleID,
		Dir:      moduleDir,
		Exe:      exe,
		DataDir:  filepath.Join(moduleDir, "data"),
		FilesDir: filepath.Join(moduleDir, "files"),
	}, nil
}

func (l Layout) ConfigPath() string {
	return filepath.Join(l.RootDir, "toolbox.cfg")
}

func (l Layout) ModulesDir() string {
	return filepath.Join(l.RootDir, "modules")
}

func (l Layout) Module(moduleID string) ModuleLayout {
	moduleDir := filepath.Join(l.ModulesDir(), moduleID)
	return ModuleLayout{
		RootDir:  l.RootDir,
		ID:       moduleID,
		Dir:      moduleDir,
		Exe:      filepath.Join(moduleDir, moduleID+".exe"),
		DataDir:  filepath.Join(moduleDir, "data"),
		FilesDir: filepath.Join(moduleDir, "files"),
	}
}

func (m ModuleLayout) Env() []string {
	return []string{
		EnvRoot + "=" + m.RootDir,
		EnvModuleID + "=" + m.ID,
		EnvModuleDir + "=" + m.Dir,
		EnvModuleDataDir + "=" + m.DataDir,
		EnvModuleFilesDir + "=" + m.FilesDir,
	}
}

func (m ModuleLayout) EnsureBaseDirs() error {
	if err := os.MkdirAll(m.DataDir, 0755); err != nil {
		return err
	}
	return os.MkdirAll(m.FilesDir, 0755)
}
