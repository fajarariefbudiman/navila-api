package handlers

import (
	"net/http"
	"time"

	"goals-api/models"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type TransactionHandler struct{ DB *gorm.DB }

func (h *TransactionHandler) List(c echo.Context) error {
	userID := c.Get("userId").(string)
	var txs []models.Transaction
	h.DB.Where("user_id = ?", userID).Order("date desc").Find(&txs)
	return c.JSON(http.StatusOK, txs)
}

type transactionCreateReq struct {
	WalletID              string  `json:"walletId"`
	Type                  string  `json:"type"`
	Amount                float64 `json:"amount"`
	Category              string  `json:"category"`
	Note                  *string `json:"note"`
	Date                  string  `json:"date"`
	IsWeddingContribution bool    `json:"isWeddingContribution"`
	// ContributeToMainGoal: dijawab "ya" saat prompt "Masukkan ke goals
	// anda?" muncul ketika user menambah saldo (deposit) di wallet ini.
	ContributeToMainGoal bool `json:"contributeToMainGoal"`
}

func (h *TransactionHandler) Create(c echo.Context) error {
	userID := c.Get("userId").(string)
	var req transactionCreateReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	body := models.Transaction{
		ID:                    genID("t"),
		UserID:                userID,
		WalletID:              req.WalletID,
		Type:                  req.Type,
		Amount:                req.Amount,
		Category:              req.Category,
		Note:                  req.Note,
		Date:                  req.Date,
		IsWeddingContribution: req.IsWeddingContribution,
		CreatedAt:             time.Now(),
	}

	var updatedGoal *models.MainGoal
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&body).Error; err != nil {
			return err
		}
		delta := body.Amount
		if body.Type == "withdraw" {
			delta = -body.Amount
		}
		if err := tx.Model(&models.Wallet{}).Where("id = ?", body.WalletID).
			Update("balance", gorm.Expr("balance + ?", delta)).Error; err != nil {
			return err
		}
		// ✅ FITUR: kalau user pilih "Masukkan ke goals anda?" saat nambah
		// saldo, dana ini juga ikut nambah ke Goal Utama Pasangan yang
		// sedang aktif (kalau ada).
		if req.ContributeToMainGoal && body.Type == "deposit" {
			g, err := contributeToMainGoal(tx, userID, body.Amount, "wallet_deposit", body.ID)
			if err != nil {
				return err
			}
			updatedGoal = g
		}
		return nil
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create transaction"})
	}
	return c.JSON(http.StatusCreated, echo.Map{
		"transaction": body,
		"mainGoal":    updatedGoal,
	})
}

// ListWeddingContributions - GET /transactions/wedding
// Ambil semua transaksi yang ditandai sebagai kontribusi dana nikah
func (h *TransactionHandler) ListWeddingContributions(c echo.Context) error {
	userID := c.Get("userId").(string)
	var txs []models.Transaction
	h.DB.Where("user_id = ? AND is_wedding_contribution = ?", userID, true).
		Order("date desc").Find(&txs)
	return c.JSON(http.StatusOK, txs)
}

func (h *TransactionHandler) Delete(c echo.Context) error {
	userID := c.Get("userId").(string)
	id := c.Param("id")
	var t models.Transaction
	if err := h.DB.First(&t, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "not found"})
	}
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&t).Error; err != nil {
			return err
		}
		delta := -t.Amount
		if t.Type == "withdraw" {
			delta = t.Amount
		}
		return tx.Model(&models.Wallet{}).Where("id = ?", t.WalletID).
			Update("balance", gorm.Expr("balance + ?", delta)).Error
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to delete transaction"})
	}
	return c.JSON(http.StatusOK, echo.Map{"deleted": true})
}