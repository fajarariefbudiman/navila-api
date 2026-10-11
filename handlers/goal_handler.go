package handlers

import (
	"encoding/json"
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

type goalCreateReq struct {
	Title        string          `json:"title"`
	Description  *string         `json:"description"`
	Category     string          `json:"category"`
	Priority     string          `json:"priority"`
	Status       string          `json:"status"`
	Progress     int             `json:"progress"`
	Deadline     *string         `json:"deadline"`
	TargetAmount *float64        `json:"targetAmount"`
	Notes        *string         `json:"notes"`
	Subtasks     models.Subtasks `json:"subtasks"`
}

func (h *GoalHandler) Create(c echo.Context) error {
	userID := c.Get("userId").(string)
	var req goalCreateReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	progress := req.Progress
	if p := computeProgressFromSubtasks(req.Subtasks); p >= 0 {
		progress = p
	}
	body := models.Goal{
		ID:           genID("g"),
		UserID:       userID,
		Title:        req.Title,
		Description:  req.Description,
		Category:     req.Category,
		Priority:     req.Priority,
		Status:       req.Status,
		Progress:     progress,
		Deadline:     req.Deadline,
		TargetAmount: req.TargetAmount,
		Notes:        req.Notes,
		Subtasks:     req.Subtasks,
		CreatedAt:    time.Now(),
	}
	if err := h.DB.Create(&body).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create goal"})
	}
	return c.JSON(http.StatusCreated, body)
}

func (h *GoalHandler) Update(c echo.Context) error {
	userID := c.Get("userId").(string)
	id := c.Param("id")
	var patch map[string]interface{}
	if err := c.Bind(&patch); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	delete(patch, "id")
	delete(patch, "userId")
	// BUG FIX: the Flutter client always sends the full object on every
	// update, including `createdAt` as an ISO8601 string (e.g.
	// "2026-07-19T10:23:00.000"). MySQL's DATETIME column expects
	// "YYYY-MM-DD HH:MM:SS" (no "T"), so leaving this in the patch made
	// GORM's UPDATE fail on *every* request — this field should never be
	// mutated via update anyway.
	delete(patch, "createdAt")

	// BUG FIX: a JSON array inside map[string]interface{} decodes to
	// []interface{}, which GORM cannot convert into the `subtasks` JSON
	// column (that only works via models.Subtasks' driver.Valuer). Passing
	// it as-is made the *entire* UPDATE statement fail — silently, since
	// the error below was never checked — so status/progress/subtask
	// changes never actually saved and the old row was returned instead.
	if raw, ok := patch["subtasks"]; ok {
		b, err := json.Marshal(raw)
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid subtasks payload"})
		}
		var subtasks models.Subtasks
		if err := json.Unmarshal(b, &subtasks); err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid subtasks payload"})
		}
		patch["subtasks"] = subtasks

		if p := computeProgressFromSubtasks(subtasks); p >= 0 {
			patch["progress"] = p
		}
	}

	patch = camelKeysToSnake(patch)

	if err := h.DB.Model(&models.Goal{}).Where("id = ? AND user_id = ?", id, userID).Updates(patch).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update goal"})
	}
	var g models.Goal
	if err := h.DB.First(&g, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "not found"})
	}
	return c.JSON(http.StatusOK, g)
}

func (h *GoalHandler) Delete(c echo.Context) error {
	userID := c.Get("userId").(string)
	h.DB.Where("id = ? AND user_id = ?", c.Param("id"), userID).Delete(&models.Goal{})
	return c.JSON(http.StatusOK, echo.Map{"deleted": true})
}