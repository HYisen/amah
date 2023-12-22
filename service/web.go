package service

import (
	"amah/client/auth"
	"amah/client/monitor"
	"context"
	"log/slog"
	"net/http"
	"reflect"
)

type Service struct {
	authClient    *auth.Client
	monitorClient *monitor.Client
	web           *Web
}

func New(authClient *auth.Client, monitorClient *monitor.Client) *Service {
	ret := &Service{
		authClient:    authClient,
		monitorClient: monitorClient,
		web:           nil,
	}
	v1PostSession := NewJSONHandler(
		Exact(http.MethodPost, "/v1/session"),
		reflect.TypeOf(LoginInfo{}),
		func(ctx context.Context, req any) (rsp any, codedError *CodedError) {
			return ret.Login(ctx, req.(*LoginInfo))
		},
	)
	v1GetApplications := NewJSONHandler(
		Exact(http.MethodGet, "/v1/applications"),
		reflect.TypeOf(Empty{}),
		func(ctx context.Context, req any) (rsp any, codedError *CodedError) {
			return ret.GetApplications(ctx)
		},
	)
	ret.web = NewWeb(v1PostSession, v1GetApplications)
	return ret
}

type LoginInfo struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Service) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	s.web.ServeHTTP(writer, request)
}

func (s *Service) Login(_ context.Context, li *LoginInfo) (t *auth.Token, e *CodedError) {
	ok, err := s.authClient.Auth(li.Username, li.Password)
	if err != nil {
		return nil, NewCodedError(http.StatusInternalServerError, err)
	}
	if !ok {
		return nil, NewCodedErrorf(http.StatusForbidden, "no password on such username")
	}
	token := s.authClient.CreateToken(li.Username)
	return &token, nil
}

func (s *Service) GetApplications(ctx context.Context) ([]monitor.Application, *CodedError) {
	tokenID := DetachToken(ctx)
	t, ok := s.authClient.FindValidToken(tokenID)
	if !ok {
		return nil, NewCodedErrorf(http.StatusForbidden, "invalid token on id %v", tokenID)
	}
	slog.Debug("getApplications", "user", t.Username)
	applications, err := s.monitorClient.Scan()
	if err != nil {
		return nil, NewCodedError(http.StatusInternalServerError, err)
	}
	return applications, nil
}
