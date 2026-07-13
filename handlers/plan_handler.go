package handlers

import (
	"net/http"
	"time"

	"goals-api/models"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type PlanHandler struct{ DB *gorm.DB }

func (h *PlanHandler) List(c echo.Context) error {
	userID := c.Get("userId").(string)
	var u models.User
	h.DB.First(&u, "id = ?", userID)
	partnerID := ""
	if u.PartnerID != nil {
		partnerID = *u.PartnerID
	}
	var rows []models.Plan
	h.DB.Where(
		"(type = 'together' AND user_id IN ?) OR (type = 'mine' AND user_id = ?) OR (type = 'partner' AND user_id IN ?)",
		[]string{userID, partnerID}, userID, []string{userID, partnerID},
	).Order("created_at desc").Find(&rows)
	return c.JSON(http.StatusOK, rows)
}

func (h *PlanHandler) Create(c echo.Context) error {
	userID := c.Get("userId").(string)
	var body models.Plan
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	body.ID = genID("p")
	body.UserID = userID
	body.CreatedAt = time.Now()
	if err := h.DB.Create(&body).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create plan"})
	}
	return c.JSON(http.StatusCreated, body)
}

func (h *PlanHandler) Update(c echo.Context) error {
	userID := c.Get("userId").(string)
	id := c.Param("id")
	var u models.User
	h.DB.First(&u, "id = ?", userID)
	partnerID := ""
	if u.PartnerID != nil {
		partnerID = *u.PartnerID
	}
	var patch map[string]interface{}
	if err := c.Bind(&patch); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	delete(patch, "id")
	delete(patch, "userId")
	h.DB.Model(&models.Plan{}).Where("id = ? AND user_id IN ?", id, []string{userID, partnerID}).Updates(patch)
	var p models.Plan
	if err := h.DB.First(&p, "id = ? AND user_id IN ?", id, []string{userID, partnerID}).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "not found"})
	}
	return c.JSON(http.StatusOK, p)
}

func (h *PlanHandler) Delete(c echo.Context) error {
	userID := c.Get("userId").(string)
	id := c.Param("id")
	var u models.User
	h.DB.First(&u, "id = ?", userID)
	partnerID := ""
	if u.PartnerID != nil {
		partnerID = *u.PartnerID
	}
	h.DB.Where("id = ? AND user_id IN ?", id, []string{userID, partnerID}).Delete(&models.Plan{})
	return c.JSON(http.StatusOK, echo.Map{"deleted": true})
}
