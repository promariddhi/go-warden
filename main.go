package main

import (
	"go_template/internal/middleware"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	logged := middleware.Logging(mux)

	if err := http.ListenAndServe(":8090", logged); err != nil {
		log.Fatal("Server Failure")
	}
}
