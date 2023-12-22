package service

import (
	"amah/client/auth"
	"amah/client/monitor"
	"context"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
	"strings"
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
	v1DeleteApplication := &ClosureHandler{
		Matcher: func(req *http.Request) bool {
			if req.Method != http.MethodDelete {
				return false
			}
			id, found := strings.CutPrefix(req.URL.Path, "/v1/applications/")
			if !found {
				return false
			}
			if _, err := strconv.Atoi(id); err != nil {
				return false
			}
			return true
		},
		Parser: func(_ []byte, path string) (any, error) {
			// The Matcher shall have guaranteed a valid number here. So we can skip validation here.
			str := path[strings.LastIndexByte(path, '/')+1:]
			num, _ := strconv.Atoi(str)
			return num, nil
		},
		Handler: func(ctx context.Context, req any) (rsp any, codedError *CodedError) {
			return nil, ret.DeleteApplication(ctx, req.(int))
		},
		Formatter:   FormatEmpty,
		ContentType: http.DetectContentType(nil),
	}
	ret.web = NewWeb(v1PostSession, v1GetApplications, v1DeleteApplication)
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
		// the incorrect credential is less sensitive, so just carry it and make it printable in error.
		return nil, NewCodedErrorf(http.StatusForbidden, "no password[%s] on username[%s]", li.Password, li.Username)
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

func (s *Service) DeleteApplication(ctx context.Context, pid int) *CodedError {
	tokenID := DetachToken(ctx)
	t, ok := s.authClient.FindValidToken(tokenID)
	if !ok {
		return NewCodedErrorf(http.StatusForbidden, "invalid token on id %v", tokenID)
	}
	slog.Info("DeleteApplication", "pid", pid, "user", t.Username)

	found, err := s.monitorClient.Kill(pid)
	if err != nil {
		return NewCodedError(http.StatusInternalServerError, err)
	}
	if !found {
		return NewCodedErrorf(http.StatusNotFound, "no process on pid %d", pid)
	}
	return nil
}
