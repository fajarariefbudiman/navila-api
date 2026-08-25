package handlers

import (
	"net/http"
	"time"

	"goals-api/models"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type WeddingTemplateHandler struct{ DB *gorm.DB }

// WeddingTemplateItem: satu baris item di template referensi (seperti
// yang biasa ada di file Excel "wedding planner checklist" yang lengkap
// dijual di marketplace) -- nama item, kategori, icon FontAwesome, dan
// persentase estimasi dari total budget supaya user tinggal isi total
// anggaran lalu tiap pos otomatis punya perkiraan.
type WeddingTemplateItem struct {
	Name    string  `json:"name"`
	Icon    string  `json:"icon"`
	Percent float64 `json:"percent"` // perkiraan porsi dari total budget
}

type WeddingTemplateDef struct {
	Key         string                 `json:"key"`
	Label       string                 `json:"label"`
	Description string                 `json:"description"`
	Items       []WeddingTemplateItem  `json:"items"`
}

// weddingTemplates: referensi statis, mencakup beberapa adat/agama besar
// di Indonesia. Ini hanya WADAH AWAL -- setelah diterapkan, semua item
// (WeddingCategory) tetap 100% bisa diedit/dihapus/ditambah bebas oleh
// user, persis seperti kalau ditulis manual.
var weddingTemplates = []WeddingTemplateDef{
	{
		Key: "islam_umum", Label: "Islam (Akad + Resepsi Umum)",
		Description: "Checklist standar pernikahan Islam: akad nikah, resepsi, katering, dokumentasi.",
		Items: []WeddingTemplateItem{
			{Name: "Mahar & Seserahan", Icon: "gift", Percent: 3},
			{Name: "Petugas KUA / Penghulu", Icon: "book", Percent: 1},
			{Name: "Gedung / Venue", Icon: "building", Percent: 25},
			{Name: "Katering", Icon: "utensils", Percent: 25},
			{Name: "Dekorasi & Bunga", Icon: "spa", Percent: 12},
			{Name: "Busana Pengantin (Akad+Resepsi)", Icon: "user-tie", Percent: 10},
			{Name: "Rias & Wedding Organizer", Icon: "palette", Percent: 8},
			{Name: "Dokumentasi (Foto+Video)", Icon: "camera", Percent: 8},
			{Name: "Souvenir & Undangan", Icon: "envelope", Percent: 4},
			{Name: "Hiburan (MC/Musik)", Icon: "music", Percent: 4},
		},
	},
	{
		Key: "islam_jawa", Label: "Islam - Adat Jawa",
		Description: "Adat Jawa lengkap dengan siraman, midodareni, dan panggih.",
		Items: []WeddingTemplateItem{
			{Name: "Mahar & Seserahan", Icon: "gift", Percent: 3},
			{Name: "Petugas KUA / Penghulu", Icon: "book", Percent: 1},
			{Name: "Prosesi Siraman", Icon: "tint", Percent: 3},
			{Name: "Prosesi Midodareni", Icon: "moon", Percent: 3},
			{Name: "Prosesi Panggih & Adat", Icon: "handshake", Percent: 6},
			{Name: "Gedung / Venue", Icon: "building", Percent: 22},
			{Name: "Katering", Icon: "utensils", Percent: 22},
			{Name: "Dekorasi Adat Jawa", Icon: "spa", Percent: 10},
			{Name: "Busana Adat (Kebaya/Beskap)", Icon: "user-tie", Percent: 10},
			{Name: "Rias Pengantin Jawa & WO", Icon: "palette", Percent: 8},
			{Name: "Dokumentasi", Icon: "camera", Percent: 7},
			{Name: "Souvenir & Undangan", Icon: "envelope", Percent: 3},
			{Name: "Gamelan / Hiburan", Icon: "music", Percent: 2},
		},
	},
	{
		Key: "islam_sunda", Label: "Islam - Adat Sunda",
		Description: "Adat Sunda dengan prosesi Sungkeman, Ngeuyeuk Seureuh, dan Saweran.",
		Items: []WeddingTemplateItem{
			{Name: "Mahar & Seserahan", Icon: "gift", Percent: 3},
			{Name: "Petugas KUA / Penghulu", Icon: "book", Percent: 1},
			{Name: "Prosesi Ngeuyeuk Seureuh", Icon: "leaf", Percent: 3},
			{Name: "Prosesi Sungkeman & Saweran", Icon: "hand-holding-heart", Percent: 4},
			{Name: "Gedung / Venue", Icon: "building", Percent: 24},
			{Name: "Katering", Icon: "utensils", Percent: 24},
			{Name: "Dekorasi Sunda", Icon: "spa", Percent: 10},
			{Name: "Busana Adat Sunda", Icon: "user-tie", Percent: 10},
			{Name: "Rias & WO", Icon: "palette", Percent: 8},
			{Name: "Dokumentasi", Icon: "camera", Percent: 7},
			{Name: "Souvenir & Undangan", Icon: "envelope", Percent: 4},
			{Name: "Kecapi Suling / Hiburan", Icon: "music", Percent: 2},
		},
	},
	{
		Key: "adat_batak", Label: "Kristen/Katolik - Adat Batak",
		Description: "Adat Batak dengan Martumpol, Marhata Sinamot, dan Mangulosi.",
		Items: []WeddingTemplateItem{
			{Name: "Pemberkatan di Gereja", Icon: "church", Percent: 3},
			{Name: "Prosesi Martumpol", Icon: "book", Percent: 2},
			{Name: "Marhata Sinamot (Adat)", Icon: "handshake", Percent: 6},
			{Name: "Mangulosi & Ulos", Icon: "gift", Percent: 4},
			{Name: "Gedung / Venue", Icon: "building", Percent: 23},
			{Name: "Katering (biasa porsi besar)", Icon: "utensils", Percent: 25},
			{Name: "Dekorasi", Icon: "spa", Percent: 9},
			{Name: "Busana Adat Batak", Icon: "user-tie", Percent: 9},
			{Name: "Rias & WO", Icon: "palette", Percent: 7},
			{Name: "Dokumentasi", Icon: "camera", Percent: 7},
			{Name: "Souvenir & Undangan", Icon: "envelope", Percent: 3},
			{Name: "Gondang / Hiburan", Icon: "music", Percent: 2},
		},
	},
	{
		Key: "hindu_bali", Label: "Hindu - Adat Bali",
		Description: "Upacara Hindu Bali dengan Mekala-kalaan dan Madengen-dengenan.",
		Items: []WeddingTemplateItem{
			{Name: "Upacara Mekala-kalaan", Icon: "book", Percent: 5},
			{Name: "Upacara Madengen-dengenan", Icon: "handshake", Percent: 4},
			{Name: "Banten & Sarana Upacara", Icon: "gift", Percent: 8},
			{Name: "Gedung / Venue", Icon: "building", Percent: 22},
			{Name: "Katering", Icon: "utensils", Percent: 22},
			{Name: "Dekorasi Bali (Penjor dll)", Icon: "spa", Percent: 10},
			{Name: "Busana Adat Bali", Icon: "user-tie", Percent: 10},
			{Name: "Rias & WO", Icon: "palette", Percent: 7},
			{Name: "Dokumentasi", Icon: "camera", Percent: 7},
			{Name: "Souvenir & Undangan", Icon: "envelope", Percent: 3},
			{Name: "Gamelan Bali", Icon: "music", Percent: 2},
		},
	},
	{
		Key: "adat_bugis", Label: "Islam - Adat Bugis/Makassar",
		Description: "Adat Bugis dengan Mappacci dan Uang Panai.",
		Items: []WeddingTemplateItem{
			{Name: "Uang Panai", Icon: "gift", Percent: 15},
			{Name: "Petugas KUA / Penghulu", Icon: "book", Percent: 1},
			{Name: "Prosesi Mappacci", Icon: "leaf", Percent: 3},
			{Name: "Gedung / Venue", Icon: "building", Percent: 20},
			{Name: "Katering", Icon: "utensils", Percent: 22},
			{Name: "Dekorasi", Icon: "spa", Percent: 9},
			{Name: "Busana Adat Bugis", Icon: "user-tie", Percent: 9},
			{Name: "Rias & WO", Icon: "palette", Percent: 7},
			{Name: "Dokumentasi", Icon: "camera", Percent: 7},
			{Name: "Souvenir & Undangan", Icon: "envelope", Percent: 4},
			{Name: "Hiburan", Icon: "music", Percent: 3},
		},
	},
	{
		Key: "adat_betawi", Label: "Islam - Adat Betawi",
		Description: "Adat Betawi dengan Palang Pintu, Sirih Nanas/Dare, dan Roti Buaya.",
		Items: []WeddingTemplateItem{
			{Name: "Mahar & Seserahan", Icon: "gift", Percent: 3},
			{Name: "Petugas KUA / Penghulu", Icon: "book", Percent: 1},
			{Name: "Prosesi Palang Pintu (Pantun & Silat)", Icon: "handshake", Percent: 4},
			{Name: "Sirih Nanas / Sirih Dare", Icon: "leaf", Percent: 2},
			{Name: "Roti Buaya", Icon: "gift", Percent: 2},
			{Name: "Gedung / Venue", Icon: "building", Percent: 23},
			{Name: "Katering (Nasi Kebuli/Betawi)", Icon: "utensils", Percent: 24},
			{Name: "Dekorasi Betawi (Ondel-ondel dll)", Icon: "spa", Percent: 10},
			{Name: "Busana Pengantin Betawi (Dandanan Cine)", Icon: "user-tie", Percent: 10},
			{Name: "Rias & Wedding Organizer", Icon: "palette", Percent: 8},
			{Name: "Dokumentasi (Foto+Video)", Icon: "camera", Percent: 7},
			{Name: "Souvenir & Undangan", Icon: "envelope", Percent: 4},
			{Name: "Gambang Kromong / Hiburan", Icon: "music", Percent: 2},
		},
	},
	{
		Key: "adat_minang", Label: "Islam - Adat Minang",
		Description: "Adat Minangkabau dengan Maminang, Malam Bainai, dan Baralek.",
		Items: []WeddingTemplateItem{
			{Name: "Uang Jemputan / Mahar", Icon: "gift", Percent: 4},
			{Name: "Petugas KUA / Penghulu", Icon: "book", Percent: 1},
			{Name: "Prosesi Maminang & Batimbang Tando", Icon: "handshake", Percent: 4},
			{Name: "Malam Bainai", Icon: "moon", Percent: 3},
			{Name: "Gedung / Venue", Icon: "building", Percent: 22},
			{Name: "Katering (Baralek/Nasi Kapau)", Icon: "utensils", Percent: 24},
			{Name: "Dekorasi Minang (Pelaminan Marawa)", Icon: "spa", Percent: 10},
			{Name: "Busana Adat Minang (Baju Kurung/Bundo Kanduang)", Icon: "user-tie", Percent: 11},
			{Name: "Rias & Wedding Organizer", Icon: "palette", Percent: 8},
			{Name: "Dokumentasi (Foto+Video)", Icon: "camera", Percent: 7},
			{Name: "Souvenir & Undangan", Icon: "envelope", Percent: 4},
			{Name: "Talempong / Hiburan", Icon: "music", Percent: 2},
		},
	},
	{
		Key: "custom", Label: "Kosong / Custom",
		Description: "Mulai dari kosong, tambahkan pos anggaran sendiri sepenuhnya.",
		Items:       []WeddingTemplateItem{},
	},
}

// ListTemplates - GET /wedding/templates
func (h *WeddingTemplateHandler) ListTemplates(c echo.Context) error {
	return c.JSON(http.StatusOK, weddingTemplates)
}

func findTemplate(key string) *WeddingTemplateDef {
	for i := range weddingTemplates {
		if weddingTemplates[i].Key == key {
			return &weddingTemplates[i]
		}
	}
	return nil
}

// GetSelection - GET /wedding/template-selection : template yang sedang
// dipakai pasangan ini (kalau ada).
func (h *WeddingTemplateHandler) GetSelection(c echo.Context) error {
	userID := c.Get("userId").(string)
	var me models.User
	if err := h.DB.First(&me, "id = ?", userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "user not found"})
	}
	key := coupleKey(me.ID, me.PartnerID)
	var sel models.WeddingTemplateSelection
	if err := h.DB.Where("couple_key = ?", key).Order("created_at desc").First(&sel).Error; err != nil {
		return c.JSON(http.StatusOK, nil)
	}
	return c.JSON(http.StatusOK, sel)
}

type applyTemplateReq struct {
	TemplateKey  string   `json:"templateKey"`
	TotalBudget  float64  `json:"totalBudget"` // dipakai buat hitung estimasi tiap pos dari Percent
	WalletID     *string  `json:"walletId"`     // opsional: link default ke wallet ini
}

// ApplyTemplate - POST /wedding/apply-template
// - Kalau pasangan BELUM pernah pilih template: langsung diterapkan (isi
//   WeddingCategory dari template, budget per item = totalBudget * percent).
// - Kalau SUDAH pernah pilih template (mau ganti ke lain): wajib approval
//   pasangan lewat WeddingTemplateChangeRequest, item lama TIDAK dihapus
//   sampai disetujui.
func (h *WeddingTemplateHandler) ApplyTemplate(c echo.Context) error {
	userID := c.Get("userId").(string)
	var me models.User
	if err := h.DB.First(&me, "id = ?", userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "user not found"})
	}
	var req applyTemplateReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	tmpl := findTemplate(req.TemplateKey)
	if tmpl == nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "template tidak ditemukan"})
	}
	key := coupleKey(me.ID, me.PartnerID)

	var existing models.WeddingTemplateSelection
	hasExisting := h.DB.Where("couple_key = ?", key).Order("created_at desc").First(&existing).Error == nil

	if hasExisting && me.PartnerID != nil && *me.PartnerID != "" {
		reqRow := models.WeddingTemplateChangeRequest{
			ID:             genID("wtr"),
			CoupleKey:      key,
			FromUserID:     me.ID,
			ToUserID:       *me.PartnerID,
			OldTemplateKey: existing.TemplateKey,
			NewTemplateKey: req.TemplateKey,
			Status:         "pending",
			CreatedAt:      time.Now(),
		}
		if err := h.DB.Create(&reqRow).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create request"})
		}
		return c.JSON(http.StatusCreated, reqRow)
	}

	var created []models.WeddingCategory
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		sel := models.WeddingTemplateSelection{
			ID: genID("wts"), CoupleKey: key, TemplateKey: req.TemplateKey,
			SelectedBy: me.ID, CreatedAt: time.Now(),
		}
		if err := tx.Create(&sel).Error; err != nil {
			return err
		}
		for _, item := range tmpl.Items {
			cat := models.WeddingCategory{
				ID: genID("wc"), CoupleID: me.ID, Name: item.Name,
				Budget: req.TotalBudget * item.Percent / 100, Paid: 0,
				// BUG FIX: status harus salah satu dari 4 nilai yang dikenal
				// Flutter ('planning'|'booked'|'paid'|'done') -- sebelumnya
				// dipakai "belum" yang tidak ada di dropdown, jadi
				// DropdownButton di detail item error begitu dibuka.
				Status: "planning", Icon: item.Icon, CreatedAt: time.Now(),
				WalletID: req.WalletID,
			}
			if err := tx.Create(&cat).Error; err != nil {
				return err
			}
			created = append(created, cat)
		}
		return nil
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to apply template"})
	}
	return c.JSON(http.StatusCreated, echo.Map{"selection": req.TemplateKey, "categories": created})
}

func (h *WeddingTemplateHandler) ListRequests(c echo.Context) error {
	userID := c.Get("userId").(string)
	var rows []models.WeddingTemplateChangeRequest
	h.DB.Where("(from_user_id = ? OR to_user_id = ?) AND status = 'pending'", userID, userID).
		Order("created_at desc").Find(&rows)
	return c.JSON(http.StatusOK, rows)
}

func (h *WeddingTemplateHandler) Approve(c echo.Context) error { return h.resolve(c, "approved") }
func (h *WeddingTemplateHandler) Reject(c echo.Context) error  { return h.resolve(c, "rejected") }

func (h *WeddingTemplateHandler) resolve(c echo.Context, status string) error {
	userID := c.Get("userId").(string)
	id := c.Param("id")
	var reqRow models.WeddingTemplateChangeRequest
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
		sel := models.WeddingTemplateSelection{
			ID: genID("wts"), CoupleKey: reqRow.CoupleKey, TemplateKey: reqRow.NewTemplateKey,
			SelectedBy: reqRow.FromUserID, CreatedAt: now,
		}
		return tx.Create(&sel).Error
		// NOTE: item WeddingCategory lama sengaja TIDAK dihapus otomatis --
		// pasangan bisa bersih-bersih manual kalau memang mau mulai ulang,
		// supaya data yang sudah terlanjur dibayar (Paid > 0) tidak hilang.
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to resolve request"})
	}
	return c.JSON(http.StatusOK, echo.Map{"status": status})
}