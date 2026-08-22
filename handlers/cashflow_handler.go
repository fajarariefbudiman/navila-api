package handlers

import (
	"net/http"
	"time"

	"goals-api/models"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type CashflowHandler struct{ DB *gorm.DB }

func (h *CashflowHandler) List(c echo.Context) error {
	userID := c.Get("userId").(string)
	var rows []models.CashFlow
	h.DB.Where("user_id = ?", userID).Order("date desc").Find(&rows)
	return c.JSON(http.StatusOK, rows)
}

type cashflowCreateReq struct {
	WalletID *string `json:"walletId"`
	Type     string  `json:"type"`
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
	Note     *string `json:"note"`
	Date     string  `json:"date"`
	// ContributeToMainGoal: sama seperti di transaction_handler.go -- opsi
	// "Masukkan ke goals anda?" untuk kas/pemasukan.
	ContributeToMainGoal bool `json:"contributeToMainGoal"`
}

func (h *CashflowHandler) Create(c echo.Context) error {
	userID := c.Get("userId").(string)
	var req cashflowCreateReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	body := models.CashFlow{
		ID:        genID("cf"),
		UserID:    userID,
		WalletID:  req.WalletID,
		Type:      req.Type,
		Category:  req.Category,
		Amount:    req.Amount,
		Note:      req.Note,
		Date:      req.Date,
		CreatedAt: time.Now(),
	}
	var updatedGoal *models.MainGoal
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&body).Error; err != nil {
			return err
		}
		// BUG FIX: kalau cashflow ditautkan ke sebuah wallet, saldo wallet
		// di DATABASE juga harus disesuaikan di sini — sebelumnya endpoint
		// ini cuma insert baris cashflow, sehingga saldo yang "berubah" di
		// app cuma state lokal (SharedPreferences). Begitu app fetch ulang
		// wallet dari server, saldo balik seperti semula karena DB tidak
		// pernah diupdate. Pola ini menyamakan dengan transaction_handler.go.
		if body.WalletID != nil && *body.WalletID != "" {
			delta := body.Amount
			if body.Type == "expense" {
				delta = -body.Amount
			}
			if err := tx.Model(&models.Wallet{}).Where("id = ?", *body.WalletID).
				Update("balance", gorm.Expr("balance + ?", delta)).Error; err != nil {
				return err
			}
		}
		// ✅ FITUR: opsi "Masukkan ke goals anda?" untuk pemasukan (income).
		if req.ContributeToMainGoal && body.Type == "income" {
			g, err := contributeToMainGoal(tx, userID, body.Amount, "cashflow_income", body.ID, body.WalletID)
			if err != nil {
				return err
			}
			updatedGoal = g
		}
		return nil
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create cashflow"})
	}
	return c.JSON(http.StatusCreated, echo.Map{
		"cashflow": body,
		"mainGoal": updatedGoal,
	})
}

func (h *CashflowHandler) Delete(c echo.Context) error {
	userID := c.Get("userId").(string)
	id := c.Param("id")
	// BUG FIX: parameter `rollback` yang dikirim app (lihat api_service.dart
	// `deleteCashFlow`) sebelumnya diabaikan total. Default true supaya
	// perilaku sama seperti sebelumnya kalau param tidak dikirim.
	rollback := c.QueryParam("rollback") != "false"

	var cf models.CashFlow
	if err := h.DB.First(&cf, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "not found"})
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&cf).Error; err != nil {
			return err
		}
		// BUG FIX: sebelumnya saldo wallet di DB tidak pernah dikembalikan
		// saat entri arus kas dihapus, sama seperti bug di Create.
		if rollback && cf.WalletID != nil && *cf.WalletID != "" {
			delta := -cf.Amount
			if cf.Type == "expense" {
				delta = cf.Amount
			}
			return tx.Model(&models.Wallet{}).Where("id = ?", *cf.WalletID).
				Update("balance", gorm.Expr("balance + ?", delta)).Error
		}
		return nil
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to delete cashflow"})
	}
	return c.JSON(http.StatusOK, echo.Map{"deleted": true})
}