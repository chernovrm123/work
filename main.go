package main

import (
	"fmt"
	"log"
	"net/http"
)

// Главная страница. Шаблон "/" в ServeMux совпадает со ВСЕМИ путями,
// которым не нашлось более точного обработчика, поэтому проверяем путь вручную.
func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "Добро пожаловать на мой первый Go-сервер!")
}

// Страница с описанием проекта.
func aboutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "Это учебный проект: простой HTTP-сервер на Go, написанный с использованием только стандартной библиотеки net/http.")
}

// /ping: разрешён только GET, на остальные методы отвечаем 405.
func pingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		// Allow обязателен по RFC 9110 для ответа 405
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "метод не разрешён", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "pong")
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/about", aboutHandler)
	mux.HandleFunc("/ping", pingHandler)

	addr := ":8080"
	log.Println("Сервер запущен на http://localhost" + addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
