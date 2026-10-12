package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"prosto-prosto/internal/models"
)

// Формат даты, который отдаёт <input type="date">.
const dateLayout = "2006-01-02"

// App хранит шаблоны и список трат (пока в памяти, без базы данных).
type App struct {
	templates map[string]*template.Template

	mu       sync.RWMutex
	expenses []models.Expense
	nextID   int
}

// Данные для страницы со списком трат.
type listData struct {
	Expenses []models.Expense
	Total    float64
}

// Данные для формы.
type formData struct {
	Error       string
	Amount      string
	Description string
	Date        string
}

// New компилирует шаблоны один раз при старте.
// Для каждой страницы парсим пару: layout.html + <страница>.html.
func New(templatesDir string) (*App, error) {
	pages := []string{"expenses", "new", "about"}
	layout := filepath.Join(templatesDir, "layout.html")

	cache := make(map[string]*template.Template, len(pages))
	for _, page := range pages {
		t, err := template.ParseFiles(layout, filepath.Join(templatesDir, page+".html"))
		if err != nil {
			return nil, fmt.Errorf("шаблон %q: %w", page, err)
		}
		cache[page] = t
	}
	return &App{templates: cache, nextID: 1}, nil
}

// Routes регистрирует маршруты.
func (a *App) Routes(staticDir string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", a.home) // {$} — только точный путь "/"
	mux.HandleFunc("GET /about", a.about)
	mux.HandleFunc("GET /ping", a.ping)

	mux.HandleFunc("GET /expenses", a.list)
	mux.HandleFunc("GET /expenses/new", a.newForm)
	mux.HandleFunc("POST /expenses", a.create)

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
	return mux
}

// render сначала выполняет шаблон в буфер: если в шаблоне ошибка,
// клиент получит 500, а не наполовину отрисованную страницу.
func (a *App) render(w http.ResponseWriter, status int, page string, data any) {
	t, ok := a.templates[page]
	if !ok {
		log.Printf("шаблон %q не найден", page)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout", data); err != nil {
		log.Printf("ошибка рендера %q: %v", page, err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	buf.WriteTo(w)
}

func (a *App) home(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/expenses", http.StatusFound) // 302
}

func (a *App) about(w http.ResponseWriter, r *http.Request) {
	a.render(w, http.StatusOK, "about", nil)
}

func (a *App) ping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, "pong")
}

// GET /expenses — список трат.
func (a *App) list(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	items := make([]models.Expense, len(a.expenses))
	copy(items, a.expenses)
	a.mu.RUnlock()

	// Сначала новые.
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Date.After(items[j].Date)
	})

	var total float64
	for _, e := range items {
		total += e.Amount
	}

	a.render(w, http.StatusOK, "expenses", listData{Expenses: items, Total: total})
}

// GET /expenses/new — пустая форма.
func (a *App) newForm(w http.ResponseWriter, r *http.Request) {
	a.render(w, http.StatusOK, "new", formData{Date: time.Now().Format(dateLayout)})
}

// POST /expenses — обработка формы.
func (a *App) create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Не удалось разобрать форму", http.StatusBadRequest)
		return
	}

	amountStr := strings.TrimSpace(r.FormValue("amount"))
	description := strings.TrimSpace(r.FormValue("description"))
	dateStr := strings.TrimSpace(r.FormValue("date"))

	// При ошибке показываем форму снова с кодом 400 и сохранёнными значениями.
	fail := func(msg string) {
		a.render(w, http.StatusBadRequest, "new", formData{
			Error:       msg,
			Amount:      amountStr,
			Description: description,
			Date:        dateStr,
		})
	}

	// Разрешаем и запятую, и точку: "12,50" и "12.50".
	amount, err := strconv.ParseFloat(strings.ReplaceAll(amountStr, ",", "."), 64)
	if err != nil || math.IsNaN(amount) || math.IsInf(amount, 0) || amount <= 0 {
		fail("Сумма должна быть положительным числом, например 250 или 99,90")
		return
	}
	if description == "" {
		fail("Введите описание траты")
		return
	}
	date, err := time.Parse(dateLayout, dateStr)
	if err != nil {
		fail("Укажите дату в формате ГГГГ-ММ-ДД")
		return
	}

	a.mu.Lock()
	a.expenses = append(a.expenses, models.Expense{
		ID:          a.nextID,
		Amount:      amount,
		Description: description,
		Date:        date,
	})
	a.nextID++
	a.mu.Unlock()

	// Post/Redirect/Get: после успешного POST отправляем клиента на GET-страницу.
	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}
