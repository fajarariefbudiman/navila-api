package handlers

import (
	"net/http"
	"time"

	"goals-api/models"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type BudgetHandler struct{ DB *gorm.DB }

func (h *BudgetHandler) List(c echo.Context) error {
	userID := c.Get("userId").(string)
	period := c.QueryParam("period")
	q := h.DB.Where("user_id = ?", userID)
	if period != "" {
		q = q.Where("period = ?", period)
	}
	var budgets []models.Budget
	q.Order("created_at asc").Find(&budgets)
	return c.JSON(http.StatusOK, budgets)
}

func (h *BudgetHandler) Create(c echo.Context) error {
	userID := c.Get("userId").(string)
	var body models.Budget
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	body.ID = genID("b")
	body.UserID = userID
	body.CreatedAt = time.Now()
	if err := h.DB.Create(&body).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create budget"})
	}
	return c.JSON(http.StatusCreated, body)
}

func (h *BudgetHandler) Update(c echo.Context) error {
	id := c.Param("id")
	var patch map[string]interface{}
	if err := c.Bind(&patch); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	delete(patch, "id")
	delete(patch, "userId")
	h.DB.Model(&models.Budget{}).Where("id = ?", id).Updates(patch)
	var b models.Budget
	h.DB.First(&b, "id = ?", id)
	return c.JSON(http.StatusOK, b)
}

func (h *BudgetHandler) Delete(c echo.Context) error {
	h.DB.Where("id = ?", c.Param("id")).Delete(&models.Budget{})
	return c.JSON(http.StatusOK, echo.Map{"deleted": true})
}
