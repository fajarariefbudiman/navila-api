package handlers

import (
	"net/http"
	"time"

	"goals-api/models"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type FundRequestHandler struct{ DB *gorm.DB }

// List mengembalikan semua request yang masih pending & melibatkan user ini
// (baik yang dia kirim maupun yang dia terima), supaya Flutter bisa
// menampilkan "menunggu persetujuan" di kedua sisi.
func (h *FundRequestHandler) List(c echo.Context) error {
	userID := c.Get("userId").(string)
	var rows []models.FundChangeRequest
	h.DB.Where("(from_user_id = ? OR to_user_id = ?) AND status = 'pending'", userID, userID).
		Order("created_at desc").Find(&rows)
	return c.JSON(http.StatusOK, rows)
}

// Create membuat permintaan perubahan target dana. Wajib sudah punya
// pasangan (targetWeddingFund adalah nilai bersama, bukan milik satu orang).
func (h *FundRequestHandler) Create(c echo.Context) error {
	userID := c.Get("userId").(string)
	var body struct {
		NewAmount float64 `json:"newAmount"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}

	var me models.User
	if err := h.DB.First(&me, "id = ?", userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "user not found"})
	}
	if me.PartnerID == nil || *me.PartnerID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "belum ada pasangan"})
	}

	req := models.FundChangeRequest{
		ID:         genID("fr"),
		FromUserID: userID,
		ToUserID:   *me.PartnerID,
		OldAmount:  me.TargetWeddingFund,
		NewAmount:  body.NewAmount,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}
	if err := h.DB.Create(&req).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create request"})
	}
	return c.JSON(http.StatusCreated, req)
}

// Approve HANYA boleh dipanggil oleh ToUserID (pihak yang diminta setuju).
// Begitu disetujui, targetWeddingFund kedua akun diupdate ke nilai baru.
func (h *FundRequestHandler) Approve(c echo.Context) error {
	return h.resolve(c, "approved")
}

// Reject juga hanya boleh dipanggil oleh ToUserID; tidak ada perubahan data.
func (h *FundRequestHandler) Reject(c echo.Context) error {
	return h.resolve(c, "rejected")
}

func (h *FundRequestHandler) resolve(c echo.Context, status string) error {
	userID := c.Get("userId").(string)
	id := c.Param("id")

	var req models.FundChangeRequest
	if err := h.DB.First(&req, "id = ? AND to_user_id = ? AND status = 'pending'", id, userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "request not found or already resolved"})
	}

	now := time.Now()
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&req).Updates(map[string]interface{}{
			"status":      status,
			"resolved_at": now,
		}).Error; err != nil {
			return err
		}
		if status == "approved" {
			if err := tx.Model(&models.User{}).Where("id IN ?", []string{req.FromUserID, req.ToUserID}).
				Update("target_wedding_fund", req.NewAmount).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to resolve request"})
	}
	return c.JSON(http.StatusOK, echo.Map{"status": status})
}