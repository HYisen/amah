package controller

import (
	"amah/monitor"
	"amah/service/auth"
	"context"
	"log/slog"
	"net/http"
	"reflect"
)

type Controller struct {
	authService *auth.Service
	web         *Web
}

func New(authService *auth.Service) *Controller {
	ret := &Controller{authService: authService}
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

func (c *Controller) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	c.web.ServeHTTP(writer, request)
}

func (c *Controller) Login(_ context.Context, li *LoginInfo) (t *auth.Token, e *CodedError) {
	ok, err := c.authService.Auth(li.Username, li.Password)
	if err != nil {
		return nil, NewCodedError(http.StatusInternalServerError, err)
	}
	if !ok {
		return nil, NewCodedErrorf(http.StatusForbidden, "no password on such username")
	}
	token := c.authService.CreateToken(li.Username)
	return &token, nil
}

func (c *Controller) GetApplications(ctx context.Context) ([]monitor.Application, *CodedError) {
	tokenID := DetachToken(ctx)
	t, ok := c.authService.FindValidToken(tokenID)
	if !ok {
		return nil, NewCodedErrorf(http.StatusForbidden, "invalid token on id %v", tokenID)
	}
	slog.Debug("getApplications", "user", t.Username)
	applications, err := monitor.Scan()
	if err != nil {
		return nil, NewCodedError(http.StatusInternalServerError, err)
	}
	return applications, nil
}
