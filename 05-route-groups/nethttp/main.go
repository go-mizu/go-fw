package main

import (
	"fmt"
	"net/http"
)

func main() {
	root := http.NewServeMux()

	root.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "root")
	})

	apiV1 := http.NewServeMux()
	apiV1.HandleFunc("GET /users", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "api v1 users")
	})
	apiV1.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "api v1 user:", r.PathValue("id"))
	})

	admin := http.NewServeMux()
	admin.HandleFunc("GET /dashboard", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "admin dashboard")
	})

	root.Handle("/api/v1/", http.StripPrefix("/api/v1", apiV1))
	root.Handle("/admin/", chain(http.StripPrefix("/admin", admin), requireToken("letmein")))

	http.ListenAndServe(":8080", root)
}

type Middleware func(http.Handler) http.Handler

func chain(h http.Handler, m ...Middleware) http.Handler {
	for i := len(m) - 1; i >= 0; i-- {
		h = m[i](h)
	}
	return h
}

func requireToken(token string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Admin-Token") != token {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("unauthorized"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
