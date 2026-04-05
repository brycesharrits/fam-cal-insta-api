package v1

import (
	"encoding/json"
	"net/http"
	"time"

	apimiddleware "github.com/brycesharrits/fam-cal-insta/internal/api/middleware"
	"github.com/brycesharrits/fam-cal-insta/internal/domain"
	"github.com/brycesharrits/fam-cal-insta/internal/repository"
)

// TokenProduct represents an IAP product that grants tokens.
// These must match your App Store Connect product IDs.
var tokenProducts = []tokenProductDTO{
	{ProductID: "com.famcalinsta.tokens.50", TokenAmount: 50, DisplayName: "50 Tokens", DisplayPrice: "$0.99"},
	{ProductID: "com.famcalinsta.tokens.150", TokenAmount: 150, DisplayName: "150 Tokens", DisplayPrice: "$1.99"},
	{ProductID: "com.famcalinsta.tokens.500", TokenAmount: 500, DisplayName: "500 Tokens", DisplayPrice: "$4.99"},
}

type tokenProductDTO struct {
	ProductID    string `json:"product_id"`
	TokenAmount  int    `json:"token_amount"`
	DisplayName  string `json:"display_name"`
	DisplayPrice string `json:"display_price"`
}

type TokenHandler struct {
	userRepo  repository.UserRepository
	tokenRepo repository.TokenRepository
}

func NewTokenHandler(userRepo repository.UserRepository, tokenRepo repository.TokenRepository) *TokenHandler {
	return &TokenHandler{userRepo: userRepo, tokenRepo: tokenRepo}
}

// GET /api/v1/tokens/products
func (h *TokenHandler) GetProducts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, tokenProducts)
}

// GET /api/v1/tokens/balance
func (h *TokenHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := apimiddleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	balance, err := h.tokenRepo.GetBalance(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch balance")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"balance": balance})
}

// GET /api/v1/tokens/history
func (h *TokenHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := apimiddleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	txs, err := h.tokenRepo.FindByUserID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch history")
		return
	}

	type txDTO struct {
		ID          string    `json:"id"`
		Amount      int       `json:"amount"`
		Type        string    `json:"type"`
		Description string    `json:"description"`
		CreatedAt   time.Time `json:"created_at"`
	}

	dtos := make([]txDTO, 0, len(txs))
	for _, tx := range txs {
		dtos = append(dtos, txDTO{
			ID:          tx.ID,
			Amount:      tx.Amount,
			Type:        string(tx.Type),
			Description: tx.Description,
			CreatedAt:   tx.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, dtos)
}

// POST /api/v1/tokens/verify-purchase
// Receives a StoreKit 2 transaction ID from the client, verifies with Apple,
// and credits the user's token balance.
func (h *TokenHandler) VerifyPurchase(w http.ResponseWriter, r *http.Request) {
	userID, ok := apimiddleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body struct {
		TransactionID string `json:"transaction_id"`
		ProductID     string `json:"product_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Find the product config
	var product *tokenProductDTO
	for _, p := range tokenProducts {
		if p.ProductID == body.ProductID {
			pp := p
			product = &pp
			break
		}
	}
	if product == nil {
		writeError(w, http.StatusBadRequest, "unknown product_id")
		return
	}

	// TODO: Verify transaction_id with Apple App Store Server API
	// For now, trust the client (acceptable for development; add Apple verification before launch)

	// Credit tokens
	if err := h.userRepo.UpdateTokenBalance(r.Context(), userID, product.TokenAmount); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to credit tokens")
		return
	}

	// Record transaction
	tx := &domain.TokenTransaction{
		UserID:      userID,
		Amount:      product.TokenAmount,
		Type:        domain.TransactionTypePurchase,
		Description: "IAP: " + product.DisplayName + " (txn: " + body.TransactionID + ")",
	}
	_ = h.tokenRepo.RecordTransaction(r.Context(), tx)

	balance, _ := h.tokenRepo.GetBalance(r.Context(), userID)
	writeJSON(w, http.StatusOK, map[string]int{
		"token_balance": balance,
		"credited":      product.TokenAmount,
	})
}
