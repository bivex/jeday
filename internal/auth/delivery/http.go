package delivery

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jeday/auth/internal/auth/service"
	"github.com/savsgio/atreugo/v11"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(s service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: s,
	}
}

func (h *AuthHandler) AuthMiddleware(ctx *atreugo.RequestCtx) error {
	authHeader := string(ctx.Request.Header.Peek("Authorization"))
	if authHeader == "" {
		return ctx.ErrorResponse(service.ErrUnauthorized, http.StatusUnauthorized)
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ctx.ErrorResponse(service.ErrUnauthorized, http.StatusUnauthorized)
	}

	subject, err := h.authService.VerifyToken(parts[1])
	if err != nil {
		return ctx.ErrorResponse(err, http.StatusUnauthorized)
	}

	ctx.SetUserValue("user_id", subject)

	return ctx.Next()
}

func (h *AuthHandler) RegisterRoutes(server *atreugo.Atreugo) {
	server.UseBefore(LoggerMiddleware)

	authGroup := server.NewGroupPath("/auth")
	authGroup.POST("/register", h.Register)
	authGroup.POST("/login", h.Login)
	authGroup.POST("/refresh", h.Refresh)

	protected := server.NewGroupPath("/auth")
	protected.UseBefore(h.AuthMiddleware)
	protected.POST("/logout", h.Logout)
	protected.GET("/me", h.GetMe)
}

type registerRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(ctx *atreugo.RequestCtx) error {
	var req registerRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		return ctx.ErrorResponse(err, http.StatusBadRequest)
	}

	user, err := h.authService.RegisterUser(context.Background(), req.Email, req.Username, req.Password)
	if err != nil {
		return ctx.ErrorResponse(err, http.StatusInternalServerError)
	}

	return ctx.JSONResponse(user)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Login(ctx *atreugo.RequestCtx) error {
	var req loginRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		return ctx.ErrorResponse(err, http.StatusBadRequest)
	}

	userAgent := string(ctx.Request.Header.UserAgent())
	ipAddress := ctx.RemoteAddr().String()

	accessToken, refreshToken, err := h.authService.LoginUser(context.Background(), req.Email, req.Password, userAgent, ipAddress)
	if err != nil {
		if err == service.ErrInvalidCreds {
			return ctx.ErrorResponse(err, http.StatusUnauthorized)
		}
		return ctx.ErrorResponse(err, http.StatusInternalServerError)
	}

	return ctx.JSONResponse(loginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Refresh(ctx *atreugo.RequestCtx) error {
	var req refreshRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		return ctx.ErrorResponse(err, http.StatusBadRequest)
	}

	accessToken, err := h.authService.RefreshToken(context.Background(), req.RefreshToken)
	if err != nil {
		return ctx.ErrorResponse(err, http.StatusUnauthorized)
	}

	return ctx.JSONResponse(map[string]string{
		"access_token": accessToken,
	})
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Logout(ctx *atreugo.RequestCtx) error {
	var req logoutRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		// Log out anyway, if error decode payload - ignore
	}

	// This validates via middleware, so user is authorized
	err := h.authService.Logout(context.Background(), req.RefreshToken)
	if err != nil {
		return ctx.ErrorResponse(err, http.StatusInternalServerError)
	}

	return ctx.TextResponse("logged out")
}

func (h *AuthHandler) GetMe(ctx *atreugo.RequestCtx) error {
	userIDStr, ok := ctx.UserValue("user_id").(string)
	if !ok {
		return ctx.ErrorResponse(service.ErrUnauthorized, http.StatusUnauthorized)
	}

	user, err := h.authService.GetUser(context.Background(), userIDStr)
	if err != nil {
		return ctx.ErrorResponse(err, http.StatusInternalServerError)
	}

	return ctx.JSONResponse(user)
}
