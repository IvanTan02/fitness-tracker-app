// Command server wires together the router, middleware, and feature
// registration. It contains no business logic of its own.
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/ivantan02/fitness-tracker-app/pkg/appserver"
)

func main() {
	ctx := context.Background()

	app, err := appserver.New(ctx)
	if err != nil {
		log.Fatalf("starting application: %v", err)
	}
	defer app.Close()

	log.Printf("listening on :%s", app.Port)
	if err := http.ListenAndServe(":"+app.Port, app.Handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
