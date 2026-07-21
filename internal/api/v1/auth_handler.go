package v1

import (
	"encoding/json"
	"net/http"
	"time"

	apimiddleware "github.com/brycesharrits/fam-cal-insta/internal/api/middleware"
	"github.com/brycesharrits/fam-cal-insta/internal/auth"
	"github.com/brycesharrits/fam-cal-insta/internal/domain"
	"github.com/brycesharrits/fam-cal-insta/internal/repository"
)

type AuthHandler struct {
	appleVerifier  auth.OIDCVerifier
	googleVerifier auth.OIDCVerifier
	jwtSvc         *auth.JWTService
	userRepo       repository.UserRepository
}

func NewAuthHandler(
	appleVerifier auth.OIDCVerifier,
	googleVerifier auth.OIDCVerifier,
	jwtSvc *auth.JWTService,
	userRepo repository.UserRepository,
) *AuthHandler {
	return &AuthHandler{
		appleVerifier:  appleVerifier,
		googleVerifier: googleVerifier,
		jwtSvc:         jwtSvc,
		userRepo:       userRepo,
	}
}

type oidcSignInRequest struct {
	IDToken string `json:"id_token"`
}

type authResponse struct {
	Token string  `json:"token"`
	User  userDTO `json:"user"`
}

type userDTO struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	TokenBalance int       `json:"token_balance"`
	CreatedAt    time.Time `json:"created_at"`
}

// GET /api/v1/users/me
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := apimiddleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	user, err := h.userRepo.FindByID(r.Context(), userID)
	if err != nil || user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, userDTO{
		ID:           user.ID,
		Email:        user.Email,
		TokenBalance: user.TokenBalance,
		CreatedAt:    user.CreatedAt,
	})
}

// POST /api/v1/auth/apple
func (h *AuthHandler) AppleSignIn(w http.ResponseWriter, r *http.Request) {
	h.oidcSignIn(w, r, h.appleVerifier)
}

// POST /api/v1/auth/google
func (h *AuthHandler) GoogleSignIn(w http.ResponseWriter, r *http.Request) {
	h.oidcSignIn(w, r, h.googleVerifier)
}

func (h *AuthHandler) oidcSignIn(w http.ResponseWriter, r *http.Request, verifier auth.OIDCVerifier) {
	var req oidcSignInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.IDToken == "" {
		writeError(w, http.StatusBadRequest, "id_token is required")
		return
	}

	claims, err := verifier.Verify(r.Context(), req.IDToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid identity token")
		return
	}

	user, err := h.userRepo.FindByProviderID(r.Context(), verifier.Provider(), claims.Subject)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	if user == nil {
		user = &domain.User{
			Provider:       verifier.Provider(),
			ProviderUserID: claims.Subject,
			Email:          claims.Email,
			TokenBalance:   0,
		}
		if err := h.userRepo.Create(r.Context(), user); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create user")
			return
		}
	}

	token, err := h.jwtSvc.Sign(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to sign token")
		return
	}

	writeJSON(w, http.StatusOK, authResponse{
		Token: token,
		User: userDTO{
			ID:           user.ID,
			Email:        user.Email,
			TokenBalance: user.TokenBalance,
			CreatedAt:    user.CreatedAt,
		},
	})
}
