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

	// ✅ FITUR: kalau pos ini sudah ditautkan ke sebuah wallet (tabungan),
	// dan nilai `paid` NAIK dibanding sebelumnya, potong saldo wallet itu
	// sejumlah kenaikannya -- persis seperti "uangnya dari tabungan
	// berkurang sesuai akunnya" begitu item ditandai lunas/dibayar.
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

// toFloat: helper karena map[string]interface{} dari JSON decode angka
// jadi float64, tapi kita tetap jaga-jaga untuk tipe lain.
func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	}
	return 0, false
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