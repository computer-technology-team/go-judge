package auth

import (
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	"net/http"
	"time"
)

// NewRoutes returns a function that registers routes with the given handler
// This allows for dependency injection when setting up routes
func NewRoutes(s Servicer) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/login", s.ShowLoginPage)
		r.Get("/signup", s.ShowSignupPage)
		r.Get("/logout", s.Logout)
		r.Post("/login", s.Login)
		r.Post("/signup", s.Signup)
		r.Post("/refresh", s.RefreshToken)
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
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	s.loginUser(w, r, username, password)
}

func (s *DefaultServicer) loginUser(w http.ResponseWriter, r *http.Request, username string, password string) {

	user, err := s.querier.GetUserByUsername(r.Context(), s.pool, username)
	if err != nil {
		http.Error(w, "Username not found", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	claims := Claims{
		UserID: user.ID.String(),
	}

	tokenString, err := s.authenticator.GenerateToken(r.Context(), claims)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	cookie := &http.Cookie{
		Name:     "token",
		Value:    tokenString,
		Expires:  time.Now().Add(s.authenticator.(*AuthenticatorImpl).tokenExpireDuration),
		HttpOnly: true,
		Path:     "/",
	}
	http.SetCookie(w, cookie)

	http.Redirect(w, r, "/", http.StatusSeeOther)

	w.WriteHeader(http.StatusOK)
}

// ShowSignupPage handles user registration
func (s *DefaultServicer) ShowSignupPage(w http.ResponseWriter, r *http.Request) {
	err := s.templates.Render(r.Context(), "signup", w, nil)
	if err != nil {
		slog.Error("could not render signup", "error", err)
		http.Error(w, "could not render", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *DefaultServicer) Signup(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error processing password", http.StatusInternalServerError)
		return
	}

	_, err = s.querier.CreateUser(r.Context(), s.pool, username, string(hashedPassword))
	if err != nil {
		http.Error(w, "Could not create user", http.StatusInternalServerError)
		return
	}

	s.loginUser(w, r, username, password)
}

// Logout handles user logout
func (s *DefaultServicer) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "", // Clear the value
		Path:     "/",
		Expires:  time.Unix(0, 0), // Expire it
		MaxAge:   -1,              // Delete immediately
		HttpOnly: true,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)

	w.WriteHeader(http.StatusOK)
}

// RefreshToken handles token refresh
func (s *DefaultServicer) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement token refresh logic
	w.Write([]byte("Refresh token endpoint"))
}
