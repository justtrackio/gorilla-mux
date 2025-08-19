package mux_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"
)

func ExampleCORSMethodMiddleware() {
	r := mux.NewRouter()

	r.HandleFunc("/foo", func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder mux.Binder) error {
		// Handle the request
		return nil
	}).Methods(http.MethodGet, http.MethodPut, http.MethodPatch)
	r.HandleFunc("/foo", func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder mux.Binder) error {
		w.Header().Set("Access-Control-Allow-Origin", "http://example.com")
		w.Header().Set("Access-Control-Max-Age", "86400")
		return nil
	}).Methods(http.MethodOptions)

	r.Use(mux.CORSMethodMiddleware(r))

	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/foo", nil)                 // needs to be OPTIONS
	req.Header.Set("Access-Control-Request-Method", "POST")           // needs to be non-empty
	req.Header.Set("Access-Control-Request-Headers", "Authorization") // needs to be non-empty
	req.Header.Set("Origin", "http://example.com")                    // needs to be non-empty

	r.ServeHTTP(context.Background(), rw, req, nil)

	fmt.Println(rw.Header().Get("Access-Control-Allow-Methods"))
	fmt.Println(rw.Header().Get("Access-Control-Allow-Origin"))
	// Output:
	// GET,PUT,PATCH,OPTIONS
	// http://example.com
}
