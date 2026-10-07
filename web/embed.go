// Package web embeds the browser application into the server binary.
package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed index.html styles.css app.js scans.js manifest.webmanifest sw.js icons/*
var files embed.FS

// Handler serves the embedded frontend assets.
func Handler() http.Handler {
	assets, err := fs.Sub(files, ".")
	if err != nil {
		panic("creating embedded web filesystem: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(assets))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.webmanifest":
			w.Header().Set("Content-Type", "application/manifest+json")
		case "/sw.js":
			w.Header().Set("Cache-Control", "no-cache")
		}
		fileServer.ServeHTTP(w, r)
	})
}
