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
	appleVerifier *auth.AppleSignInVerifier
	jwtSvc        *auth.JWTService
	userRepo      repository.UserRepository
}

func NewAuthHandler(
	appleVerifier *auth.AppleSignInVerifier,
	jwtSvc *auth.JWTService,
	userRepo repository.UserRepository,
) *AuthHandler {
	return &AuthHandler{
		appleVerifier: appleVerifier,
		jwtSvc:        jwtSvc,
		userRepo:      userRepo,
	}
}

type appleAuthRequest struct {
	IdentityToken     string `json:"identity_token"`
	AuthorizationCode string `json:"authorization_code"`
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

// POST /api/v1/dev/auth
// Temporary unauthenticated endpoint that upserts a fixed seed user and
// returns a real JWT signed with the same key as production auth.
// Lets the iOS app exercise authenticated endpoints in local dev without
// real Apple Sign In. Gated to APP_ENV != "production" at the router.
// Remove once real Apple Sign In is wired up end-to-end.
func (h *AuthHandler) DevAuthSeed(w http.ResponseWriter, r *http.Request) {
	const devAppleUserID = "dev-local-user"
	const devEmail = "dev@famcalinsta.local"
	const devTokenBalance = 100

	user, err := h.userRepo.FindByAppleUserID(r.Context(), devAppleUserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if user == nil {
		user = &domain.User{
			AppleUserID:  devAppleUserID,
			Email:        devEmail,
			TokenBalance: devTokenBalance,
		}
		if err := h.userRepo.Create(r.Context(), user); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create dev user")
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

// POST /api/v1/auth/apple
func (h *AuthHandler) AppleSignIn(w http.ResponseWriter, r *http.Request) {
	var req appleAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.IdentityToken == "" {
		writeError(w, http.StatusBadRequest, "identity_token is required")
		return
	}

	claims, err := h.appleVerifier.Verify(r.Context(), req.IdentityToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid identity token")
		return
	}

	// Upsert user — find existing or create new
	user, err := h.userRepo.FindByAppleUserID(r.Context(), claims.Sub)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	if user == nil {
		user = &domain.User{
			AppleUserID:  claims.Sub,
			Email:        claims.Email,
			TokenBalance: 0,
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
