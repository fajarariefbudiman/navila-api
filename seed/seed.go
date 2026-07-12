package seed

import (
	"log"
	"time"

	"goals-api/models"

	"gorm.io/gorm"
)

func strPtr(s string) *string { return &s }
func f64Ptr(f float64) *float64 { return &f }

// Run seeds a small, realistic dummy dataset (couple: Fajar & Raisa) so the
// API has something to serve out of the box when the database is empty.
// It is intentionally idempotent: it only seeds when no users exist yet.
func Run(db *gorm.DB) {
	var count int64
	db.Model(&models.User{}).Count(&count)
	if count > 0 {
		return
	}

	log.Println("Seeding dummy data: Fajar & Raisa...")
	now := time.Now()
	weddingDate := "2026-11-14"

	fajar := models.User{
		ID:                "usr-fajar",
		Name:              "Fajar Nugroho",
		Email:             "fajar@demo.com",
		Password:          hashDemoPassword("demo123"),
		Gender:            "male",
		PartnerID:         strPtr("usr-raisa"),
		PartnerCode:       "FJR001",
		WeddingDate:       &weddingDate,
		TargetWeddingFund: 120000000,
		CreatedAt:         now,
	}
	raisa := models.User{
		ID:                "usr-raisa",
		Name:              "Raisa Amelia",
		Email:             "raisa@demo.com",
		Password:          hashDemoPassword("demo123"),
		Gender:            "female",
		PartnerID:         strPtr("usr-fajar"),
		PartnerCode:       "RSA001",
		WeddingDate:       &weddingDate,
		TargetWeddingFund: 120000000,
		CreatedAt:         now,
	}
	db.Create(&fajar)
	db.Create(&raisa)

	wallets := []models.Wallet{
		{ID: "w-fjr-1", UserID: "usr-fajar", Name: "BCA Tabungan Nikah", Type: "Bank", Balance: 18000000, TargetAmount: f64Ptr(50000000), Color: "#3B82F6", CreatedAt: now},
		{ID: "w-fjr-2", UserID: "usr-fajar", Name: "GoPay", Type: "E-Wallet", Balance: 1200000, Color: "#00AA13", CreatedAt: now},
		{ID: "w-rsa-1", UserID: "usr-raisa", Name: "Mandiri Tabungan", Type: "Bank", Balance: 14500000, TargetAmount: f64Ptr(40000000), Color: "#F59E0B", CreatedAt: now},
		{ID: "w-rsa-2", UserID: "usr-raisa", Name: "Emas Antam", Type: "Emas", Balance: 8000000, TargetAmount: f64Ptr(15000000), Color: "#D97706", CreatedAt: now},
	}
	db.Create(&wallets)

	period := now.Format("2006-01")
	budgets := []models.Budget{
		{ID: "bud-fjr-1", UserID: "usr-fajar", Category: "Makan & Minum", AllocatedAmount: 2500000, Period: period, CreatedAt: now},
		{ID: "bud-fjr-2", UserID: "usr-fajar", Category: "Transport", AllocatedAmount: 1200000, Period: period, CreatedAt: now},
		{ID: "bud-rsa-1", UserID: "usr-raisa", Category: "Kecantikan", AllocatedAmount: 900000, Period: period, CreatedAt: now},
	}
	db.Create(&budgets)

	day := func(offset int) string { return now.AddDate(0, 0, -offset).Format("2006-01-02") }

	transactions := []models.Transaction{
		{ID: "tx-1", UserID: "usr-fajar", WalletID: "w-fjr-1", Type: "deposit", Amount: 4000000, Category: "Tabungan", Note: strPtr("Gajian bulan ini"), Date: day(2), CreatedAt: now},
		{ID: "tx-2", UserID: "usr-fajar", WalletID: "w-fjr-2", Type: "deposit", Amount: 500000, Category: "Top-up", Date: day(6), CreatedAt: now},
		{ID: "tx-3", UserID: "usr-raisa", WalletID: "w-rsa-1", Type: "deposit", Amount: 3500000, Category: "Tabungan", Note: strPtr("Nabung nikah"), Date: day(3), CreatedAt: now},
	}
	db.Create(&transactions)

	cashflows := []models.CashFlow{
		{ID: "cf-1", UserID: "usr-fajar", Type: "income", Category: "Gaji", Amount: 11000000, Note: strPtr("Gaji bulan ini"), Date: day(5), CreatedAt: now},
		{ID: "cf-2", UserID: "usr-fajar", Type: "expense", Category: "Makan & Minum", Amount: 1500000, Date: day(3), CreatedAt: now},
		{ID: "cf-3", UserID: "usr-raisa", Type: "income", Category: "Gaji", Amount: 8500000, Note: strPtr("Gaji bulan ini"), Date: day(5), CreatedAt: now},
	}
	db.Create(&cashflows)

	wedding := []models.WeddingCategory{
		{ID: "wc-1", CoupleID: "usr-fajar", Name: "Gedung / Venue", Budget: 45000000, Paid: 10000000, Vendor: strPtr("Ballroom Grha Sabha"), Status: "booked", Icon: "building", CreatedAt: now},
		{ID: "wc-2", CoupleID: "usr-fajar", Name: "Catering", Budget: 25000000, Paid: 0, Status: "planning", Icon: "utensils", CreatedAt: now},
		{ID: "wc-3", CoupleID: "usr-fajar", Name: "Cincin Nikah", Budget: 8000000, Paid: 8000000, Vendor: strPtr("Toko Emas Mulia"), Status: "done", Icon: "ring", CreatedAt: now},
	}
	db.Create(&wedding)

	plans := []models.Plan{
		{ID: "pl-1", UserID: "usr-fajar", Type: "together", Title: "Beli Rumah Pertama", Description: strPtr("KPR rumah area Bekasi"), Category: "Properti", Status: "planning", Subtasks: models.Subtasks{{ID: "st-1", Title: "Survei lokasi", Done: false}}, CreatedAt: now},
		{ID: "pl-2", UserID: "usr-raisa", Type: "partner", Title: "Kursus Baking", Description: strPtr("Ikut kelas baking profesional"), Category: "Pendidikan", Status: "idea", Subtasks: models.Subtasks{}, CreatedAt: now},
	}
	db.Create(&plans)

	goals := []models.Goal{
		{ID: "gl-1", UserID: "usr-fajar", Title: "Kumpulkan Dana Nikah 120jt", Category: "Finansial", Priority: "High", Status: "In Progress", Progress: 38, TargetAmount: f64Ptr(120000000), Subtasks: models.Subtasks{}, CreatedAt: now},
		{ID: "gl-2", UserID: "usr-raisa", Title: "Investasi Emas Rutin", Category: "Finansial", Priority: "Medium", Status: "In Progress", Progress: 55, Subtasks: models.Subtasks{}, CreatedAt: now},
	}
	db.Create(&goals)

	log.Println("Seeding done.")
}

// hashDemoPassword mirrors handlers.hashPassword (sha256 hex) without an
// import cycle; kept identical so the seeded demo accounts can log in with
// "demo123" out of the box.
func hashDemoPassword(pw string) string {
	return sha256Hex(pw)
}
