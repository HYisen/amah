package auth

import (
	"testing"
)

func Test_Encrypt(t *testing.T) {
	tests := []struct {
		name             string
		username         string
		password         string
		registerUsername string
		registerPassword string
		wantPass         bool
	}{
		{"happy pass", "alice", "123456", "alice", "123456", true},
		{"wrong password", "alice", "123456", "alice", "aaa", false},
		{"wrong username", "ben", "123456", "alice", "123456", false},
		{"neither right", "ben", "123456", "alice", "aaa", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, err := Register(tt.registerUsername, tt.registerPassword)
			if err != nil {
				t.Error(err)
				return
			}
			account, err := newAccount(line)
			if err != nil {
				t.Error(err)
				return
			}
			service, err := NewService([]Account{account})
			if err != nil {
				t.Error(err)
				return
			}
			got, err := service.Auth(tt.username, tt.password)
			if err != nil {
				t.Error(err)
				return
			}
			if tt.wantPass != got {
				t.Errorf("Auth got = %v, want %v", got, tt.wantPass)
			}
		})
	}
}
