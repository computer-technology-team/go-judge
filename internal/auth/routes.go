package auth

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewRoutes returns a function that registers routes with the given handler
// This allows for dependency injection when setting up routes
func NewRoutes(s Servicer) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/login", s.ShowLoginPage)
		r.Get("/signup", s.ShowSignupPage)
		r.Post("/login", s.Login)
		r.Post("/signup", s.Signup)
		//r.Post("/logout", s.Logout)
		//r.Post("/refresh", s.RefreshToken)
	}
}

// ShowLoginPage handles user login
func (s *DefaultServicer) ShowLoginPage(w http.ResponseWriter, r *http.Request) {

	err := s.templates.Render(r.Context(), "login", w, nil)
	if err != nil {
		slog.Error("could not render login", "error", err)
		http.Error(w, "could not render", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *DefaultServicer) Login(w http.ResponseWriter, r *http.Request) {
	//TODO: Implement sql login logic
	w.Write([]byte("welcome"))
}

// ShowSignupPage handles user registration
func (s *DefaultServicer) ShowSignupPage(w http.ResponseWriter, r *http.Request) {
	err := s.templates.Render(r.Context(), "signup", w, nil)
	if err != nil {
		slog.Error("could not render signup", "error", err)
		http.Error(w, "could not render", http.StatusInternalServerError)
		return
	}

	//TODO: Implement sql ShowSignupPage logic
	w.WriteHeader(http.StatusOK)
}

func (s *DefaultServicer) Signup(w http.ResponseWriter, r *http.Request) {
	// Parse the form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Extract values from the form
	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")

	fmt.Println(username, password, email)

	// Validate input (you might add more validation here)
	//if username == "" || email == "" || password == "" {
	//	http.Error(w, "All fields are required", http.StatusBadRequest)
	//	return
	//}

	// Hash the password using bcrypt
	//hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	//if err != nil {
	//	http.Error(w, "Error processing password", http.StatusInternalServerError)
	//	w.Writ
	//	return
	//}

	// Create the user record (this is pseudocode; implement your own DB logic)
	//err = CreateUser(username, email, string(hashedPassword))
	//if err != nil {
	//	http.Error(w, "Could not create user", http.StatusInternalServerError)
	//	return
	//}

	// Redirect the user to login or show a success message
	//http.Redirect(w, r, "/login", http.StatusSeeOther)

	w.WriteHeader(http.StatusOK)
}

// Logout handles user logout
func (s *DefaultServicer) Logout(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement logout logic
	w.Write([]byte("Logout endpoint"))
}

// RefreshToken handles token refresh
func (s *DefaultServicer) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement token refresh logic
	w.Write([]byte("Refresh token endpoint"))
}
