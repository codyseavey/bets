package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/codyseavey/bets/middleware"
	"github.com/codyseavey/bets/services"
)

type AuthHandler struct {
	authService  *services.AuthService
	baseURL      string
	secureCookie bool
}

func NewAuthHandler(authService *services.AuthService, baseURL string) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		baseURL:      baseURL,
		secureCookie: strings.HasPrefix(baseURL, "https://"),
	}
}

func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	state := generateState()
	// Store state in a short-lived cookie for CSRF protection
	c.SetCookie("oauth_state", state, 300, "/", "", h.secureCookie, true)
	url := h.authService.GetAuthURL(state)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	state := c.Query("state")
	storedState, err := c.Cookie("oauth_state")
	if err != nil || state != storedState {
		c.Redirect(http.StatusTemporaryRedirect, h.baseURL+"/login?error=invalid_state")
		return
	}
	// Clear the state cookie
	c.SetCookie("oauth_state", "", -1, "/", "", h.secureCookie, true)

	code := c.Query("code")
	if code == "" {
		c.Redirect(http.StatusTemporaryRedirect, h.baseURL+"/login?error=no_code")
		return
	}

	user, err := h.authService.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, h.baseURL+"/login?error=exchange_failed")
		return
	}

	token, err := h.authService.GenerateJWT(user)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, h.baseURL+"/login?error=token_failed")
		return
	}

	c.SetCookie(middleware.CookieName, token, 7*24*3600, "/", "", h.secureCookie, true)
	c.Redirect(http.StatusTemporaryRedirect, h.baseURL+"/")
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := middleware.GetUserID(c)
	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=128"`
	Name     string `json:"name" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authService.Register(req.Email, req.Password, req.Name)
	if err != nil {
		if errors.Is(err, services.ErrEmailTaken) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "registration failed"})
		}
		return
	}

	token, err := h.authService.GenerateJWT(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.SetCookie(middleware.CookieName, token, 7*24*3600, "/", "", h.secureCookie, true)
	c.JSON(http.StatusCreated, user)
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	token, err := h.authService.GenerateJWT(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.SetCookie(middleware.CookieName, token, 7*24*3600, "/", "", h.secureCookie, true)
	c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie(middleware.CookieName, "", -1, "/", "", h.secureCookie, true)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func generateState() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "fallback-state"
	}
	return hex.EncodeToString(b)
}
