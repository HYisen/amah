package controller

import (
	"amah/monitor"
	"amah/service/auth"
	"context"
	"encoding/json"
	"net/http"
	"reflect"
)

type Controller struct {
	authService *auth.Service
	web         *Web
}

func New(authService *auth.Service) *Controller {
	ret := &Controller{authService: authService}
	v1PostSession := &ClosureHandler{
		Matcher: Exact(http.MethodPost, "/v1/session"),
		Parser:  JSONParser(reflect.TypeOf(LoginInfo{})),
		Handler: func(ctx context.Context, req any) (rsp any, codedError *CodedError) {
			return ret.Login(ctx, req.(*LoginInfo))
		},
		Formatter:   json.Marshal,
		ContentType: "application/json; charset=utf-8",
	}
	v1GetApplications := &ClosureHandler{
		Matcher: Exact(http.MethodGet, "/v1/applications"),
		Parser:  ParseEmpty,
		Handler: func(ctx context.Context, req any) (rsp any, codedError *CodedError) {
			return ret.GetApplications(ctx)
		},
		Formatter:   json.Marshal,
		ContentType: "application/json; charset=utf-8",
	}
	ret.web = NewWeb(v1PostSession, v1GetApplications)
	return ret
}

type LoginInfo struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (c *Controller) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
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

func (c *Controller) GetApplications(_ context.Context) ([]monitor.Application, *CodedError) {
	applications, err := monitor.Scan()
	if err != nil {
		return nil, NewCodedError(http.StatusInternalServerError, err)
	}
	return applications, nil
}
