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
	if err := h.DB.Create(&body).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create cashflow"})
	}
	return c.JSON(http.StatusCreated, body)
}

func (h *CashflowHandler) Delete(c echo.Context) error {
	userID := c.Get("userId").(string)
	h.DB.Where("id = ? AND user_id = ?", c.Param("id"), userID).Delete(&models.CashFlow{})
	return c.JSON(http.StatusOK, echo.Map{"deleted": true})
}