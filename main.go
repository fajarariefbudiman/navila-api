package main

import (
	"log"

	"goals-api/config"
	"goals-api/handlers"
	appmw "goals-api/middleware"
	"goals-api/models"
	"goals-api/pkg/crypto"
	"goals-api/seed"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to MySQL: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Wallet{},
		&models.Budget{},
		&models.Transaction{},
		&models.CashFlow{},
		&models.WeddingCategory{},
		&models.Plan{},
		&models.Goal{},
		&models.BackupLog{},
	); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	seed.Run(db)

	box, err := crypto.NewBox(cfg.EncKeyHex)
	if err != nil {
		log.Fatalf("failed to init encryption box (check APP_ENC_KEY): %v", err)
	}

	e := echo.New()
	e.HideBanner = true
	e.Use(echomw.Logger())
	e.Use(echomw.Recover())
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins: []string{cfg.AllowOrigin},
		AllowHeaders: []string{echo.HeaderContentType, echo.HeaderAuthorization},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
	}))
	// Every request/response body below is transparently AES-256-GCM
	// encrypted/decrypted using the shared key in .env (APP_ENC_KEY).
	e.Use(appmw.Encryption(box))

	authH := &handlers.AuthHandler{DB: db, JWTSecret: cfg.JWTSecret}
	walletH := &handlers.WalletHandler{DB: db}
	budgetH := &handlers.BudgetHandler{DB: db}
	txH := &handlers.TransactionHandler{DB: db}
	cfH := &handlers.CashflowHandler{DB: db}
	weddingH := &handlers.WeddingHandler{DB: db}
	planH := &handlers.PlanHandler{DB: db}
	goalH := &handlers.GoalHandler{DB: db}
	backupH := &handlers.BackupHandler{DB: db}

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, echo.Map{"status": "ok"})
	})

	// ---- Public auth routes ----
	auth := e.Group("/auth")
	auth.POST("/register", authH.Register)
	auth.POST("/login", authH.Login)

	// ---- Protected routes (JWT required) ----
	api := e.Group("", appmw.JWTAuth(cfg.JWTSecret))

	api.GET("/auth/me", authH.Me)
	api.POST("/auth/link-partner", authH.LinkPartner)
	api.POST("/auth/unlink-partner", authH.UnlinkPartner)
	api.PATCH("/auth/profile", authH.UpdateProfile)

	api.GET("/wallets", walletH.List)
	api.POST("/wallets", walletH.Create)
	api.PATCH("/wallets/:id", walletH.Update)
	api.DELETE("/wallets/:id", walletH.Delete)

	api.GET("/budgets", budgetH.List)
	api.POST("/budgets", budgetH.Create)
	api.PATCH("/budgets/:id", budgetH.Update)
	api.DELETE("/budgets/:id", budgetH.Delete)

	api.GET("/transactions", txH.List)
	api.POST("/transactions", txH.Create)
	api.DELETE("/transactions/:id", txH.Delete)

	api.GET("/cashflows", cfH.List)
	api.POST("/cashflows", cfH.Create)
	api.DELETE("/cashflows/:id", cfH.Delete)

	api.GET("/wedding", weddingH.List)
	api.POST("/wedding", weddingH.Create)
	api.PATCH("/wedding/:id", weddingH.Update)
	api.DELETE("/wedding/:id", weddingH.Delete)

	api.GET("/plans", planH.List)
	api.POST("/plans", planH.Create)
	api.PATCH("/plans/:id", planH.Update)
	api.DELETE("/plans/:id", planH.Delete)

	api.GET("/goals", goalH.List)
	api.POST("/goals", goalH.Create)
	api.PATCH("/goals/:id", goalH.Update)
	api.DELETE("/goals/:id", goalH.Delete)

	// ---- Backup / restore (full JSON export & import + 28-day reminder) ----
	api.GET("/backup/export", backupH.Export)
	api.POST("/backup/import", backupH.Import)
	api.GET("/backup/status", backupH.Status)

	log.Printf("Goals API listening on :%s (MySQL: %s@%s:%s/%s)", cfg.AppPort, cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)
	e.Logger.Fatal(e.Start(":" + cfg.AppPort))
}
