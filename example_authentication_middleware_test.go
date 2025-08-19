package mux_test

import (
	"context"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// Define our struct
type authenticationMiddleware struct {
	tokenUsers map[string]string
}

// Initialize it somewhere
func (amw *authenticationMiddleware) Populate() {
	amw.tokenUsers["00000000"] = "user0"
	amw.tokenUsers["aaaaaaaa"] = "userA"
	amw.tokenUsers["05f717e5"] = "randomUser"
	amw.tokenUsers["deadbeef"] = "user0"
}

// Middleware function, which will be called for each request
func (amw *authenticationMiddleware) Middleware(next mux.HandlerFunc) mux.HandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder mux.Binder) error {
		token := r.Header.Get("X-Session-Token")

		if user, found := amw.tokenUsers[token]; found {
			// We found the token in our map
			log.Printf("Authenticated user %s\n", user)
			next.ServeHTTP(ctx, w, r, binder)
		} else {
			http.Error(w, "Forbidden", http.StatusForbidden)
		}

		return nil
	}
}

func Example_authenticationMiddleware() {
	r := mux.NewRouter()
	r.HandleFunc("/", func(ctx context.Context, w http.ResponseWriter, r *http.Request, binder mux.Binder) error {
		return nil
	})
	amw := authenticationMiddleware{make(map[string]string)}
	amw.Populate()
	r.Use(amw.Middleware)
}
