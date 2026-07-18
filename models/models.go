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
	ID                   string    `json:"id" gorm:"primaryKey;type:varchar(40)"`
	UserID               string    `json:"userId" gorm:"type:varchar(40);index"`
	WalletID             string    `json:"walletId" gorm:"type:varchar(40);index"`
	Type                 string    `json:"type" gorm:"type:varchar(10)"` // deposit | withdraw
	Amount               float64   `json:"amount"`
	Category             string    `json:"category" gorm:"type:varchar(80)"`
	Note                 *string   `json:"note" gorm:"type:varchar(255)"`
	Date                 string    `json:"date" gorm:"type:varchar(10)"`
	IsWeddingContribution bool     `json:"isWeddingContribution" gorm:"default:false"`
	CreatedAt            time.Time `json:"createdAt"`
}

// ============================================================
// CashFlow (income/expense not tied to a wallet)
// ============================================================
type CashFlow struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(40)"`
	UserID    string    `json:"userId" gorm:"type:varchar(40);index"`
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
