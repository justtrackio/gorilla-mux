package mux

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"
)

type testMiddleware struct {
	timesCalled uint
}

func (tm *testMiddleware) Middleware(h HandlerFunc) HandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
		tm.timesCalled++

		return h.ServeHTTP(ctx, w, r, binder)
	}
}

func dummyHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, b Binder) error {
	return nil
}

func TestMiddlewareAdd(t *testing.T) {
	router := NewRouter()
	router.HandleFunc("/", dummyHandler).Methods("GET")

	mw := &testMiddleware{}

	router.useInterface(mw)
	if len(router.middlewares) != 1 || router.middlewares[0] != mw {
		t.Fatal("Middleware interface was not added correctly")
	}

	router.Use(mw.Middleware)
	if len(router.middlewares) != 2 {
		t.Fatal("Middleware method was not added correctly")
	}

	banalMw := func(handler HandlerFunc) HandlerFunc {
		return handler
	}
	router.Use(banalMw)
	if len(router.middlewares) != 3 {
		t.Fatal("Middleware function was not added correctly")
	}

	route := router.HandleFunc("/route", dummyHandler)
	route.useInterface(mw)
	if len(route.middlewares) != 1 {
		t.Fatal("Route middleware function was not added correctly")
	}

	route.Use(banalMw)
	if len(route.middlewares) != 2 {
		t.Fatal("Route middleware function was not added correctly")
	}
}

func TestMiddleware(t *testing.T) {
	router := NewRouter()
	router.HandleFunc("/", dummyHandler).Methods("GET")

	mw := &testMiddleware{}
	router.useInterface(mw)

	rw := NewRecorder()
	req := newRequest("GET", "/")

	t.Run("regular middleware call", func(t *testing.T) {
		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}

		if mw.timesCalled != 1 {
			t.Fatalf("Expected %d calls, but got only %d", 1, mw.timesCalled)
		}
	})

	t.Run("not called for 404", func(t *testing.T) {
		req = newRequest("GET", "/not/found")
		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}
		if mw.timesCalled != 1 {
			t.Fatalf("Expected %d calls, but got only %d", 1, mw.timesCalled)
		}
	})

	t.Run("not called for method mismatch", func(t *testing.T) {
		req = newRequest("POST", "/")
		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}
		if mw.timesCalled != 1 {
			t.Fatalf("Expected %d calls, but got only %d", 1, mw.timesCalled)
		}
	})

	t.Run("regular call using function middleware", func(t *testing.T) {
		router.Use(mw.Middleware)
		req = newRequest("GET", "/")
		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}
		if mw.timesCalled != 3 {
			t.Fatalf("Expected %d calls, but got only %d", 3, mw.timesCalled)
		}
	})

	t.Run("regular call using route middleware func", func(t *testing.T) {
		router.HandleFunc("/route", dummyHandler).Use(mw.Middleware)
		req = newRequest("GET", "/route")
		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}
		if mw.timesCalled != 6 {
			t.Fatalf("Expected %d calls, but got only %d", 6, mw.timesCalled)
		}
	})

	t.Run("regular call using route middleware interface", func(t *testing.T) {
		router.HandleFunc("/route", dummyHandler).useInterface(mw)
		req = newRequest("GET", "/route")
		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}
		if mw.timesCalled != 9 {
			t.Fatalf("Expected %d calls, but got only %d", 9, mw.timesCalled)
		}
	})
}

func TestMiddlewareSubrouter(t *testing.T) {
	router := NewRouter()
	router.HandleFunc("/", dummyHandler).Methods("GET")

	subrouter := router.PathPrefix("/sub").Subrouter()
	subrouter.HandleFunc("/x", dummyHandler).Methods("GET")

	mw := &testMiddleware{}
	subrouter.useInterface(mw)

	rw := NewRecorder()
	req := newRequest("GET", "/")

	t.Run("not called for route outside subrouter", func(t *testing.T) {
		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}
		if mw.timesCalled != 0 {
			t.Fatalf("Expected %d calls, but got only %d", 0, mw.timesCalled)
		}
	})

	t.Run("not called for subrouter root 404", func(t *testing.T) {
		req = newRequest("GET", "/sub/")
		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}
		if mw.timesCalled != 0 {
			t.Fatalf("Expected %d calls, but got only %d", 0, mw.timesCalled)
		}
	})

	t.Run("called once for route inside subrouter", func(t *testing.T) {
		req = newRequest("GET", "/sub/x")
		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}
		if mw.timesCalled != 1 {
			t.Fatalf("Expected %d calls, but got only %d", 1, mw.timesCalled)
		}
	})

	t.Run("not called for 404 inside subrouter", func(t *testing.T) {
		req = newRequest("GET", "/sub/not/found")
		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}
		if mw.timesCalled != 1 {
			t.Fatalf("Expected %d calls, but got only %d", 1, mw.timesCalled)
		}
	})

	t.Run("middleware added to router", func(t *testing.T) {
		router.useInterface(mw)

		t.Run("called once for route outside subrouter", func(t *testing.T) {
			req = newRequest("GET", "/")
			if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
				t.Fatalf("Failed to call ServeHTTP: %v", err)
			}
			if mw.timesCalled != 2 {
				t.Fatalf("Expected %d calls, but got only %d", 2, mw.timesCalled)
			}
		})

		t.Run("called twice for route inside subrouter", func(t *testing.T) {
			req = newRequest("GET", "/sub/x")
			if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
				t.Fatalf("Failed to call ServeHTTP: %v", err)
			}
			if mw.timesCalled != 4 {
				t.Fatalf("Expected %d calls, but got only %d", 4, mw.timesCalled)
			}
		})
	})
}

func TestMiddlewareExecution(t *testing.T) {
	mwStr := []byte("Middleware\n")
	handlerStr := []byte("Logic\n")

	handlerFunc := func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
		_, err := w.Write(handlerStr)

		return err
	}

	router := NewRouter()
	router.HandleFunc("/", handlerFunc)

	t.Run("responds normally without middleware", func(t *testing.T) {
		rw := NewRecorder()
		req := newRequest("GET", "/")

		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}

		if !bytes.Equal(rw.Body.Bytes(), handlerStr) {
			t.Fatal("Handler response is not what it should be")
		}
	})

	t.Run("responds with handler and middleware response", func(t *testing.T) {
		rw := NewRecorder()
		req := newRequest("GET", "/")

		router.Use(func(h HandlerFunc) HandlerFunc {
			return func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
				_, err := w.Write(mwStr)
				if err != nil {
					t.Fatalf("Failed writing HTTP response: %v", err)
				}

				return h.ServeHTTP(ctx, w, r, binder)
			}
		})

		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}
		if !bytes.Equal(rw.Body.Bytes(), append(mwStr, handlerStr...)) {
			t.Fatal("Middleware + handler response is not what it should be")
		}
	})

	t.Run("responds with handler, middleware and route middleware response", func(t *testing.T) {
		routeMwStr := []byte("Route Middleware\n")
		rw := NewRecorder()
		req := newRequest("GET", "/route")

		router.HandleFunc("/route", handlerFunc).Use(func(h HandlerFunc) HandlerFunc {
			return func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
				_, err := w.Write(routeMwStr)
				if err != nil {
					t.Fatalf("Failed writing HTTP response: %v", err)
				}

				return h.ServeHTTP(ctx, w, r, binder)
			}
		})

		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}
		expectedString := append(append(mwStr, routeMwStr...), handlerStr...)
		if !bytes.Equal(rw.Body.Bytes(), expectedString) {
			fmt.Println(rw.Body.String())
			t.Fatal("Middleware + handler response is not what it should be")
		}
	})
}

func TestMiddlewareNotFound(t *testing.T) {
	mwStr := []byte("Middleware\n")
	handlerStr := []byte("Logic\n")

	router := NewRouter()
	router.HandleFunc("/", func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
		_, err := w.Write(handlerStr)

		return err
	})
	router.Use(func(h HandlerFunc) HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
			_, err := w.Write(mwStr)
			if err != nil {
				t.Fatalf("Failed writing HTTP response: %v", err)
			}

			return h.ServeHTTP(ctx, w, r, binder)
		}
	})

	// Test not found call with default handler
	t.Run("not called", func(t *testing.T) {
		rw := NewRecorder()
		req := newRequest("GET", "/notfound")

		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}
		if bytes.Contains(rw.Body.Bytes(), mwStr) {
			t.Fatal("Middleware was called for a 404")
		}
	})

	t.Run("not called with custom not found handler", func(t *testing.T) {
		rw := NewRecorder()
		req := newRequest("GET", "/notfound")

		router.NotFoundHandler = HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
			_, err := rw.Write([]byte("Custom 404 handler"))

			return err
		})
		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}

		if bytes.Contains(rw.Body.Bytes(), mwStr) {
			t.Fatal("Middleware was called for a custom 404")
		}
	})
}

func TestMiddlewareMethodMismatch(t *testing.T) {
	mwStr := []byte("Middleware\n")
	handlerStr := []byte("Logic\n")

	router := NewRouter()
	router.HandleFunc("/", func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
		_, err := w.Write(handlerStr)

		return err
	}).Methods("GET")

	router.Use(func(h HandlerFunc) HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
			_, err := w.Write(mwStr)
			if err != nil {
				t.Fatalf("Failed writing HTTP response: %v", err)
			}

			return h.ServeHTTP(ctx, w, r, binder)
		}
	})

	t.Run("not called", func(t *testing.T) {
		rw := NewRecorder()
		req := newRequest("POST", "/")

		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}
		if bytes.Contains(rw.Body.Bytes(), mwStr) {
			t.Fatal("Middleware was called for a method mismatch")
		}
	})

	t.Run("not called with custom method not allowed handler", func(t *testing.T) {
		rw := NewRecorder()
		req := newRequest("POST", "/")

		router.MethodNotAllowedHandler = HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
			_, err := rw.Write([]byte("Method not allowed"))

			return err
		})
		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}

		if bytes.Contains(rw.Body.Bytes(), mwStr) {
			t.Fatal("Middleware was called for a method mismatch")
		}
	})
}

func TestMiddlewareNotFoundSubrouter(t *testing.T) {
	mwStr := []byte("Middleware\n")
	handlerStr := []byte("Logic\n")

	router := NewRouter()
	router.HandleFunc("/", func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
		_, err := w.Write(handlerStr)

		return err
	})

	subrouter := router.PathPrefix("/sub/").Subrouter()
	subrouter.HandleFunc("/", func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
		_, err := w.Write(handlerStr)

		return err
	})

	router.Use(func(h HandlerFunc) HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
			_, err := w.Write(mwStr)
			if err != nil {
				t.Fatalf("Failed writing HTTP response: %v", err)
			}

			return h.ServeHTTP(ctx, w, r, binder)
		}
	})

	t.Run("not called", func(t *testing.T) {
		rw := NewRecorder()
		req := newRequest("GET", "/sub/notfound")

		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}
		if bytes.Contains(rw.Body.Bytes(), mwStr) {
			t.Fatal("Middleware was called for a 404")
		}
	})

	t.Run("not called with custom not found handler", func(t *testing.T) {
		rw := NewRecorder()
		req := newRequest("GET", "/sub/notfound")

		subrouter.NotFoundHandler = HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
			_, err := rw.Write([]byte("Custom 404 handler"))

			return err
		})
		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}

		if bytes.Contains(rw.Body.Bytes(), mwStr) {
			t.Fatal("Middleware was called for a custom 404")
		}
	})
}

func TestMiddlewareMethodMismatchSubrouter(t *testing.T) {
	mwStr := []byte("Middleware\n")
	handlerStr := []byte("Logic\n")

	router := NewRouter()
	router.HandleFunc("/", func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
		_, err := w.Write(handlerStr)

		return err
	})

	subrouter := router.PathPrefix("/sub/").Subrouter()
	subrouter.HandleFunc("/", func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
		_, err := w.Write(handlerStr)

		return err
	}).Methods("GET")

	router.Use(func(h HandlerFunc) HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
			_, err := w.Write(mwStr)
			if err != nil {
				t.Fatalf("Failed writing HTTP response: %v", err)
			}
			return h.ServeHTTP(ctx, w, r, binder)
		}
	})

	t.Run("not called", func(t *testing.T) {
		rw := NewRecorder()
		req := newRequest("POST", "/sub/")

		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}
		if bytes.Contains(rw.Body.Bytes(), mwStr) {
			t.Fatal("Middleware was called for a method mismatch")
		}
	})

	t.Run("not called with custom method not allowed handler", func(t *testing.T) {
		rw := NewRecorder()
		req := newRequest("POST", "/sub/")

		router.MethodNotAllowedHandler = HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
			_, err := rw.Write([]byte("Method not allowed"))

			return err
		})
		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}

		if bytes.Contains(rw.Body.Bytes(), mwStr) {
			t.Fatal("Middleware was called for a method mismatch")
		}
	})
}

func TestCORSMethodMiddleware(t *testing.T) {
	testCases := []struct {
		name                                    string
		registerRoutes                          func(r *Router)
		requestHeader                           http.Header
		requestMethod                           string
		requestPath                             string
		expectedAccessControlAllowMethodsHeader string
		expectedResponse                        string
	}{
		{
			name: "does not set without OPTIONS matcher",
			registerRoutes: func(r *Router) {
				r.HandleFunc("/foo", stringHandler("a")).Methods(http.MethodGet, http.MethodPut, http.MethodPatch)
			},
			requestMethod:                           "GET",
			requestPath:                             "/foo",
			expectedAccessControlAllowMethodsHeader: "",
			expectedResponse:                        "a",
		},
		{
			name: "sets on non OPTIONS",
			registerRoutes: func(r *Router) {
				r.HandleFunc("/foo", stringHandler("a")).Methods(http.MethodGet, http.MethodPut, http.MethodPatch)
				r.HandleFunc("/foo", stringHandler("b")).Methods(http.MethodOptions)
			},
			requestMethod:                           "GET",
			requestPath:                             "/foo",
			expectedAccessControlAllowMethodsHeader: "GET,PUT,PATCH,OPTIONS",
			expectedResponse:                        "a",
		},
		{
			name: "sets without preflight headers",
			registerRoutes: func(r *Router) {
				r.HandleFunc("/foo", stringHandler("a")).Methods(http.MethodGet, http.MethodPut, http.MethodPatch)
				r.HandleFunc("/foo", stringHandler("b")).Methods(http.MethodOptions)
			},
			requestMethod:                           "OPTIONS",
			requestPath:                             "/foo",
			expectedAccessControlAllowMethodsHeader: "GET,PUT,PATCH,OPTIONS",
			expectedResponse:                        "b",
		},
		{
			name: "does not set on error",
			registerRoutes: func(r *Router) {
				r.HandleFunc("/foo", stringHandler("a"))
			},
			requestMethod:                           "OPTIONS",
			requestPath:                             "/foo",
			expectedAccessControlAllowMethodsHeader: "",
			expectedResponse:                        "a",
		},
		{
			name: "sets header on valid preflight",
			registerRoutes: func(r *Router) {
				r.HandleFunc("/foo", stringHandler("a")).Methods(http.MethodGet, http.MethodPut, http.MethodPatch)
				r.HandleFunc("/foo", stringHandler("b")).Methods(http.MethodOptions)
			},
			requestMethod: "OPTIONS",
			requestPath:   "/foo",
			requestHeader: http.Header{
				"Access-Control-Request-Method":  []string{"GET"},
				"Access-Control-Request-Headers": []string{"Authorization"},
				"Origin":                         []string{"http://example.com"},
			},
			expectedAccessControlAllowMethodsHeader: "GET,PUT,PATCH,OPTIONS",
			expectedResponse:                        "b",
		},
		{
			name: "does not set methods from unmatching routes",
			registerRoutes: func(r *Router) {
				r.HandleFunc("/foo", stringHandler("c")).Methods(http.MethodDelete)
				r.HandleFunc("/foo/bar", stringHandler("a")).Methods(http.MethodGet, http.MethodPut, http.MethodPatch)
				r.HandleFunc("/foo/bar", stringHandler("b")).Methods(http.MethodOptions)
			},
			requestMethod: "OPTIONS",
			requestPath:   "/foo/bar",
			requestHeader: http.Header{
				"Access-Control-Request-Method":  []string{"GET"},
				"Access-Control-Request-Headers": []string{"Authorization"},
				"Origin":                         []string{"http://example.com"},
			},
			expectedAccessControlAllowMethodsHeader: "GET,PUT,PATCH,OPTIONS",
			expectedResponse:                        "b",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter()

			tt.registerRoutes(router)

			router.Use(CORSMethodMiddleware(router))

			rw := NewRecorder()
			req := newRequest(tt.requestMethod, tt.requestPath)
			req.Header = tt.requestHeader

			if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
				t.Fatalf("Failed to call ServeHTTP: %v", err)
			}

			actualMethodsHeader := rw.Header().Get("Access-Control-Allow-Methods")
			if actualMethodsHeader != tt.expectedAccessControlAllowMethodsHeader {
				t.Fatalf("Expected Access-Control-Allow-Methods to equal %s but got %s", tt.expectedAccessControlAllowMethodsHeader, actualMethodsHeader)
			}

			actualResponse := rw.Body.String()
			if actualResponse != tt.expectedResponse {
				t.Fatalf("Expected response to equal %s but got %s", tt.expectedResponse, actualResponse)
			}
		})
	}
}

func TestCORSMethodMiddlewareSubrouter(t *testing.T) {
	router := NewRouter().StrictSlash(true)

	subrouter := router.PathPrefix("/test").Subrouter()
	subrouter.HandleFunc("/hello", stringHandler("a")).Methods(http.MethodGet, http.MethodOptions, http.MethodPost)
	subrouter.HandleFunc("/hello/{name}", stringHandler("b")).Methods(http.MethodGet, http.MethodOptions)

	subrouter.Use(CORSMethodMiddleware(subrouter))

	rw := NewRecorder()
	req := newRequest("GET", "/test/hello/asdf")
	if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
		t.Fatalf("Failed to call ServeHTTP: %v", err)
	}

	actualMethods := rw.Header().Get("Access-Control-Allow-Methods")
	expectedMethods := "GET,OPTIONS"
	if actualMethods != expectedMethods {
		t.Fatalf("expected methods %q but got: %q", expectedMethods, actualMethods)
	}
}

func TestMiddlewareOnMultiSubrouter(t *testing.T) {
	first := "first"
	second := "second"
	notFound := "404 not found"

	router := NewRouter()
	firstSubRouter := router.PathPrefix("/").Subrouter()
	secondSubRouter := router.PathPrefix("/").Subrouter()

	router.NotFoundHandler = HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
		_, err := w.Write([]byte(notFound))

		return err
	})

	firstSubRouter.HandleFunc("/first", func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
		return nil
	})

	secondSubRouter.HandleFunc("/second", func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
		return nil
	})

	firstSubRouter.Use(func(h HandlerFunc) HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
			_, err := w.Write([]byte(first))
			if err != nil {
				t.Fatalf("Failed writing HTTP response: %v", err)
			}
			return h.ServeHTTP(ctx, w, r, binder)
		}
	})

	secondSubRouter.Use(func(h HandlerFunc) HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder Binder) error {
			_, err := w.Write([]byte(second))
			if err != nil {
				t.Fatalf("Failed writing HTTP response: %v", err)
			}
			return h.ServeHTTP(ctx, w, r, binder)
		}
	})

	t.Run("/first uses first middleware", func(t *testing.T) {
		rw := NewRecorder()
		req := newRequest("GET", "/first")

		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}
		if rw.Body.String() != first {
			t.Fatalf("Middleware did not run: expected %s middleware to write a response (got %s)", first, rw.Body.String())
		}
	})

	t.Run("/second uses second middleware", func(t *testing.T) {
		rw := NewRecorder()
		req := newRequest("GET", "/second")

		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}
		if rw.Body.String() != second {
			t.Fatalf("Middleware did not run: expected %s middleware to write a response (got %s)", second, rw.Body.String())
		}
	})

	t.Run("uses not found handler", func(t *testing.T) {
		rw := NewRecorder()
		req := newRequest("GET", "/second/not-exist")

		if err := router.ServeHTTP(context.Background(), rw, req, nil); err != nil {
			t.Fatalf("Failed to call ServeHTTP: %v", err)
		}
		if rw.Body.String() != notFound {
			t.Fatalf("Notfound handler did not run: expected %s for not-exist, (got %s)", notFound, rw.Body.String())
		}
	})
}
