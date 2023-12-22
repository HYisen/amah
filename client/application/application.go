package application

import (
	_ "embed"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

type Application struct {
	ID   int
	Name string
	Exec struct {
		WorkingDirectory string `yaml:"workingDirectory"`
		Path             string
		Args             []string
	}
}

func (a Application) AbsolutePath() string {
	exe := path.Clean(a.Exec.Path)
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

type Repository struct {
	data []Application
}

func parse(r io.Reader) ([]Application, error) {
	var ret []Application
	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
}

//go:embed apps.yaml
var dummyConfigStr string

func NewRepository() (*Repository, error) {
	apps, err := parse(strings.NewReader(dummyConfigStr))
	if err != nil {
		return nil, err
	}
	return &Repository{data: apps}, nil
}

func (r *Repository) FindAll() []Application {
	return r.data
}
