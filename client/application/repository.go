package application

import (
	_ "embed"
	"gopkg.in/yaml.v3"
	"io"
	"strings"
)

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

func (r *Repository) Find(id int) (app Application, ok bool) {
	for _, app := range r.data {
		if app.ID == id {
			return app, true
		}
	}
	return Application{}, false
}
