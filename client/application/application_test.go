package application

import (
	"reflect"
	"strings"
	"testing"
)

type Exec struct {
	WorkingDirectory string
	Path             string
	Args             []string
}

func convert(raw Exec) struct {
	WorkingDirectory string `yaml:"workingDirectory"`
	Path             string
	Args             []string
} {
	return struct {
		WorkingDirectory string `yaml:"workingDirectory"`
		Path             string
		Args             []string
	}(raw)
}

func TestApplication_AbsolutePath(t *testing.T) {

	tests := []struct {
		name string
		exec Exec
		want string
	}{
		{"$PATH", Exec{
			WorkingDirectory: "/tmp",
			Path:             "top",
			Args:             nil,
		}, "/tmp/top"},
		{"abs", Exec{
			WorkingDirectory: "/tmp",
			Path:             "/usr/bin/top",
			Args:             nil,
		}, "/usr/bin/top"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Application{Exec: convert(tt.exec)}
			if got := a.AbsolutePath(); got != tt.want {
				t.Errorf("AbsolutePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []Application
		wantErr bool
	}{
		{"happy path", `- id: 1000
  name: top
  exec:
    workingDirectory: /tmp
    path: top
    args: [ "-o" ,"%MEM" ]`, []Application{{
			ID:   1000,
			Name: "top",
			Exec: convert(Exec{
				WorkingDirectory: "/tmp",
				Path:             "top",
				Args:             []string{"-o", "%MEM"},
			}),
		}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parse(strings.NewReader(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parse() got = %v, want %v", got, tt.want)
			}
		})
	}
}
