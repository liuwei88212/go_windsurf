package app

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"

	"GoTemplate/internal/config"
)

// App represents the application
type App struct {
	cfg    *config.Config
	server *http.Server
	router *http.ServeMux
}

// PageData represents the data to be displayed in the template
type PageData struct {
	Title string
	Items []Item
}

// Item represents a single item in the list
type Item struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// User represents a user in the system
type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the response from the login API
type LoginResponse struct {
	Success  bool   `json:"success"`
	Token    string `json:"token,omitempty"`
	Username string `json:"username,omitempty"`
	Message  string `json:"message,omitempty"`
}

// New creates a new application instance
func New(cfg *config.Config) *App {
	return &App{
		cfg:    cfg,
		router: http.NewServeMux(),
	}
}

// Run starts the application
func (a *App) Run() error {
	return a.startHTTPServer()
}

// Stop gracefully shuts down the application
func (a *App) Stop(ctx context.Context) error {
	if a.server != nil {
		return a.server.Shutdown(ctx)
	}
	return nil
}

// startHTTPServer initializes and starts the HTTP server
func (a *App) startHTTPServer() error {
	a.routes()

	a.server = &http.Server{
		Addr:    a.cfg.Server.Address,
		Handler: a.router,
	}

	return a.server.ListenAndServe()
}

func (a *App) routes() {
	// Serve static files
	fs := http.FileServer(http.Dir("web/static"))
	a.router.Handle("/static/", http.StripPrefix("/static/", fs))

	// API endpoints
	a.router.HandleFunc("/api/items", a.handleItems)
	a.router.HandleFunc("/api/login", a.handleLogin)

	// Template endpoints
	a.router.HandleFunc("/", a.handleHome)
	a.router.HandleFunc("/items", a.handleItemsPage)
}

// handleHome renders the home page
func (a *App) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tmpl, err := template.ParseFiles("web/templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title: "Go Template App",
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleItemsPage renders the items page
func (a *App) handleItemsPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("web/templates/items.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Sample data
	data := PageData{
		Title: "Manage Items",
		Items: []Item{
			{ID: 1, Name: "Item 1", Value: "Value 1"},
			{ID: 2, Name: "Item 2", Value: "Value 2"},
			{ID: 3, Name: "Item 3", Value: "Value 3"},
		},
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleItems handles the API endpoint for items
func (a *App) handleItems(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Sample data for GET request
		items := []Item{
			{ID: 1, Name: "Item 1", Value: "Value 1"},
			{ID: 2, Name: "Item 2", Value: "Value 2"},
			{ID: 3, Name: "Item 3", Value: "Value 3"},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)

	case http.MethodPost:
		// Handle creating new item
		var newItem Item
		if err := json.NewDecoder(r.Body).Decode(&newItem); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		// Here you would typically save the item to a database
		// For now, we'll just return the item
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(newItem)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleLogin processes login requests
func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: In a real application, validate against a database
	// This is just a simple example
	response := LoginResponse{}
	if user.Username == "admin" && user.Password == "admin123" {
		// In a real application, you would:
		// 1. Hash the password before comparing
		// 2. Generate a proper JWT token
		// 3. Store session information
		response.Success = true
		response.Token = "sample-jwt-token"
		response.Username = user.Username
		response.Message = "Login successful"
	} else {
		response.Success = false
		response.Message = "Invalid credentials"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
