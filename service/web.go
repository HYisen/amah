package service

import (
	"amah/client/application"
	"amah/client/auth"
	"amah/client/monitor"
	"context"
	"encoding/json"
	"errors"
	. "github.com/hyisen/wf"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
	"sync"
)

type Service struct {
	authClient            *auth.Client
	monitorClient         *monitor.Client
	applicationRepository *application.Repository
	appIDToClient         map[int]*application.Client
	mu                    sync.Mutex // guard actions likes exec with scan that shall escape race condition
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
		appIDToClient:         make(map[int]*application.Client),
		mu:                    sync.Mutex{},
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
	v1DeleteProcess := NewClosureHandler(
		ResourceWithID(http.MethodDelete, "/v1/processes/", ""),
		PathIDParser(""),
		func(ctx context.Context, req any) (rsp any, codedError *CodedError) {
			return nil, ret.DeleteProcess(ctx, req.(int))
		},
		FormatEmpty,
		http.DetectContentType(nil),
	)
	v1GetApplications := NewJSONHandler(
		Exact(http.MethodGet, "/v1/applications"),
		reflect.TypeOf(Empty{}),
		func(ctx context.Context, req any) (rsp any, codedError *CodedError) {
			return ret.GetApplications(ctx)
		},
	)
	const v1PutApplicationPathSuffix = "/instances"
	v1PutApplication := NewClosureHandler(
		ResourceWithID(http.MethodPut, "/v1/applications/", v1PutApplicationPathSuffix),
		PathIDParser(v1PutApplicationPathSuffix),
		func(ctx context.Context, req any) (rsp any, codedError *CodedError) {
			return ret.StartApplication(ctx, req.(int))
		},
		json.Marshal,
		JSONContentType,
	)
	v1DeleteApplicationInstanceMatcher, v1DeleteApplicationInstanceParser := ResourceWithIDs(
		http.MethodDelete,
		[]string{"v1", "applications", "", "instances"},
	)
	v1DeleteApplicationInstance := NewClosureHandler(
		v1DeleteApplicationInstanceMatcher,
		v1DeleteApplicationInstanceParser,
		func(ctx context.Context, req any) (rsp any, codedError *CodedError) {
			return nil, ret.KillApplication(ctx, req.([]int)[0])
		},
		FormatEmpty,
		http.DetectContentType(nil),
	)
	v1PutDashboardAppConfigReload := NewJSONHandler(
		Exact(http.MethodPut, "/v1/dashboard/app-config/reload"),
		reflect.TypeOf(Empty{}),
		func(ctx context.Context, req any) (rsp any, codedError *CodedError) {
			return ret.ReloadAppConfig(ctx)
		})
	const v1GetApplicationOutputSuffix = "/output"
	v1GetApplicationOutput := NewClosureHandler(
		ResourceWithID(http.MethodGet, "/v1/applications/", v1GetApplicationOutputSuffix),
		PathIDParser(v1GetApplicationOutputSuffix),
		func(ctx context.Context, req any) (rsp any, codedError *CodedError) {
			return ret.GetApplicationOutput(ctx, req.(int))
		},
		json.Marshal,
		JSONContentType,
	)
	v0Forbidden := NewClosureHandler(
		Exact(http.MethodGet, "/v0/forbidden"),
		ParseEmpty,
		func(_ context.Context, _ any) (_ any, codedError *CodedError) {
			return nil, NewCodedError(http.StatusForbidden, errors.New("no privilege to access this"))
		},
		FormatEmpty,
		JSONContentType,
	)
	ret.web = NewWeb(
		true,
		v1PostSession,
		v1GetProcesses,
		v1DeleteProcess,
		v1GetApplications,
		v1PutApplication,
		v1DeleteApplicationInstance,
		v1PutDashboardAppConfigReload,
		v1GetApplicationOutput,
		v0Forbidden,
	)
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

func (s *Service) findClientByPIDOptional(pid int) (optional *application.Client) {
	// Normal process has a natural number PID.
	if pid <= 0 {
		return nil
	}
	for _, client := range s.appIDToClient {
		if client.PID() == pid {
			return client
		}
	}
	return nil
}

func (s *Service) DeleteProcess(ctx context.Context, pid int) *CodedError {
	if err := s.authenticate(ctx, "DeleteProcess"); err != nil {
		return err
	}

	if client := s.findClientByPIDOptional(pid); client != nil {
		if err := client.Terminate(); err != nil {
			return NewCodedError(http.StatusInternalServerError, err)
		}
		return nil
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

func (s *Service) findApplicationComplex(appID int) (ApplicationComplex, *CodedError) {
	app, ok := s.applicationRepository.Find(appID)
	if !ok {
		return ApplicationComplex{}, NewCodedErrorf(http.StatusNotFound, "no app on id %d", appID)
	}

	processes, err := s.monitorClient.Scan()
	if err != nil {
		return ApplicationComplex{}, NewCodedError(http.StatusInternalServerError, err)
	}

	return CombineTheoryAndReality([]application.Application{app}, processes)[0], nil
}

func (s *Service) StartApplication(ctx context.Context, appID int) (ApplicationComplex, *CodedError) {
	if err := s.authenticate(ctx, "StartApplication "+strconv.Itoa(appID)); err != nil {
		return ApplicationComplex{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// Just prevent concurrent StartApplication,
	// It's user's duty to keep external exec away to achieve atomicity.
	app, err := s.findApplicationComplex(appID)
	if err != nil {
		return ApplicationComplex{}, err
	}

	if len(app.Instances) > 0 {
		return ApplicationComplex{}, NewCodedErrorf(http.StatusConflict, "running duplicates %d", len(app.Instances))
	}

	// I think 1k line is long enough.
	client, e := application.NewClient(app.Application, 1000)
	if e != nil {
		return ApplicationComplex{}, NewCodedError(http.StatusServiceUnavailable, e)
	}
	s.appIDToClient[appID] = client

	app, err = s.findApplicationComplex(appID)
	if err != nil {
		return ApplicationComplex{}, err
	}
	return app, nil
}

func (s *Service) KillApplication(ctx context.Context, appID int) *CodedError {
	if err := s.authenticate(ctx, "KillApplication "+strconv.Itoa(appID)); err != nil {
		return err
	}

	client, ok := s.appIDToClient[appID]
	if !ok {
		return NewCodedErrorf(http.StatusNotFound, "no app on id %d to kill", appID)
	}

	if err := client.Terminate(); err != nil {
		return NewCodedErrorf(http.StatusServiceUnavailable, "failed to kill app %d: %v", appID, err)
	}
	return nil
}

func (s *Service) ReloadAppConfig(ctx context.Context) (*application.ReloadResult, *CodedError) {
	if err := s.authenticate(ctx, ""); err != nil {
		return nil, err
	}
	ret, err := s.applicationRepository.Reload()
	if err != nil {
		return nil, NewCodedError(http.StatusServiceUnavailable, err)
	}
	return ret, nil
}

func (s *Service) GetApplicationOutput(ctx context.Context, appID int) ([]string, *CodedError) {
	if err := s.authenticate(ctx, ""); err != nil {
		return nil, err
	}
	app, ok := s.appIDToClient[appID]
	if !ok {
		return nil, NewCodedErrorf(http.StatusNotFound, "app on not exists id %d", appID)
	}
	return app.Query(), nil
}
