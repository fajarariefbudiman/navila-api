package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// ============================================================
// JSONSubtasks: custom type so GORM can store subtasks as a JSON column
// ============================================================
type Subtask struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

type Subtasks []Subtask

func (s Subtasks) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *Subtasks) Scan(value interface{}) error {
	if value == nil {
		*s = Subtasks{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		if str, ok2 := value.(string); ok2 {
			bytes = []byte(str)
		} else {
			return nil
		}
	}
	return json.Unmarshal(bytes, s)
}

// ============================================================
// User
// ============================================================
type User struct {
	ID                string    `json:"id" gorm:"primaryKey;type:varchar(40)"`
	Name              string    `json:"name" gorm:"type:varchar(120)"`
	Email             string    `json:"email" gorm:"type:varchar(160);uniqueIndex"`
	Password          string    `json:"-" gorm:"type:varchar(255)"`
	Gender            string    `json:"gender" gorm:"type:varchar(10)"`
	Mode              string    `json:"mode" gorm:"type:varchar(10);default:'jomblo'"` // jomblo | myKisah
	PartnerID         *string   `json:"partnerId" gorm:"type:varchar(40)"`
	PartnerCode       string    `json:"partnerCode" gorm:"type:varchar(20);uniqueIndex"`
	WeddingDate       *string   `json:"weddingDate" gorm:"type:varchar(10)"`
	TargetWeddingFund float64   `json:"targetWeddingFund"`
	Avatar            *string   `json:"avatar" gorm:"type:varchar(255)"`
	LastBackupAt      *time.Time `json:"lastBackupAt"`
	CreatedAt         time.Time `json:"createdAt"`
}

// ============================================================
// Wallet
// ============================================================
type Wallet struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(40)"`
	UserID       string    `json:"userId" gorm:"type:varchar(40);index"`
	Name         string    `json:"name" gorm:"type:varchar(120)"`
	Type         string    `json:"type" gorm:"type:varchar(30)"`
	Balance      float64   `json:"balance"`
	TargetAmount *float64  `json:"targetAmount"`
	Color        string    `json:"color" gorm:"type:varchar(20)"`
	CreatedAt    time.Time `json:"createdAt"`
}

// ============================================================
// Budget
// ============================================================
type Budget struct {
	ID              string    `json:"id" gorm:"primaryKey;type:varchar(40)"`
	UserID          string    `json:"userId" gorm:"type:varchar(40);index"`
	Category        string    `json:"category" gorm:"type:varchar(80)"`
	AllocatedAmount float64   `json:"allocatedAmount"`
	Period          string    `json:"period" gorm:"type:varchar(7)"` // YYYY-MM
	CreatedAt       time.Time `json:"createdAt"`
}

// ============================================================
// Transaction (per wallet: deposit/withdraw)
// ============================================================
type Transaction struct {
	ID                    string    `json:"id" gorm:"primaryKey;type:varchar(40)"`
	UserID                string    `json:"userId" gorm:"type:varchar(40);index"`
	WalletID              string    `json:"walletId" gorm:"type:varchar(40);index"`
	Type                  string    `json:"type" gorm:"type:varchar(10)"` // deposit | withdraw
	Amount                float64   `json:"amount"`
	Category              string    `json:"category" gorm:"type:varchar(80)"`
	Note                  *string   `json:"note" gorm:"type:varchar(255)"`
	Date                  string    `json:"date" gorm:"type:varchar(10)"`
	CreatedAt             time.Time `json:"createdAt"`
	IsWeddingContribution bool      `json:"isWeddingContribution"`
}

// ============================================================
// CashFlow (income/expense; opsional tertaut ke sebuah Wallet)
// ============================================================
type CashFlow struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(40)"`
	UserID    string    `json:"userId" gorm:"type:varchar(40);index"`
	WalletID  *string   `json:"walletId" gorm:"type:varchar(40);index"`
	Type      string    `json:"type" gorm:"type:varchar(10)"` // income | expense
	Category  string    `json:"category" gorm:"type:varchar(80)"`
	Amount    float64   `json:"amount"`
	Note      *string   `json:"note" gorm:"type:varchar(255)"`
	Date      string    `json:"date" gorm:"type:varchar(10)"`
	CreatedAt time.Time `json:"createdAt"`
}

// ============================================================
// WeddingCategory
// ============================================================
type WeddingCategory struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(40)"`
	CoupleID  string    `json:"coupleId" gorm:"type:varchar(40);index"`
	Name      string    `json:"name" gorm:"type:varchar(120)"`
	Budget    float64   `json:"budget"`
	Paid      float64   `json:"paid"`
	Vendor    *string   `json:"vendor" gorm:"type:varchar(160)"`
	Status    string    `json:"status" gorm:"type:varchar(20)"`
	Notes     *string   `json:"notes" gorm:"type:varchar(255)"`
	Icon      string    `json:"icon" gorm:"type:varchar(40)"` // Font Awesome icon key, e.g. "building"
	// WalletID: tabungan/kas yang jadi sumber dana pos ini. Kalau diisi,
	// setiap kali `Paid` bertambah (misal ditandai "lunas"/"done"), saldo
	// wallet ini otomatis berkurang sejumlah kenaikan Paid -- lihat
	// wedding_handler.go Update().
	WalletID  *string   `json:"walletId" gorm:"type:varchar(40)"`
	CreatedAt time.Time `json:"createdAt"`
}

// ============================================================
// Plan (Rencana Kita)
// ============================================================
type Plan struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(40)"`
	UserID      string    `json:"userId" gorm:"type:varchar(40);index"`
	Type        string    `json:"type" gorm:"type:varchar(20)"` // mine | partner | together
	Title       string    `json:"title" gorm:"type:varchar(160)"`
	Description *string   `json:"description" gorm:"type:varchar(500)"`
	Category    string    `json:"category" gorm:"type:varchar(40)"`
	TargetDate  *string   `json:"targetDate" gorm:"type:varchar(10)"`
	Budget      *float64  `json:"budget"`
	Status      string    `json:"status" gorm:"type:varchar(20)"`
	// ✅ Progress (0-100) -- dihitung otomatis dari Subtasks (persis
	// seperti Goal), supaya konsisten: cek subtask, persen ikut naik.
	Progress    int       `json:"progress"`
	Subtasks    Subtasks  `json:"subtasks" gorm:"type:json"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ============================================================
// Goal
// ============================================================
type Goal struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(40)"`
	UserID       string    `json:"userId" gorm:"type:varchar(40);index"`
	Title        string    `json:"title" gorm:"type:varchar(160)"`
	Description  *string   `json:"description" gorm:"type:varchar(500)"`
	Category     string    `json:"category" gorm:"type:varchar(40)"`
	Priority     string    `json:"priority" gorm:"type:varchar(10)"`
	Status       string    `json:"status" gorm:"type:varchar(20)"`
	Progress     int       `json:"progress"`
	Deadline     *string   `json:"deadline" gorm:"type:varchar(10)"`
	TargetAmount *float64  `json:"targetAmount"`
	Notes        *string   `json:"notes" gorm:"type:varchar(255)"`
	Subtasks     Subtasks  `json:"subtasks" gorm:"type:json"`
	CreatedAt    time.Time `json:"createdAt"`
}

// ============================================================
// BackupLog: keeps track of every export/import so the 28-day
// reminder can be computed from real history if needed.
// ============================================================
type BackupLog struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    string    `json:"userId" gorm:"type:varchar(40);index"`
	Action    string    `json:"action" gorm:"type:varchar(10)"` // export | import
	CreatedAt time.Time `json:"createdAt"`
}

// ============================================================
// FundChangeRequest: perubahan "Target Dana Masa Depan" adalah nilai
// bersama pasangan, jadi tidak langsung diterapkan begitu salah satu
// pihak mengubahnya — harus disetujui pasangan dulu.
// ============================================================
type FundChangeRequest struct {
	ID         string     `json:"id" gorm:"primaryKey;type:varchar(40)"`
	FromUserID string     `json:"fromUserId" gorm:"type:varchar(40);index"`
	ToUserID   string     `json:"toUserId" gorm:"type:varchar(40);index"`
	OldAmount  float64    `json:"oldAmount"`
	NewAmount  float64    `json:"newAmount"`
	Status     string     `json:"status" gorm:"type:varchar(10)"` // pending|approved|rejected
	CreatedAt  time.Time  `json:"createdAt"`
	ResolvedAt *time.Time `json:"resolvedAt"`
}

// ============================================================
// FITUR 1: MainGoal -- "Goals Utama Pasangan". Menggantikan konsep lama
// TargetWeddingFund (dana nikah tetap). Sekarang goal utama bisa apa saja
// (nama bebas), target dana, dan target tanggal (opsional). Disimpan satu
// baris per pasangan, diidentifikasi lewat CoupleKey (gabungan kedua
// userId, diurutkan supaya konsisten dari sisi manapun query-nya). Untuk
// mode jomblo, CoupleKey cuma berisi userId itu sendiri.
// ============================================================
type MainGoal struct {
	ID           string     `json:"id" gorm:"primaryKey;type:varchar(40)"`
	CoupleKey    string     `json:"coupleKey" gorm:"type:varchar(90);index"`
	OwnerUserID  string     `json:"ownerUserId" gorm:"type:varchar(40)"` // pembuat goal ini
	Name         string     `json:"name" gorm:"type:varchar(160)"`
	TargetAmount float64    `json:"targetAmount"`
	TargetDate   *string    `json:"targetDate" gorm:"type:varchar(10)"` // opsional, YYYY-MM-DD
	CurrentAmount float64   `json:"currentAmount"`
	Status       string     `json:"status" gorm:"type:varchar(10)"` // active | achieved | archived
	AchievedAt   *time.Time `json:"achievedAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

// MainGoalContribution: jejak riwayat setiap "nambah ke goals" dari wallet
// atau cashflow, supaya bisa ditampilkan siapa nyumbang berapa & kapan.
type MainGoalContribution struct {
	ID         string    `json:"id" gorm:"primaryKey;type:varchar(40)"`
	MainGoalID string    `json:"mainGoalId" gorm:"type:varchar(40);index"`
	UserID     string    `json:"userId" gorm:"type:varchar(40);index"`
	SourceType string    `json:"sourceType" gorm:"type:varchar(20)"` // wallet_deposit | cashflow_income | wallet_direct
	SourceID   string    `json:"sourceId" gorm:"type:varchar(40)"`
	// WalletID: dompet asal dana kontribusi ini. Dicatat supaya saat goal
	// tercapai, saldo tiap dompet yang berkontribusi bisa dipotong sesuai
	// jumlah yang sudah "dijanjikan" ke goal -- lihat settleMainGoalWallets
	// di main_goal_handler.go. Nullable karena data lama sebelum kolom ini
	// ada tidak punya wallet asal yang tercatat.
	WalletID   *string   `json:"walletId" gorm:"type:varchar(40)"`
	Amount     float64   `json:"amount"`
	CreatedAt  time.Time `json:"createdAt"`
}

// MainGoalChangeRequest: sama seperti FundChangeRequest tapi generik untuk
// MainGoal -- dipakai kalau pasangan sudah punya goal aktif dan salah
// satu pihak mau ganti nama/target dana/target tanggal, ATAU membuat goal
// baru setelah goal sebelumnya tercapai (dengan opsi rollover saldo).
// Kalau user belum punya pasangan (mode jomblo), perubahan berlaku
// langsung tanpa approval -- lihat main_goal_handler.go.
type MainGoalChangeRequest struct {
	ID              string     `json:"id" gorm:"primaryKey;type:varchar(40)"`
	CoupleKey       string     `json:"coupleKey" gorm:"type:varchar(90);index"`
	FromUserID      string     `json:"fromUserId" gorm:"type:varchar(40);index"`
	ToUserID        string     `json:"toUserId" gorm:"type:varchar(40);index"`
	Action          string     `json:"action" gorm:"type:varchar(10)"` // create | edit
	OldMainGoalID   *string    `json:"oldMainGoalId" gorm:"type:varchar(40)"`
	Name            string     `json:"name" gorm:"type:varchar(160)"`
	TargetAmount    float64    `json:"targetAmount"`
	TargetDate      *string    `json:"targetDate" gorm:"type:varchar(10)"`
	RolloverAmount  float64    `json:"rolloverAmount"` // dibawa dari goal lama yang sudah achieved (kalau action=create)
	Status          string     `json:"status" gorm:"type:varchar(10)"` // pending|approved|rejected
	CreatedAt       time.Time  `json:"createdAt"`
	ResolvedAt      *time.Time `json:"resolvedAt"`
}

// ============================================================
// FITUR 2: Wedding template & linkage ke wallet
// ============================================================

// WeddingTemplateSelection: template yang sedang dipakai pasangan. Kalau
// mau ganti template padahal sudah pernah pilih, wajib lewat
// WeddingTemplateChangeRequest (approval pasangan) -- template hanya
// "wadah" awal, isinya (WeddingCategory) tetap bebas diedit siapa saja.
type WeddingTemplateSelection struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(40)"`
	CoupleKey    string    `json:"coupleKey" gorm:"type:varchar(90);index"`
	TemplateKey  string    `json:"templateKey" gorm:"type:varchar(60)"`
	SelectedBy   string    `json:"selectedBy" gorm:"type:varchar(40)"`
	CreatedAt    time.Time `json:"createdAt"`
}

type WeddingTemplateChangeRequest struct {
	ID             string     `json:"id" gorm:"primaryKey;type:varchar(40)"`
	CoupleKey      string     `json:"coupleKey" gorm:"type:varchar(90);index"`
	FromUserID     string     `json:"fromUserId" gorm:"type:varchar(40);index"`
	ToUserID       string     `json:"toUserId" gorm:"type:varchar(40);index"`
	OldTemplateKey string     `json:"oldTemplateKey" gorm:"type:varchar(60)"`
	NewTemplateKey string     `json:"newTemplateKey" gorm:"type:varchar(60)"`
	Status         string     `json:"status" gorm:"type:varchar(10)"` // pending|approved|rejected
	CreatedAt      time.Time  `json:"createdAt"`
	ResolvedAt     *time.Time `json:"resolvedAt"`
}

// ============================================================
// FITUR 3: Habit schedule
// ============================================================

// HabitProfile: input mentah dari user (gaji, agama, budget kontrakan,
// bensin, makan, status kuliah, jam freelance) yang jadi bahan generate
// jadwal otomatis.
type HabitProfile struct {
	ID              string    `json:"id" gorm:"primaryKey;type:varchar(40)"`
	UserID          string    `json:"userId" gorm:"type:varchar(40);uniqueIndex"`
	Salary          float64   `json:"salary"`
	Religion        string    `json:"religion" gorm:"type:varchar(30)"`
	RentBudget      float64   `json:"rentBudget"`
	FuelBudget      float64   `json:"fuelBudget"`
	FoodBudget      float64   `json:"foodBudget"`
	IsCollege       bool      `json:"isCollege"`
	HasFreelance    bool      `json:"hasFreelance"`
	FreelanceStart  *string   `json:"freelanceStart" gorm:"type:varchar(5)"` // "HH:MM"
	FreelanceEnd    *string   `json:"freelanceEnd" gorm:"type:varchar(5)"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// HabitScheduleEntry: satu blok jadwal di satu hari (Senin=1 ... Minggu=7).
type HabitScheduleEntry struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(40)"`
	UserID    string    `json:"userId" gorm:"type:varchar(40);index"`
	DayOfWeek int       `json:"dayOfWeek"` // 1=Senin ... 7=Minggu
	StartTime string    `json:"startTime" gorm:"type:varchar(5)"`
	EndTime   string    `json:"endTime" gorm:"type:varchar(5)"`
	Title     string    `json:"title" gorm:"type:varchar(120)"`
	Category  string    `json:"category" gorm:"type:varchar(30)"` // ibadah|kuliah|kerja|freelance|istirahat|keuangan|lainnya
	Source    string    `json:"source" gorm:"type:varchar(10)"`   // ai|static|manual
	CreatedAt time.Time `json:"createdAt"`
}