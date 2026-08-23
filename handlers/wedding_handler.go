package handlers

import (
	"net/http"
	"time"

	"goals-api/models"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// WeddingHandler - ORIGINAL + ENHANCED ENDPOINTS
//
// ORIGINAL Endpoints (ALREADY EXIST):
// - POST   /wedding              → Create
// - GET    /wedding              → List
// - PATCH  /wedding/:id          → Update
// - DELETE /wedding/:id          → Delete
//
// ENHANCED Endpoints (OPTIONAL, can add):
// - GET    /wedding/:id          → GetByID
// - GET    /wedding/filter       → Filter by status/name/vendor
// - GET    /wedding/summary      → Budget summary
// - PATCH  /wedding/bulk         → Bulk update multiple items
// - DELETE /wedding/bulk         → Bulk delete multiple items

type WeddingHandler struct{ DB *gorm.DB }

// ============================================================
// ORIGINAL ENDPOINTS (dari existing code)
// ============================================================

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

	var before models.WeddingCategory
	if err := h.DB.First(&before, "id = ? AND couple_id IN ?", id, ids).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "not found"})
	}

	var patch map[string]interface{}
	if err := c.Bind(&patch); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	delete(patch, "id")
	delete(patch, "coupleId")
	patch = camelKeysToSnake(patch)

	walletID := before.WalletID
	if wid, ok := patch["wallet_id"]; ok {
		if s, ok2 := wid.(string); ok2 && s != "" {
			walletID = &s
		}
	}
	var paidDelta float64
	if rawPaid, ok := patch["paid"]; ok {
		newPaid, ok2 := toFloat(rawPaid)
		if ok2 {
			paidDelta = newPaid - before.Paid
		}
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.WeddingCategory{}).Where("id = ? AND couple_id IN ?", id, ids).Updates(patch).Error; err != nil {
			return err
		}
		if paidDelta != 0 && walletID != nil && *walletID != "" {
			if err := tx.Model(&models.Wallet{}).Where("id = ?", *walletID).
				Update("balance", gorm.Expr("balance - ?", paidDelta)).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update"})
	}

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

// ============================================================
// HELPER FUNCTIONS
// ============================================================

func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	}
	return 0, false
}

// ============================================================
// ENHANCED ENDPOINTS (OPTIONAL - Recommended to add)
// ============================================================

// GetByID - GET /wedding/:id
// Retrieve single wedding item by ID
// Usage: GET /wedding/wc_xyz123
func (h *WeddingHandler) GetByID(c echo.Context) error {
	userID := c.Get("userId").(string)
	id := c.Param("id")
	ids := h.coupleIDs(userID)

	var w models.WeddingCategory
	if err := h.DB.First(&w, "id = ? AND couple_id IN ?", id, ids).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "not found"})
	}
	return c.JSON(http.StatusOK, w)
}

// Filter - GET /wedding/filter
// Filter by status, name, vendor
// Usage: 
//   GET /wedding/filter?status=pending
//   GET /wedding/filter?vendor=amin
//   GET /wedding/filter?name=catering&status=completed
func (h *WeddingHandler) Filter(c echo.Context) error {
	userID := c.Get("userId").(string)
	ids := h.coupleIDs(userID)

	query := h.DB.Where("couple_id IN ?", ids)

	// Filter by status (exact match)
	if status := c.QueryParam("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// Filter by vendor (case-insensitive LIKE)
	if vendor := c.QueryParam("vendor"); vendor != "" {
		query = query.Where("vendor ILIKE ?", "%"+vendor+"%")
	}

	// Filter by name (case-insensitive LIKE)
	if name := c.QueryParam("name"); name != "" {
		query = query.Where("name ILIKE ?", "%"+name+"%")
	}

	var rows []models.WeddingCategory
	if err := query.Order("created_at asc").Find(&rows).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to fetch"})
	}

	return c.JSON(http.StatusOK, rows)
}

// GetSummary - GET /wedding/summary
// Get total budget, paid, and remaining amount
// Response: { "totalBudget": 100000000, "totalPaid": 50000000, "remaining": 50000000 }
func (h *WeddingHandler) GetSummary(c echo.Context) error {
	userID := c.Get("userId").(string)
	ids := h.coupleIDs(userID)

	type Summary struct {
		TotalBudget float64 `json:"totalBudget"`
		TotalPaid   float64 `json:"totalPaid"`
		Remaining   float64 `json:"remaining"`
		ItemCount   int     `json:"itemCount"`
	}

	var counts int64
	var totalBudget, totalPaid float64

	h.DB.Model(&models.WeddingCategory{}).
		Where("couple_id IN ?", ids).
		Count(&counts)

	h.DB.Model(&models.WeddingCategory{}).
		Where("couple_id IN ?", ids).
		Select("COALESCE(SUM(budget), 0) as total_budget, COALESCE(SUM(paid), 0) as total_paid").
		Row().
		Scan(&totalBudget, &totalPaid)

	summary := Summary{
		TotalBudget: totalBudget,
		TotalPaid:   totalPaid,
		Remaining:   totalBudget - totalPaid,
		ItemCount:   int(counts),
	}

	return c.JSON(http.StatusOK, summary)
}

// BulkUpdate - PATCH /wedding/bulk
// Update multiple wedding items at once
// Request: {
//   "updates": {
//     "wc_1": { "status": "completed" },
//     "wc_2": { "paid": 5000000, "status": "in_progress" }
//   }
// }
func (h *WeddingHandler) BulkUpdate(c echo.Context) error {
	userID := c.Get("userId").(string)
	ids := h.coupleIDs(userID)

	type BulkUpdateReq struct {
		Updates map[string]map[string]interface{} `json:"updates"`
	}
	var req BulkUpdateReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}

	if len(req.Updates) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "no items to update"})
	}

	updated := 0
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		for id, patch := range req.Updates {
			// Convert camelCase to snake_case
			patch = camelKeysToSnake(patch)

			// Clean up fields that shouldn't be updated
			delete(patch, "id")
			delete(patch, "coupleId")
			delete(patch, "createdAt")

			result := tx.Model(&models.WeddingCategory{}).
				Where("id = ? AND couple_id IN ?", id, ids).
				Updates(patch)

			if result.Error != nil {
				return result.Error
			}
			updated += int(result.RowsAffected)
		}
		return nil
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update"})
	}

	return c.JSON(http.StatusOK, echo.Map{"updated": updated})
}

// BulkDelete - DELETE /wedding/bulk
// Delete multiple wedding items at once
// Request: { "ids": ["wc_1", "wc_2", "wc_3"] }
func (h *WeddingHandler) BulkDelete(c echo.Context) error {
	userID := c.Get("userId").(string)
	ids := h.coupleIDs(userID)

	type BulkDeleteReq struct {
		IDs []string `json:"ids"`
	}
	var req BulkDeleteReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}

	if len(req.IDs) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "no items to delete"})
	}

	result := h.DB.Where("id IN ? AND couple_id IN ?", req.IDs, ids).
		Delete(&models.WeddingCategory{})

	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to delete"})
	}

	return c.JSON(http.StatusOK, echo.Map{"deleted": result.RowsAffected})
}

