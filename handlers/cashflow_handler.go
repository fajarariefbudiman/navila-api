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

func (h *CashflowHandler) Create(c echo.Context) error {
	userID := c.Get("userId").(string)
	var body models.CashFlow
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	body.ID = genID("cf")
	body.UserID = userID
	body.CreatedAt = time.Now()
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
