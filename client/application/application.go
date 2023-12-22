package application

import (
	_ "embed"
	"gopkg.in/yaml.v3"
	"io"
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
	if strings.HasPrefix(a.Exec.Path, "/") {
		return a.Exec.Path
	}
	return filepath.Join(a.Exec.WorkingDirectory, a.Exec.Path)
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
