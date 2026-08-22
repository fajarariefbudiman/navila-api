package handlers

import (
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"goals-api/models"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

var errNoActiveGoal = errors.New("no active main goal")
var errWithdrawTooMuch = errors.New("withdraw amount exceeds current goal amount")

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
// pilih "Masukkan ke goals anda?", ATAU dari MainGoalHandler.ContributeFromWallet
// saat user eksplisit mengisi goal dari saldo yang sudah ada di dompet.
//
// ✅ PENTING (akuntansi): kontribusi TIDAK langsung memotong saldo wallet
// saat ini terjadi -- supaya uangnya tidak dihitung dobel selama masa
// menabung (progress goal cuma "menandai" berapa dari saldo yang sudah
// dikomit). Saldo wallet BARU benar-benar dipotong sekali saja, tepat
// saat goal mencapai 100% (lihat settleMainGoalWallets), merepresentasikan
// "uangnya resmi sudah dipakai/dibuat jadi goal itu".
func contributeToMainGoal(tx *gorm.DB, userID string, amount float64, sourceType, sourceID string, walletID *string) (*models.MainGoal, error) {
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
		WalletID:   walletID,
		Amount:     amount,
		CreatedAt:  time.Now(),
	}
	if err := tx.Create(&contrib).Error; err != nil {
		return nil, err
	}

	goal.CurrentAmount = newAmount
	if achieved {
		goal.Status = "achieved"
		// ✅ Goal baru saja mencapai target -- sekarang saatnya benar-benar
		// memotong saldo tiap dompet yang berkontribusi, sesuai total yang
		// pernah "dijanjikan" dari dompet itu. Sebelum titik ini saldo
		// wallet TIDAK pernah dipotong oleh kontribusi goal manapun.
		if err := settleMainGoalWallets(tx, goal.ID); err != nil {
			return nil, err
		}
	}
	return &goal, nil
}

// settleMainGoalWallets dipanggil TEPAT SEKALI saat sebuah MainGoal baru
// saja mencapai status "achieved". Menjumlahkan semua kontribusi goal ini
// per dompet asal (WalletID), memotong saldo tiap dompet itu sebesar
// jumlahnya, dan mencatat satu baris Transaction (type=withdraw) per
// dompet supaya riwayat transaksi tetap akurat & bisa ditelusuri --
// merepresentasikan "uang ini resmi sudah dipakai untuk goal yang selesai".
func settleMainGoalWallets(tx *gorm.DB, mainGoalID string) error {
	var contribs []models.MainGoalContribution
	if err := tx.Where("main_goal_id = ? AND wallet_id IS NOT NULL", mainGoalID).Find(&contribs).Error; err != nil {
		return err
	}

	var goal models.MainGoal
	if err := tx.First(&goal, "id = ?", mainGoalID).Error; err != nil {
		return err
	}

	perWallet := map[string]float64{}
	perWalletUser := map[string]string{}
	for _, c := range contribs {
		if c.WalletID == nil || *c.WalletID == "" {
			continue
		}
		perWallet[*c.WalletID] += c.Amount
		perWalletUser[*c.WalletID] = c.UserID
	}

	for walletID, total := range perWallet {
		if total <= 0 {
			continue
		}
		if err := tx.Model(&models.Wallet{}).Where("id = ?", walletID).
			Update("balance", gorm.Expr("balance - ?", total)).Error; err != nil {
			return err
		}
		note := "Dana untuk goal \"" + goal.Name + "\" tercapai"
		settleTx := models.Transaction{
			ID:        genID("t"),
			UserID:    perWalletUser[walletID],
			WalletID:  walletID,
			Type:      "withdraw",
			Amount:    total,
			Category:  "Goal Tercapai",
			Note:      &note,
			Date:      time.Now().Format("2006-01-02"),
			CreatedAt: time.Now(),
		}
		if err := tx.Create(&settleTx).Error; err != nil {
			return err
		}
	}
	return nil
}

// ContributeFromWallet - POST /main-goal/contribute
// ✅ FITUR: input eksplisit "ambil dari dompet mana, berapa" untuk isi
// Goal Utama dari SALDO YANG SUDAH ADA di dompet (bukan cuma nebeng di
// form deposit baru). Sesuai desain akuntansi di contributeToMainGoal:
// saldo dompet TIDAK langsung dipotong di sini -- baru dipotong nanti
// sekali saat goal ini mencapai 100% (lihat settleMainGoalWallets).
type contributeFromWalletReq struct {
	WalletID string  `json:"walletId"`
	Amount   float64 `json:"amount"`
}

func (h *MainGoalHandler) ContributeFromWallet(c echo.Context) error {
	userID := c.Get("userId").(string)
	var req contributeFromWalletReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	if req.WalletID == "" || req.Amount <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "walletId dan amount wajib diisi"})
	}

	// Pastikan dompet ini benar milik user yang login.
	var wallet models.Wallet
	if err := h.DB.First(&wallet, "id = ? AND user_id = ?", req.WalletID, userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "dompet tidak ditemukan"})
	}

	var updatedGoal *models.MainGoal
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		g, err := contributeToMainGoal(tx, userID, req.Amount, "wallet_direct", req.WalletID, &req.WalletID)
		if err != nil {
			return err
		}
		if g == nil {
			return errNoActiveGoal
		}
		updatedGoal = g
		return nil
	})
	if err == errNoActiveGoal {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "belum ada goal utama yang aktif"})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to contribute"})
	}
	return c.JSON(http.StatusOK, updatedGoal)
}

// Withdraw - POST /main-goal/withdraw
// ✅ FITUR: kurangi dana Goal Utama -- misal goal "Umrah" ternyata mau
// dialihkan prioritasnya ke "Mahar" dulu. Karena kontribusi tidak
// pernah benar-benar memotong saldo wallet sampai goal 100% (lihat
// contributeToMainGoal), mengurangi dana di sini juga TIDAK
// mengembalikan apa pun ke wallet -- uangnya memang tidak pernah
// kemana-mana, cuma dilepas statusnya dari "sudah dikomit ke goal ini"
// jadi bebas lagi (bisa dikomit ulang ke goal lain, atau dibiarkan).
//
// Kalau `walletId` diisi: pengurangan diambil murni dari kontribusi
// yang berasal dari dompet itu saja. Kalau tidak diisi: pengurangan
// dibagi proporsional ke semua dompet yang pernah berkontribusi, supaya
// jejak per-dompetnya tetap akurat kalau nanti goal ini toh mencapai
// 100% (settleMainGoalWallets butuh angka per-dompet yang benar).
type withdrawMainGoalReq struct {
	Amount   float64 `json:"amount"`
	WalletID *string `json:"walletId"`
}

func (h *MainGoalHandler) Withdraw(c echo.Context) error {
	userID := c.Get("userId").(string)
	var req withdrawMainGoalReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	if req.Amount <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "jumlah wajib lebih dari 0"})
	}

	var me models.User
	if err := h.DB.First(&me, "id = ?", userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "user not found"})
	}
	key := coupleKey(me.ID, me.PartnerID)

	var updatedGoal models.MainGoal
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var goal models.MainGoal
		if err := tx.Where("couple_key = ? AND status = 'active'", key).
			Order("created_at desc").First(&goal).Error; err != nil {
			return errNoActiveGoal
		}
		if req.Amount > goal.CurrentAmount {
			return errWithdrawTooMuch
		}

		now := time.Now()
		if err := tx.Model(&goal).Updates(map[string]interface{}{
			"current_amount": goal.CurrentAmount - req.Amount,
			"updated_at":      now,
		}).Error; err != nil {
			return err
		}

		if req.WalletID != nil && *req.WalletID != "" {
			// Kurangi murni dari kontribusi dompet yang diminta.
			contrib := models.MainGoalContribution{
				ID: genID("mgc"), MainGoalID: goal.ID, UserID: userID,
				SourceType: "withdrawal", SourceID: "", WalletID: req.WalletID,
				Amount: -req.Amount, CreatedAt: now,
			}
			if err := tx.Create(&contrib).Error; err != nil {
				return err
			}
		} else {
			// Bagi proporsional ke semua dompet yang pernah berkontribusi.
			var contribs []models.MainGoalContribution
			if err := tx.Where("main_goal_id = ? AND wallet_id IS NOT NULL", goal.ID).Find(&contribs).Error; err != nil {
				return err
			}
			perWallet := map[string]float64{}
			for _, c := range contribs {
				if c.WalletID != nil && *c.WalletID != "" {
					perWallet[*c.WalletID] += c.Amount
				}
			}
			var totalPositive float64
			for _, v := range perWallet {
				if v > 0 {
					totalPositive += v
				}
			}
			if totalPositive > 0 {
				remaining := req.Amount
				i := 0
				keys := make([]string, 0, len(perWallet))
				for wid, v := range perWallet {
					if v > 0 {
						keys = append(keys, wid)
					}
				}
				for _, wid := range keys {
					i++
					var share float64
					if i == len(keys) {
						share = remaining // sisa pembulatan masuk ke yang terakhir
					} else {
						share = req.Amount * (perWallet[wid] / totalPositive)
						remaining -= share
					}
					if share <= 0 {
						continue
					}
					contrib := models.MainGoalContribution{
						ID: genID("mgc"), MainGoalID: goal.ID, UserID: userID,
						SourceType: "withdrawal", SourceID: "", WalletID: &wid,
						Amount: -share, CreatedAt: now,
					}
					if err := tx.Create(&contrib).Error; err != nil {
						return err
					}
				}
			}
		}

		return tx.First(&updatedGoal, "id = ?", goal.ID).Error
	})

	switch err {
	case nil:
		return c.JSON(http.StatusOK, updatedGoal)
	case errNoActiveGoal:
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "belum ada goal utama yang aktif"})
	case errWithdrawTooMuch:
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "jumlah melebihi dana yang terkumpul di goal ini"})
	default:
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to withdraw"})
	}
}