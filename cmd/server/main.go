package main

import (
	"log"
	"net/http"

	"prosto-prosto/internal/handlers"
)

func main() {
	// Пути относительные: запускать нужно из корня проекта (go run ./cmd/server).
	app, err := handlers.New("web/templates")
	if err != nil {
		log.Fatal(err)
	}

	addr := ":8080"
	log.Println("Сервер запущен на http://localhost" + addr)
	log.Fatal(http.ListenAndServe(addr, app.Routes("web/static")))
}
