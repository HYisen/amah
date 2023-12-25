package application

import (
	_ "embed"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"strings"
	"sync/atomic"
)

type Repository struct {
	configFilePath string
	pd             atomic.Pointer[[]Application]
}

func parse(r io.Reader) ([]Application, error) {
	var ret []Application
	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func NewRepository(configFilePath string) (*Repository, error) {
	ret := Repository{configFilePath: configFilePath, pd: atomic.Pointer[[]Application]{}}
	return &ret, ret.Reload()
}

func (r *Repository) FindAll() []Application {
	return *r.pd.Load()
}

func (r *Repository) Find(id int) (app Application, ok bool) {
	for _, app := range *r.pd.Load() {
		if app.ID == id {
			return app, true
		}
	}
	return Application{}, false
}

func (r *Repository) Reload() error {
	file, err := os.ReadFile(r.configFilePath)
	if err != nil {
		return err
	}
	apps, err := parse(strings.NewReader(string(file)))
	if err != nil {
		return err
	}
	r.pd.Store(&apps)
	return nil
}
