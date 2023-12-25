package application

import (
	"os"
	"os/exec"
	"path/filepath"
)

type Application struct {
	ID   int
	Name string
	Exec Exec
}

type Exec struct {
	WorkingDirectory string `yaml:"workingDirectory"`
	Path             string
	Args             []string
	RedirectPath     string `yaml:"redirectPath"`
}

func (a Application) AbsolutePath() string {
	exe := filepath.Clean(a.Exec.Path)
	if filepath.IsAbs(exe) {
		return exe
	}
	ret := filepath.Join(a.Exec.WorkingDirectory, exe)
	if _, err := os.Stat(ret); os.IsNotExist(err) {
		if p, e := exec.LookPath(exe); e == nil {
			// For case if not in WorkingDirectory but in $PATH
			return p
		}
	}
	return ret
}

func (a Application) AbsoluteRedirectPath() string {
	if filepath.IsAbs(a.Exec.RedirectPath) {
		return filepath.Clean(a.Exec.RedirectPath)
	}
	return filepath.Clean(filepath.Join(a.Exec.WorkingDirectory, a.Exec.RedirectPath))
}
