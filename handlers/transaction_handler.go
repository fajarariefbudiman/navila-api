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

func (h *TransactionHandler) Create(c echo.Context) error {
	userID := c.Get("userId").(string)
	var body models.Transaction
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	body.ID = genID("t")
	body.UserID = userID
	body.CreatedAt = time.Now()

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&body).Error; err != nil {
			return err
		}
		delta := body.Amount
		if body.Type == "withdraw" {
			delta = -body.Amount
		}
		return tx.Model(&models.Wallet{}).Where("id = ?", body.WalletID).
			Update("balance", gorm.Expr("balance + ?", delta)).Error
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create transaction"})
	}
	return c.JSON(http.StatusCreated, body)
}

func (h *TransactionHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	var t models.Transaction
	if err := h.DB.First(&t, "id = ?", id).Error; err != nil {
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
