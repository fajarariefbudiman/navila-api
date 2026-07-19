package handlers

import (
	"net/http"
	"time"

	"goals-api/models"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type WalletHandler struct{ DB *gorm.DB }

func (h *WalletHandler) List(c echo.Context) error {
	userID := c.Get("userId").(string)
	var wallets []models.Wallet
	h.DB.Where("user_id = ?", userID).Order("created_at asc").Find(&wallets)
	return c.JSON(http.StatusOK, wallets)
}

type walletCreateReq struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Balance      float64  `json:"balance"`
	TargetAmount *float64 `json:"targetAmount"`
	Color        string   `json:"color"`
}

func (h *WalletHandler) Create(c echo.Context) error {
	userID := c.Get("userId").(string)
	var req walletCreateReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	body := models.Wallet{
		ID:           genID("w"),
		UserID:       userID,
		Name:         req.Name,
		Type:         req.Type,
		Balance:      req.Balance,
		TargetAmount: req.TargetAmount,
		Color:        req.Color,
		CreatedAt:    time.Now(),
	}
	if err := h.DB.Create(&body).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create wallet"})
	}
	return c.JSON(http.StatusCreated, body)
}

func (h *WalletHandler) Update(c echo.Context) error {
	userID := c.Get("userId").(string)
	id := c.Param("id")
	var patch map[string]interface{}
	if err := c.Bind(&patch); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	delete(patch, "id")
	delete(patch, "userId")
	if err := h.DB.Model(&models.Wallet{}).Where("id = ? AND user_id = ?", id, userID).Updates(patch).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update wallet"})
	}
	var w models.Wallet
	if err := h.DB.First(&w, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "not found"})
	}
	return c.JSON(http.StatusOK, w)
}

func (h *WalletHandler) Delete(c echo.Context) error {
	userID := c.Get("userId").(string)
	id := c.Param("id")
	var w models.Wallet
	if err := h.DB.First(&w, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "not found"})
	}
	h.DB.Where("wallet_id = ?", id).Delete(&models.Transaction{})
	h.DB.Delete(&w)
	return c.JSON(http.StatusOK, echo.Map{"deleted": true})
}