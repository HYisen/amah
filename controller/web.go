package controller

import (
	"amah/monitor"
	"amah/service/auth"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

type Controller struct {
	authService *auth.Service
}

func New(authService *auth.Service) *Controller {
	return &Controller{authService: authService}
}

type LoginInfo struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (c *Controller) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path == "/v1/session" {
		if request.Method == http.MethodPost {
			all, err := io.ReadAll(request.Body)
			if err != nil {
				slog.Error("unexpected failure on read", "err", err, "req", request)
				writer.WriteHeader(http.StatusInternalServerError)
				return
			}
			var li LoginInfo
			err = json.Unmarshal(all, &li)
			if err != nil {
				slog.Warn("bad login request", "err", err, "req", request)
				writer.WriteHeader(http.StatusBadRequest)
				// Let it go when can not send the optional error info to client, which could be their problem.
				_, _ = writer.Write([]byte(fmt.Sprintf("can not parse req %v as %v", request, err)))
			}
			token, e := c.Login(li.Username, li.Password)
			if e != nil {
				writer.WriteHeader(e.Code)
				_, _ = writer.Write([]byte(e.Err.Error()))
				return
			}
			data, err := json.Marshal(token)
			if err != nil {
				slog.Error("unexpected failure on marshal", "err", err)
				writer.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, _ = writer.Write(data)
			return
		}
	}
	if request.URL.Path == "/v1/applications" && request.Method == http.MethodGet {
		applications, e := c.GetApplications()
		if e != nil {
			writer.WriteHeader(e.Code)
			_, _ = writer.Write([]byte(e.Err.Error()))
			return
		}
		data, err := json.Marshal(applications)
		if err != nil {
			slog.Error("unexpected failure on marshal", "err", err)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = writer.Write(data)
		return
	}
}

func (c *Controller) Login(username, password string) (t *auth.Token, e *CodedError) {
	ok, err := c.authService.Auth(username, password)
	if err != nil {
		return nil, NewCodedError(http.StatusInternalServerError, err)
	}
	if !ok {
		return nil, NewCodedErrorf(http.StatusForbidden, "no password on such username")
	}
	token := c.authService.CreateToken(username)
	return &token, nil
}

func (c *Controller) GetApplications() ([]monitor.Application, *CodedError) {
	applications, err := monitor.Scan()
	if err != nil {
		return nil, NewCodedError(http.StatusInternalServerError, err)
	}
	return applications, nil
}

type CodedError struct {
	Code int
	Err  error
}

func NewCodedError(code int, err error) *CodedError {
	return &CodedError{Code: code, Err: err}
}

func NewCodedErrorf(code int, format string, a ...any) *CodedError {
	return &CodedError{Code: code, Err: fmt.Errorf(format, a...)}
}

func (e CodedError) Error() string {
	return fmt.Sprintf("%d: %v", e.Code, e.Err.Error())
}
