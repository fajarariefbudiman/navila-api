package handlers

import (
	"encoding/json"
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

type planCreateReq struct {
	Type        string          `json:"type"`
	Title       string          `json:"title"`
	Description *string         `json:"description"`
	Category    string          `json:"category"`
	TargetDate  *string         `json:"targetDate"`
	Budget      *float64        `json:"budget"`
	Status      string          `json:"status"`
	Progress    int             `json:"progress"`
	Subtasks    models.Subtasks `json:"subtasks"`
}

func (h *PlanHandler) Create(c echo.Context) error {
	userID := c.Get("userId").(string)
	var req planCreateReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	// ✅ Sama seperti Goal -- kalau ada subtasks, progress dihitung dari
	// situ, bukan dari nilai manual.
	progress := req.Progress
	if p := computeProgressFromSubtasks(req.Subtasks); p >= 0 {
		progress = p
	}
	body := models.Plan{
		ID:          genID("p"),
		UserID:      userID,
		Type:        req.Type,
		Title:       req.Title,
		Description: req.Description,
		Category:    req.Category,
		TargetDate:  req.TargetDate,
		Budget:      req.Budget,
		Status:      req.Status,
		Progress:    progress,
		Subtasks:    req.Subtasks,
		CreatedAt:   time.Now(),
	}
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
	// BUG FIX: same as goal_handler.go — the Flutter client always sends
	// `createdAt` as an ISO8601 string on every update, which MySQL's
	// DATETIME column rejects, breaking the whole UPDATE. Strip it; it
	// should never change after creation anyway.
	delete(patch, "createdAt")

	// BUG FIX: same issue as goal_handler.go — a JSON array inside
	// map[string]interface{} decodes to []interface{}, which GORM can't
	// convert into the `subtasks` JSON column, so the whole UPDATE
	// silently failed (error was never checked) and checklist/status
	// changes never actually persisted.
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

		// ✅ FITUR: progress otomatis sinkron dengan checklist, sama
		// persis seperti Goal.
		if p := computeProgressFromSubtasks(subtasks); p >= 0 {
			patch["progress"] = p
		}
	}

	// BUG FIX: this is the real reason updates always failed, even after
	// the fixes above. GORM's Updates() with a raw map[string]interface{}
	// does NOT translate camelCase map keys to the actual snake_case DB
	// column names — it uses the key literally. The Flutter client always
	// sends `targetDate` (camelCase), but the real column is
	// `target_date`, so MySQL rejected every single update with
	// "Unknown column 'targetDate' in 'SET'". Normalize every key to
	// snake_case so it actually matches the DB column.
	patch = camelKeysToSnake(patch)

	if err := h.DB.Model(&models.Plan{}).Where("id = ? AND user_id IN ?", id, []string{userID, partnerID}).Updates(patch).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update plan"})
	}
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