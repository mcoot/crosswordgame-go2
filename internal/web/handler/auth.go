package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/mcoot/crosswordgame-go2/internal/services/auth"
	"github.com/mcoot/crosswordgame-go2/internal/web/middleware"
	"github.com/mcoot/crosswordgame-go2/internal/web/templates/layout"
	"github.com/mcoot/crosswordgame-go2/internal/web/templates/pages"
)

// AuthHandler handles authentication pages and actions
type AuthHandler struct {
	authService *auth.Service
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// LoginPage renders the login page
func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	player := middleware.GetPlayer(r.Context())
	if player != nil {
		// Already logged in, redirect to home
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	flash := middleware.GetFlash(r.Context())
	activeLobbyCode := middleware.GetActiveLobbyCode(r.Context())
	next := r.URL.Query().Get("next")

	data := pages.LoginData{
		PageData: layout.PageData{
			Title:           "Login",
			Flash:           flash,
			ActiveLobbyCode: activeLobbyCode,
		},
		Next: next,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.Login(data).Render(r.Context(), w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// RegisterPage renders the registration page
func (h *AuthHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	player := middleware.GetPlayer(r.Context())
	if player != nil {
		// Already logged in, redirect to home
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	flash := middleware.GetFlash(r.Context())
	activeLobbyCode := middleware.GetActiveLobbyCode(r.Context())

	data := pages.RegisterData{
		PageData: layout.PageData{
			Title:           "Register",
			Flash:           flash,
			ActiveLobbyCode: activeLobbyCode,
		},
		FieldErrors: make(map[string]string),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.Register(data).Render(r.Context(), w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// CreateGuest handles guest player creation
func (h *AuthHandler) CreateGuest(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderLoginError(w, r, "Invalid form data")
		return
	}

	displayName := strings.TrimSpace(r.FormValue("display_name"))
	next := r.FormValue("next")

	if displayName == "" {
		middleware.SetFlash(w, "error", "Display name is required")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if len(displayName) > 20 {
		displayName = displayName[:20]
	}

	session, err := h.authService.CreateGuestPlayer(r.Context(), displayName)
	if err != nil {
		middleware.SetFlash(w, "error", "Failed to create guest player")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	h.setSessionCookie(w, session.Token)
	middleware.SetFlash(w, "success", "Welcome, "+session.Player.DisplayName+"!")

	// Redirect to original destination or home
	if next != "" && strings.HasPrefix(next, "/") {
		http.Redirect(w, r, next, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// Login handles login form submission
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderLoginError(w, r, "Invalid form data")
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	next := r.FormValue("next")

	if username == "" || password == "" {
		h.renderLoginErrorWithData(w, r, "Username and password are required", username, next)
		return
	}

	session, err := h.authService.Login(r.Context(), username, password)
	if err != nil {
		h.renderLoginErrorWithData(w, r, "Invalid username or password", username, next)
		return
	}

	h.setSessionCookie(w, session.Token)
	middleware.SetFlash(w, "success", "Welcome back, "+session.Player.DisplayName+"!")

	// Redirect to original destination or home
	if next != "" && strings.HasPrefix(next, "/") {
		http.Redirect(w, r, next, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// Register handles registration form submission
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderRegisterError(w, r, "Invalid form data", "", "", nil)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	displayName := strings.TrimSpace(r.FormValue("display_name"))
	password := r.FormValue("password")
	passwordConfirm := r.FormValue("password_confirm")

	fieldErrors := make(map[string]string)

	// Validate inputs
	if username == "" {
		fieldErrors["username"] = "Username is required"
	} else if len(username) < 3 {
		fieldErrors["username"] = "Username must be at least 3 characters"
	} else if len(username) > 20 {
		fieldErrors["username"] = "Username must be at most 20 characters"
	}

	if displayName == "" {
		fieldErrors["display_name"] = "Display name is required"
	} else if len(displayName) > 20 {
		fieldErrors["display_name"] = "Display name must be at most 20 characters"
	}

	if password == "" {
		fieldErrors["password"] = "Password is required"
	} else if len(password) < 8 {
		fieldErrors["password"] = "Password must be at least 8 characters"
	}

	if password != passwordConfirm {
		fieldErrors["password_confirm"] = "Passwords do not match"
	}

	if len(fieldErrors) > 0 {
		h.renderRegisterError(w, r, "", username, displayName, fieldErrors)
		return
	}

	session, err := h.authService.RegisterPlayer(r.Context(), username, password, displayName)
	if err != nil {
		// Check for specific errors
		errMsg := err.Error()
		if strings.Contains(errMsg, "already exists") {
			fieldErrors["username"] = "Username already taken"
			h.renderRegisterError(w, r, "", username, displayName, fieldErrors)
		} else {
			h.renderRegisterError(w, r, "Registration failed: "+errMsg, username, displayName, nil)
		}
		return
	}

	h.setSessionCookie(w, session.Token)
	middleware.SetFlash(w, "success", "Account created! Welcome, "+session.Player.DisplayName+"!")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Logout handles logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	middleware.SetFlash(w, "info", "You have been logged out")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) renderLoginError(w http.ResponseWriter, r *http.Request, errorMsg string) {
	h.renderLoginErrorWithData(w, r, errorMsg, "", "")
}

func (h *AuthHandler) renderLoginErrorWithData(w http.ResponseWriter, r *http.Request, errorMsg, username, next string) {
	activeLobbyCode := middleware.GetActiveLobbyCode(r.Context())
	data := pages.LoginData{
		PageData: layout.PageData{
			Title:           "Login",
			ActiveLobbyCode: activeLobbyCode,
		},
		Username: username,
		Error:    errorMsg,
		Next:     next,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.Login(data).Render(r.Context(), w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *AuthHandler) renderRegisterError(w http.ResponseWriter, r *http.Request, errorMsg, username, displayName string, fieldErrors map[string]string) {
	if fieldErrors == nil {
		fieldErrors = make(map[string]string)
	}

	activeLobbyCode := middleware.GetActiveLobbyCode(r.Context())
	data := pages.RegisterData{
		PageData: layout.PageData{
			Title:           "Register",
			ActiveLobbyCode: activeLobbyCode,
		},
		Username:    username,
		DisplayName: displayName,
		Error:       errorMsg,
		FieldErrors: fieldErrors,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.Register(data).Render(r.Context(), w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
