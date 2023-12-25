package application

import (
	"reflect"
	"strings"
	"testing"
)

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
		}, "/usr/bin/top"},
		{"abs", Exec{
			WorkingDirectory: "/tmp",
			Path:             "/usr/bin/top",
			Args:             nil,
		}, "/usr/bin/top"},
		{"not exists", Exec{
			WorkingDirectory: "/tmp",
			Path:             "a-test-file-not-exists",
			Args:             nil,
		}, "/tmp/a-test-file-not-exists"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Application{Exec: tt.exec}
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
			Exec: Exec{
				WorkingDirectory: "/tmp",
				Path:             "top",
				Args:             []string{"-o", "%MEM"},
			},
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
