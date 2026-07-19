package handlers

import (
	"net/http"
	"time"

	"goals-api/models"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type WeddingHandler struct{ DB *gorm.DB }

func (h *WeddingHandler) List(c echo.Context) error {
	userID := c.Get("userId").(string)
	var u models.User
	h.DB.First(&u, "id = ?", userID)
	ids := []string{userID}
	if u.PartnerID != nil {
		ids = append(ids, *u.PartnerID)
	}
	var rows []models.WeddingCategory
	h.DB.Where("couple_id IN ?", ids).Order("created_at asc").Find(&rows)
	return c.JSON(http.StatusOK, rows)
}

type weddingCreateReq struct {
	Name   string  `json:"name"`
	Budget float64 `json:"budget"`
	Paid   float64 `json:"paid"`
	Vendor *string `json:"vendor"`
	Status string  `json:"status"`
	Notes  *string `json:"notes"`
	Icon   string  `json:"icon"`
}

func (h *WeddingHandler) Create(c echo.Context) error {
	userID := c.Get("userId").(string)
	var req weddingCreateReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	body := models.WeddingCategory{
		ID:        genID("wc"),
		CoupleID:  userID,
		Name:      req.Name,
		Budget:    req.Budget,
		Paid:      req.Paid,
		Vendor:    req.Vendor,
		Status:    req.Status,
		Notes:     req.Notes,
		Icon:      req.Icon,
		CreatedAt: time.Now(),
	}
	if err := h.DB.Create(&body).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create"})
	}
	return c.JSON(http.StatusCreated, body)
}

func (h *WeddingHandler) Update(c echo.Context) error {
	userID := c.Get("userId").(string)
	id := c.Param("id")
	ids := h.coupleIDs(userID)
	var patch map[string]interface{}
	if err := c.Bind(&patch); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	delete(patch, "id")
	delete(patch, "coupleId")
	h.DB.Model(&models.WeddingCategory{}).Where("id = ? AND couple_id IN ?", id, ids).Updates(patch)
	var w models.WeddingCategory
	if err := h.DB.First(&w, "id = ? AND couple_id IN ?", id, ids).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "not found"})
	}
	return c.JSON(http.StatusOK, w)
}

func (h *WeddingHandler) Delete(c echo.Context) error {
	userID := c.Get("userId").(string)
	ids := h.coupleIDs(userID)
	h.DB.Where("id = ? AND couple_id IN ?", c.Param("id"), ids).Delete(&models.WeddingCategory{})
	return c.JSON(http.StatusOK, echo.Map{"deleted": true})
}

func (h *WeddingHandler) coupleIDs(userID string) []string {
	var u models.User
	h.DB.First(&u, "id = ?", userID)
	ids := []string{userID}
	if u.PartnerID != nil {
		ids = append(ids, *u.PartnerID)
	}
	return ids
}