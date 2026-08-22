package handlers

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"goals-api/models"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type MainGoalHandler struct{ DB *gorm.DB }

// coupleKey menghasilkan key yang sama persis dari sisi manapun goal ini
// diakses (baik dari user A maupun user B), supaya satu goal utama benar-
// benar dianggap "milik pasangan", bukan milik satu akun saja.
func coupleKey(userID string, partnerID *string) string {
	if partnerID == nil || *partnerID == "" {
		return userID
	}
	ids := []string{userID, *partnerID}
	sort.Strings(ids)
	return strings.Join(ids, "_")
}

func (h *MainGoalHandler) currentUser(c echo.Context) (models.User, error) {
	userID := c.Get("userId").(string)
	var u models.User
	err := h.DB.First(&u, "id = ?", userID).Error
	return u, err
}

// Get - GET /main-goal : goal aktif (atau paling baru) milik pasangan ini.
func (h *MainGoalHandler) Get(c echo.Context) error {
	me, err := h.currentUser(c)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "user not found"})
	}
	key := coupleKey(me.ID, me.PartnerID)
	var goal models.MainGoal
	if err := h.DB.Where("couple_key = ? AND status = 'active'", key).
		Order("created_at desc").First(&goal).Error; err != nil {
		return c.JSON(http.StatusOK, nil)
	}
	return c.JSON(http.StatusOK, goal)
}

// History - GET /main-goal/history : semua goal yang sudah achieved/archived
// (buat ditampilkan sebagai riwayat "goals yang pernah tercapai").
func (h *MainGoalHandler) History(c echo.Context) error {
	me, err := h.currentUser(c)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "user not found"})
	}
	key := coupleKey(me.ID, me.PartnerID)
	var rows []models.MainGoal
	h.DB.Where("couple_key = ? AND status != 'active'", key).Order("achieved_at desc").Find(&rows)
	return c.JSON(http.StatusOK, rows)
}

type mainGoalCreateReq struct {
	Name           string  `json:"name"`
	TargetAmount   float64 `json:"targetAmount"`
	TargetDate     *string `json:"targetDate"`
	RolloverAmount float64 `json:"rolloverAmount"` // opsional: dibawa dari goal lama yang achieved
}

// Create - POST /main-goal : membuat/mengganti goal utama.
// - Kalau belum ada pasangan (jomblo): langsung dibuat, tidak perlu approval.
// - Kalau sudah ada pasangan: dibuatkan MainGoalChangeRequest (action=create)
//   yang harus disetujui pasangan dulu, karena ini keuangan bersama.
func (h *MainGoalHandler) Create(c echo.Context) error {
	me, err := h.currentUser(c)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "user not found"})
	}
	var req mainGoalCreateReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	if req.Name == "" || req.TargetAmount <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "nama dan target dana wajib diisi"})
	}

	key := coupleKey(me.ID, me.PartnerID)

	if me.PartnerID == nil || *me.PartnerID == "" {
		goal := models.MainGoal{
			ID:            genID("mg"),
			CoupleKey:     key,
			OwnerUserID:   me.ID,
			Name:          req.Name,
			TargetAmount:  req.TargetAmount,
			TargetDate:    req.TargetDate,
			CurrentAmount: req.RolloverAmount,
			Status:        "active",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		if err := h.DB.Create(&goal).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create goal"})
		}
		return c.JSON(http.StatusCreated, goal)
	}

	reqRow := models.MainGoalChangeRequest{
		ID:             genID("mgr"),
		CoupleKey:      key,
		FromUserID:     me.ID,
		ToUserID:       *me.PartnerID,
		Action:         "create",
		Name:           req.Name,
		TargetAmount:   req.TargetAmount,
		TargetDate:     req.TargetDate,
		RolloverAmount: req.RolloverAmount,
		Status:         "pending",
		CreatedAt:      time.Now(),
	}
	if err := h.DB.Create(&reqRow).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create request"})
	}
	return c.JSON(http.StatusCreated, reqRow)
}

// Edit - PATCH /main-goal/:id : mengubah nama/target dana/target tanggal
// goal yang sedang aktif. Sama seperti Create, kalau sudah ada pasangan
// harus lewat approval.
func (h *MainGoalHandler) Edit(c echo.Context) error {
	me, err := h.currentUser(c)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "user not found"})
	}
	id := c.Param("id")
	key := coupleKey(me.ID, me.PartnerID)

	var existing models.MainGoal
	if err := h.DB.First(&existing, "id = ? AND couple_key = ?", id, key).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "goal not found"})
	}

	var req mainGoalCreateReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	if req.Name == "" {
		req.Name = existing.Name
	}
	if req.TargetAmount <= 0 {
		req.TargetAmount = existing.TargetAmount
	}

	if me.PartnerID == nil || *me.PartnerID == "" {
		h.DB.Model(&existing).Updates(map[string]interface{}{
			"name": req.Name, "target_amount": req.TargetAmount,
			"target_date": req.TargetDate, "updated_at": time.Now(),
		})
		h.DB.First(&existing, "id = ?", id)
		return c.JSON(http.StatusOK, existing)
	}

	reqRow := models.MainGoalChangeRequest{
		ID:            genID("mgr"),
		CoupleKey:     key,
		FromUserID:    me.ID,
		ToUserID:      *me.PartnerID,
		Action:        "edit",
		OldMainGoalID: &existing.ID,
		Name:          req.Name,
		TargetAmount:  req.TargetAmount,
		TargetDate:    req.TargetDate,
		Status:        "pending",
		CreatedAt:     time.Now(),
	}
	if err := h.DB.Create(&reqRow).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create request"})
	}
	return c.JSON(http.StatusCreated, reqRow)
}

// ListRequests - GET /main-goal-requests : request yang masih pending
// (dari kedua sisi), supaya UI bisa munculkan "menunggu persetujuan".
func (h *MainGoalHandler) ListRequests(c echo.Context) error {
	userID := c.Get("userId").(string)
	var rows []models.MainGoalChangeRequest
	h.DB.Where("(from_user_id = ? OR to_user_id = ?) AND status = 'pending'", userID, userID).
		Order("created_at desc").Find(&rows)
	return c.JSON(http.StatusOK, rows)
}

func (h *MainGoalHandler) Approve(c echo.Context) error { return h.resolveRequest(c, "approved") }
func (h *MainGoalHandler) Reject(c echo.Context) error  { return h.resolveRequest(c, "rejected") }

func (h *MainGoalHandler) resolveRequest(c echo.Context, status string) error {
	userID := c.Get("userId").(string)
	id := c.Param("id")

	var reqRow models.MainGoalChangeRequest
	if err := h.DB.First(&reqRow, "id = ? AND to_user_id = ? AND status = 'pending'", id, userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "request not found or already resolved"})
	}

	now := time.Now()
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&reqRow).Updates(map[string]interface{}{
			"status": status, "resolved_at": now,
		}).Error; err != nil {
			return err
		}
		if status != "approved" {
			return nil
		}
		switch reqRow.Action {
		case "create":
			goal := models.MainGoal{
				ID:            genID("mg"),
				CoupleKey:     reqRow.CoupleKey,
				OwnerUserID:   reqRow.FromUserID,
				Name:          reqRow.Name,
				TargetAmount:  reqRow.TargetAmount,
				TargetDate:    reqRow.TargetDate,
				CurrentAmount: reqRow.RolloverAmount,
				Status:        "active",
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			return tx.Create(&goal).Error
		case "edit":
			if reqRow.OldMainGoalID == nil {
				return nil
			}
			return tx.Model(&models.MainGoal{}).Where("id = ?", *reqRow.OldMainGoalID).Updates(map[string]interface{}{
				"name": reqRow.Name, "target_amount": reqRow.TargetAmount,
				"target_date": reqRow.TargetDate, "updated_at": now,
			}).Error
		}
		return nil
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to resolve request"})
	}
	return c.JSON(http.StatusOK, echo.Map{"status": status})
}

// Contribute menambah CurrentAmount goal aktif pasangan sejumlah `amount`,
// dipanggil dari transaction_handler.go / cashflow_handler.go saat user
// pilih "Masukkan ke goals anda?". Kalau CurrentAmount >= TargetAmount,
// goal langsung ditandai "achieved" supaya Flutter bisa munculkan layar
// perayaan.
func contributeToMainGoal(tx *gorm.DB, userID string, amount float64, sourceType, sourceID string) (*models.MainGoal, error) {
	var me models.User
	if err := tx.First(&me, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	key := coupleKey(me.ID, me.PartnerID)

	var goal models.MainGoal
	if err := tx.Where("couple_key = ? AND status = 'active'", key).
		Order("created_at desc").First(&goal).Error; err != nil {
		return nil, nil // tidak ada goal aktif, diamkan saja (bukan error)
	}

	newAmount := goal.CurrentAmount + amount
	updates := map[string]interface{}{"current_amount": newAmount, "updated_at": time.Now()}
	achieved := newAmount >= goal.TargetAmount && goal.TargetAmount > 0
	if achieved {
		now := time.Now()
		updates["status"] = "achieved"
		updates["achieved_at"] = now
	}
	if err := tx.Model(&goal).Updates(updates).Error; err != nil {
		return nil, err
	}

	contrib := models.MainGoalContribution{
		ID:         genID("mgc"),
		MainGoalID: goal.ID,
		UserID:     userID,
		SourceType: sourceType,
		SourceID:   sourceID,
		Amount:     amount,
		CreatedAt:  time.Now(),
	}
	if err := tx.Create(&contrib).Error; err != nil {
		return nil, err
	}

	goal.CurrentAmount = newAmount
	if achieved {
		goal.Status = "achieved"
	}
	return &goal, nil
}
