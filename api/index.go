// Package handler exposes the Compo application as one Vercel Go Function.
package handler

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/ivantan02/fitness-tracker-app/pkg/appserver"
)

var (
	initOnce sync.Once
	app      *appserver.App
	initErr  error
)

// Handler is the entry point used by Vercel's Go runtime.
func Handler(w http.ResponseWriter, r *http.Request) {
	initOnce.Do(func() {
		app, initErr = appserver.New(context.Background())
		if initErr != nil {
			log.Printf("initializing application: %v", initErr)
		}
	})

	if initErr != nil {
		http.Error(w, "application unavailable", http.StatusServiceUnavailable)
		return
	}

	// vercel.json sends the original path as a rewrite parameter so the
	// existing Chi routes see the same URL locally and in production.
	if originalPath := r.URL.Query().Get("__path"); originalPath != "" {
		r.URL.Path = "/" + strings.TrimPrefix(originalPath, "/")
		query := r.URL.Query()
		query.Del("__path")
		r.URL.RawQuery = query.Encode()
	} else {
		r.URL.Path = "/"
	}

	app.Handler.ServeHTTP(w, r)
}
