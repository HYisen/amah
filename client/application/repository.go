package application

import (
	_ "embed"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

type Repository struct {
	configFilePath string
	pd             atomic.Pointer[[]Application]
	stat           ConfigFileStat
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
	if _, err := ret.Reload(); err != nil {
		return nil, err
	}
	return &ret, nil
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

type ReloadResult struct {
	Before ConfigFileStat
	After  ConfigFileStat
}

type ConfigFileStat struct {
	ModifiedTime time.Time
	Size         int
	ItemCount    int
}

func (r *Repository) Reload() (*ReloadResult, error) {
	file, err := os.ReadFile(r.configFilePath)
	if err != nil {
		return nil, err
	}
	apps, err := parse(strings.NewReader(string(file)))
	if err != nil {
		return nil, err
	}
	r.pd.Store(&apps)

	// As the previous read succeeded, expect no error here.
	stat, _ := os.Stat(r.configFilePath)
	neo := ConfigFileStat{
		ModifiedTime: stat.ModTime(),
		Size:         int(stat.Size()),
		ItemCount:    len(apps),
	}
	ret := &ReloadResult{
		Before: r.stat,
		After:  neo,
	}
	r.stat = neo
	return ret, nil
}
