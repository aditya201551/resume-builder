package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/repository"
	"resume-builder/backend/internal/service"
)

const sessionCookieName = "session"
const oauthStateCookieName = "oauth_state"

type AuthHandler struct {
	providers    auth.Providers
	states       *auth.StateSigner
	jwt          *auth.JWTIssuer
	authService  *service.AuthService
	users        *repository.UserRepository
	frontendURL  string
	cookieSecure bool
	sessionTTL   time.Duration
}

func NewAuthHandler(
	providers auth.Providers,
	states *auth.StateSigner,
	jwtIssuer *auth.JWTIssuer,
	authService *service.AuthService,
	users *repository.UserRepository,
	frontendURL string,
	cookieSecure bool,
	sessionTTL time.Duration,
) *AuthHandler {
	return &AuthHandler{
		providers:    providers,
		states:       states,
		jwt:          jwtIssuer,
		authService:  authService,
		users:        users,
		frontendURL:  frontendURL,
		cookieSecure: cookieSecure,
		sessionTTL:   sessionTTL,
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	providerName := r.PathValue("provider")
	provider, ok := h.providers[providerName]
	if !ok {
		http.Error(w, "provider not configured", http.StatusNotImplemented)
		return
	}

	state, err := h.states.New()
	if err != nil {
		http.Error(w, "failed to start login", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})

	http.Redirect(w, r, provider.OAuth2.AuthCodeURL(state, oauth2.AccessTypeOnline), http.StatusFound)
}

func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	providerName := r.PathValue("provider")
	provider, ok := h.providers[providerName]
	if !ok {
		http.Error(w, "provider not configured", http.StatusNotImplemented)
		return
	}

	stateCookie, err := r.Cookie(oauthStateCookieName)
	if err != nil {
		http.Error(w, "missing state cookie", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: oauthStateCookieName, Value: "", Path: "/", MaxAge: -1})

	if r.URL.Query().Get("state") != stateCookie.Value {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}
	if err := h.states.Verify(stateCookie.Value); err != nil {
		http.Error(w, "invalid or expired state", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	token, err := provider.OAuth2.Exchange(r.Context(), code)
	if err != nil {
		log.Printf("oauth exchange failed for %s: %v", providerName, err)
		http.Error(w, "authentication failed", http.StatusBadGateway)
		return
	}

	profile, err := auth.FetchProfile(r.Context(), providerName, provider.OAuth2, token)
	if err != nil {
		log.Printf("fetch profile failed for %s: %v", providerName, err)
		http.Error(w, "authentication failed", http.StatusBadGateway)
		return
	}

	user, err := h.authService.Login(r.Context(), providerName, profile)
	if err != nil {
		log.Printf("login failed for %s: %v", providerName, err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	jwtToken, err := h.jwt.Issue(user.ID)
	if err != nil {
		http.Error(w, "failed to issue session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    jwtToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.sessionTTL.Seconds()),
	})

	http.Redirect(w, r, h.frontendURL, http.StatusFound)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.users.FindByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
