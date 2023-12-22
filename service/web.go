package service

import (
	"amah/client/application"
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
	authClient            *auth.Client
	monitorClient         *monitor.Client
	applicationRepository *application.Repository
	web                   *Web
}

func New(
	authClient *auth.Client,
	monitorClient *monitor.Client,
	applicationRepository *application.Repository,
) *Service {
	ret := &Service{
		authClient:            authClient,
		monitorClient:         monitorClient,
		applicationRepository: applicationRepository,
		web:                   nil,
	}
	v1PostSession := NewJSONHandler(
		Exact(http.MethodPost, "/v1/session"),
		reflect.TypeOf(LoginInfo{}),
		func(ctx context.Context, req any) (rsp any, codedError *CodedError) {
			return ret.Login(ctx, req.(*LoginInfo))
		},
	)
	v1GetProcesses := NewJSONHandler(
		Exact(http.MethodGet, "/v1/processes"),
		reflect.TypeOf(Empty{}),
		func(ctx context.Context, req any) (rsp any, codedError *CodedError) {
			return ret.GetProcesses(ctx)
		},
	)
	v1DeleteProcess := &ClosureHandler{
		Matcher: func(req *http.Request) bool {
			if req.Method != http.MethodDelete {
				return false
			}
			id, found := strings.CutPrefix(req.URL.Path, "/v1/processes/")
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
			return nil, ret.DeleteProcess(ctx, req.(int))
		},
		Formatter:   FormatEmpty,
		ContentType: http.DetectContentType(nil),
	}
	v1GetApplications := NewJSONHandler(
		Exact(http.MethodGet, "/v1/applications"),
		reflect.TypeOf(Empty{}),
		func(ctx context.Context, req any) (rsp any, codedError *CodedError) {
			return ret.GetApplications(ctx)
		},
	)
	ret.web = NewWeb(v1PostSession, v1GetProcesses, v1DeleteProcess, v1GetApplications)
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

// authenticate auth by ctx and return an 403 *CodedError if fails.
// If logHeaderNullable not empty string, a log with the header name and username would happen.
func (s *Service) authenticate(ctx context.Context, logHeaderNullable string) *CodedError {
	tokenID := DetachToken(ctx)
	t, ok := s.authClient.FindValidToken(tokenID)
	if !ok {
		return NewCodedErrorf(http.StatusForbidden, "invalid token on id %v", tokenID)
	}

	if logHeaderNullable != "" {
		slog.Info(logHeaderNullable, "user", t.Username)
	}
	return nil
}

func (s *Service) GetProcesses(ctx context.Context) ([]monitor.Process, *CodedError) {
	if err := s.authenticate(ctx, ""); err != nil {
		return nil, err
	}

	processes, err := s.monitorClient.Scan()
	if err != nil {
		return nil, NewCodedError(http.StatusInternalServerError, err)
	}
	return processes, nil
}

func (s *Service) DeleteProcess(ctx context.Context, pid int) *CodedError {
	if err := s.authenticate(ctx, "DeleteProcess"); err != nil {
		return err
	}

	found, err := s.monitorClient.Kill(pid)
	if err != nil {
		return NewCodedError(http.StatusInternalServerError, err)
	}
	if !found {
		return NewCodedErrorf(http.StatusNotFound, "no process on pid %d", pid)
	}
	return nil
}

func (s *Service) GetApplications(ctx context.Context) ([]ApplicationComplex, *CodedError) {
	if err := s.authenticate(ctx, ""); err != nil {
		return nil, err
	}
	applications := s.applicationRepository.FindAll()
	processes, err := s.monitorClient.Scan()
	if err != nil {
		return nil, NewCodedError(http.StatusInternalServerError, err)
	}
	return CombineTheoryAndReality(applications, processes), nil
}
