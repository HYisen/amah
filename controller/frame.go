package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"time"
)

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

type Handler interface {
	Match(req *http.Request) bool
	Parse(data []byte) (any, error)
	Handle(ctx context.Context, req any) (rsp any, codedError *CodedError)
	Format(output any) (data []byte, err error)
}

func Exact(method string, path string) func(req *http.Request) bool {
	return func(req *http.Request) bool {
		return req.URL.Path == path && req.Method == method
	}
}

type ClosureHandler struct {
	Matcher   func(req *http.Request) bool
	Parser    func(data []byte) (any, error)
	Handler   func(ctx context.Context, req any) (rsp any, codedError *CodedError)
	Formatter func(output any) (data []byte, err error)
}

func (ch *ClosureHandler) Match(req *http.Request) bool {
	return ch.Matcher(req)
}

func (ch *ClosureHandler) Parse(data []byte) (any, error) {
	return ch.Parser(data)
}

func (ch *ClosureHandler) Handle(
	ctx context.Context,
	req any,
) (rsp any, codedError *CodedError) {
	return ch.Handler(ctx, req)
}

func (ch *ClosureHandler) Format(output any) (data []byte, err error) {
	return ch.Formatter(output)
}

// Web is a helper to implements http.Handler as mux.
// There was a Handler[RequestType,ResponseType] design,
// which is good as guaranteed type consistency between its methods,
// but failed as it's []any, not []Handler[any,any] that accepts Handler[One,Two],
// and in runtime, the interface conversion from Handler[any,any] to Handler[One,Two] failed.
// Once I drop the type info, it can not come back even through cast.
// The best performance strategy could be a code generator, which is complicated to implements.
// Or just put the dirty transform work together as it was, which causes a lot of redundancy.
type Web struct {
	handlers []Handler
}

func NewWeb(handlers ...Handler) *Web {
	return &Web{handlers: handlers}
}

var serverContextCreator = func() (ctx context.Context, cancel context.CancelFunc) {
	const threshold = 1000 * time.Millisecond
	cause := fmt.Errorf("handler exceed timeout %v", threshold)
	return context.WithTimeoutCause(context.Background(), threshold, cause)
}

func (w *Web) findHandler(req *http.Request) Handler {
	// Maybe a Trie when it's more complicated and the performance difference matters.
	for _, h := range w.handlers {
		if h.Match(req) {
			return h
		}
	}
	return nil
}

// ServeHTTP implements that in interface.
func (w *Web) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	h := w.findHandler(request)
	if h == nil {
		writer.WriteHeader(http.StatusNotAcceptable)
		slog.Warn("unmatched request", "req", request)
		_, _ = writer.Write([]byte(fmt.Sprintf("unsupported request on %v %v", request.Method, request.URL)))
		return
	}

	inputData, err := io.ReadAll(request.Body)
	if err != nil {
		// What if it's client's fault? Maybe warn rather than error?
		slog.Error("unexpected failure on read", "err", err, "req", request)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	input, err := h.Parse(inputData)
	if err != nil {
		slog.Warn("bad input format", "err", err, "req", request)
		writer.WriteHeader(http.StatusBadRequest)
		// Let it go when can not send the optional error info to client, which could be their problem.
		_, _ = writer.Write([]byte(fmt.Sprintf("can not parse req %v as %v", request, err)))
		return
	}

	ctx, cancel := serverContextCreator()
	defer cancel()
	output, e := h.Handle(ctx, input)
	if e != nil {
		writer.WriteHeader(e.Code)
		_, _ = writer.Write([]byte(e.Err.Error()))
		return
	}

	outputData, err := h.Format(output)
	if err != nil {
		slog.Error("unexpected failure on marshal", "err", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Without setting, rely on http.DetectContentType invoked by Write to fulfill the MIME type.
	_, _ = writer.Write(outputData)
}

func JSONParser(clazz reflect.Type) func(data []byte) (any, error) {
	return func(data []byte) (any, error) {
		value := reflect.New(clazz)
		if err := json.Unmarshal(data, value.Interface()); err != nil {
			return value, err
		}
		return value.Interface(), nil
	}
}
