package handlers

import (
	"net/http"
	"time"

	"goals-api/models"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type HabitHandler struct {
	DB              *gorm.DB
	AnthropicAPIKey string
}

// GetProfile - GET /habit/profile
func (h *HabitHandler) GetProfile(c echo.Context) error {
	userID := c.Get("userId").(string)
	var p models.HabitProfile
	if err := h.DB.First(&p, "user_id = ?", userID).Error; err != nil {
		return c.JSON(http.StatusOK, nil)
	}
	return c.JSON(http.StatusOK, p)
}

type habitProfileReq struct {
	Salary         float64 `json:"salary"`
	Religion       string  `json:"religion"`
	RentBudget     float64 `json:"rentBudget"`
	FuelBudget     float64 `json:"fuelBudget"`
	FoodBudget     float64 `json:"foodBudget"`
	IsCollege      bool    `json:"isCollege"`
	HasFreelance   bool    `json:"hasFreelance"`
	FreelanceStart *string `json:"freelanceStart"`
	FreelanceEnd   *string `json:"freelanceEnd"`
}

// SaveProfile - POST /habit/profile
// Simpan/update profil, lalu generate ulang seluruh jadwal otomatis
// (Source="ai" atau "static"). Entry yang sudah diedit manual oleh user
// (Source="manual") TIDAK dihapus/ditimpa supaya perubahan user tetap
// dihormati.
func (h *HabitHandler) SaveProfile(c echo.Context) error {
	userID := c.Get("userId").(string)
	var req habitProfileReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}

	var existing models.HabitProfile
	isNew := h.DB.First(&existing, "user_id = ?", userID).Error != nil
	now := time.Now()
	profile := models.HabitProfile{
		ID: existing.ID, UserID: userID,
		Salary: req.Salary, Religion: req.Religion,
		RentBudget: req.RentBudget, FuelBudget: req.FuelBudget, FoodBudget: req.FoodBudget,
		IsCollege: req.IsCollege, HasFreelance: req.HasFreelance,
		FreelanceStart: req.FreelanceStart, FreelanceEnd: req.FreelanceEnd,
		UpdatedAt: now,
	}
	if isNew {
		profile.ID = genID("hp")
		profile.CreatedAt = now
		if err := h.DB.Create(&profile).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to save profile"})
		}
	} else {
		if err := h.DB.Model(&existing).Updates(profile).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to save profile"})
		}
	}

	source := "static"
	entries := generateSchedule(h.AnthropicAPIKey, profile)
	if h.AnthropicAPIKey != "" {
		source = "ai"
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND source != 'manual'", userID).Delete(&models.HabitScheduleEntry{}).Error; err != nil {
			return err
		}
		for _, e := range entries {
			row := models.HabitScheduleEntry{
				ID: genID("hs"), UserID: userID, DayOfWeek: e.DayOfWeek,
				StartTime: e.StartTime, EndTime: e.EndTime, Title: e.Title,
				Category: e.Category, Source: source, CreatedAt: now,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to generate schedule"})
	}
	return c.JSON(http.StatusOK, echo.Map{"profile": profile, "scheduleSource": source})
}

// GetSchedule - GET /habit/schedule : jadwal Senin-Minggu milik sendiri.
func (h *HabitHandler) GetSchedule(c echo.Context) error {
	userID := c.Get("userId").(string)
	var rows []models.HabitScheduleEntry
	h.DB.Where("user_id = ?", userID).Order("day_of_week asc, start_time asc").Find(&rows)
	return c.JSON(http.StatusOK, rows)
}

// GetPartnerSchedule - GET /habit/schedule/partner : jadwal pasangan,
// READ-ONLY (tidak ada endpoint edit yang menerima userId pasangan).
func (h *HabitHandler) GetPartnerSchedule(c echo.Context) error {
	userID := c.Get("userId").(string)
	var me models.User
	if err := h.DB.First(&me, "id = ?", userID).Error; err != nil || me.PartnerID == nil {
		return c.JSON(http.StatusOK, []models.HabitScheduleEntry{})
	}
	var rows []models.HabitScheduleEntry
	h.DB.Where("user_id = ?", *me.PartnerID).Order("day_of_week asc, start_time asc").Find(&rows)
	return c.JSON(http.StatusOK, rows)
}

type habitEntryReq struct {
	DayOfWeek int    `json:"dayOfWeek"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Title     string `json:"title"`
	Category  string `json:"category"`
}

// CreateEntry - POST /habit/schedule : tambah blok manual (mis. jadwal
// tambahan yang tidak kebayang saat generate awal).
func (h *HabitHandler) CreateEntry(c echo.Context) error {
	userID := c.Get("userId").(string)
	var req habitEntryReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	row := models.HabitScheduleEntry{
		ID: genID("hs"), UserID: userID, DayOfWeek: req.DayOfWeek,
		StartTime: req.StartTime, EndTime: req.EndTime, Title: req.Title,
		Category: req.Category, Source: "manual", CreatedAt: time.Now(),
	}
	if err := h.DB.Create(&row).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create entry"})
	}
	return c.JSON(http.StatusCreated, row)
}

// UpdateEntry - PATCH /habit/schedule/:id : user tetap bisa ubah jadwal
// hasil generate. Begitu diedit, Source dipaksa jadi "manual" supaya
// generate ulang berikutnya tidak menimpa perubahan ini.
func (h *HabitHandler) UpdateEntry(c echo.Context) error {
	userID := c.Get("userId").(string)
	id := c.Param("id")
	var patch map[string]interface{}
	if err := c.Bind(&patch); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	delete(patch, "id")
	delete(patch, "userId")
	delete(patch, "createdAt")
	patch = camelKeysToSnake(patch)
	patch["source"] = "manual"

	if err := h.DB.Model(&models.HabitScheduleEntry{}).Where("id = ? AND user_id = ?", id, userID).Updates(patch).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update entry"})
	}
	var row models.HabitScheduleEntry
	if err := h.DB.First(&row, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "not found"})
	}
	return c.JSON(http.StatusOK, row)
}

func (h *HabitHandler) DeleteEntry(c echo.Context) error {
	userID := c.Get("userId").(string)
	h.DB.Where("id = ? AND user_id = ?", c.Param("id"), userID).Delete(&models.HabitScheduleEntry{})
	return c.JSON(http.StatusOK, echo.Map{"deleted": true})
}
