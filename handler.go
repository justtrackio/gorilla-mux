package mux

import (
	"context"
	"net/http"
	"reflect"
)

type Binder interface {
}

type Handler interface {
	ServeHTTP(ctx context.Context, writer http.ResponseWriter, request *http.Request, binder Binder) error
}

func HandlerToHandlerFunc(handler Handler) HandlerFunc {
	return func(ctx context.Context, writer http.ResponseWriter, request *http.Request, binder Binder) error {
		return handler.ServeHTTP(ctx, writer, request, binder)
	}
}

type HandlerFunc func(ctx context.Context, writer http.ResponseWriter, request *http.Request, binder Binder) error

func (f HandlerFunc) ServeHTTP(ctx context.Context, writer http.ResponseWriter, request *http.Request, binder Binder) error {
	return f(ctx, writer, request, binder)
}

// NotFound replies to the request with an HTTP 404 not found error.
func NotFound(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
	http.Error(w, "404 page not found", http.StatusNotFound)

	return nil
}

// NotFoundHandler returns a simple request handler
// that replies to each request with a “404 page not found” reply.
func NotFoundHandler() HandlerFunc { return NotFound }

func isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice, reflect.Func:
		return reflect.ValueOf(i).IsNil()
	default:
		return false
	}
}
