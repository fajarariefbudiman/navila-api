package handlers

import (
	"net/http"
	"time"

	"goals-api/models"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type GoalHandler struct{ DB *gorm.DB }

func (h *GoalHandler) List(c echo.Context) error {
	userID := c.Get("userId").(string)
	var rows []models.Goal
	h.DB.Where("user_id = ?", userID).Order("created_at desc").Find(&rows)
	return c.JSON(http.StatusOK, rows)
}

func (h *GoalHandler) Create(c echo.Context) error {
	userID := c.Get("userId").(string)
	var body models.Goal
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	body.ID = genID("g")
	body.UserID = userID
	body.CreatedAt = time.Now()
	if err := h.DB.Create(&body).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create goal"})
	}
	return c.JSON(http.StatusCreated, body)
}

func (h *GoalHandler) Update(c echo.Context) error {
	id := c.Param("id")
	var patch map[string]interface{}
	if err := c.Bind(&patch); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	delete(patch, "id")
	delete(patch, "userId")
	h.DB.Model(&models.Goal{}).Where("id = ?", id).Updates(patch)
	var g models.Goal
	h.DB.First(&g, "id = ?", id)
	return c.JSON(http.StatusOK, g)
}

func (h *GoalHandler) Delete(c echo.Context) error {
	h.DB.Where("id = ?", c.Param("id")).Delete(&models.Goal{})
	return c.JSON(http.StatusOK, echo.Map{"deleted": true})
}
