package main

import (
	"log"

	"goals-api/config"
	"goals-api/handlers"
	appmw "goals-api/middleware"
	"goals-api/models"
	"goals-api/pkg/crypto"

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
		&models.FundChangeRequest{},
		&models.MainGoal{},
		&models.MainGoalContribution{},
		&models.MainGoalChangeRequest{},
		&models.WeddingTemplateSelection{},
		&models.WeddingTemplateChangeRequest{},
		&models.HabitProfile{},
		&models.HabitScheduleEntry{},
	); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// seed.Run(db)

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
	fundReqH := &handlers.FundRequestHandler{DB: db}
	mainGoalH := &handlers.MainGoalHandler{DB: db}
	weddingTmplH := &handlers.WeddingTemplateHandler{DB: db}
	habitH := &handlers.HabitHandler{DB: db, AnthropicAPIKey: cfg.AnthropicAPIKey}

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
	api.GET("/auth/partner", authH.GetPartner)
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

	// ---- Target dana bersama: perlu persetujuan pasangan ----
	api.GET("/fund-requests", fundReqH.List)
	api.POST("/fund-requests", fundReqH.Create)
	api.POST("/fund-requests/:id/approve", fundReqH.Approve)
	api.POST("/fund-requests/:id/reject", fundReqH.Reject)

	// ---- FITUR 1: Goals Utama Pasangan ----
	api.GET("/main-goal", mainGoalH.Get)
	api.GET("/main-goal/history", mainGoalH.History)
	api.POST("/main-goal", mainGoalH.Create)
	api.PATCH("/main-goal/:id", mainGoalH.Edit)
	api.GET("/main-goal-requests", mainGoalH.ListRequests)
	api.POST("/main-goal-requests/:id/approve", mainGoalH.Approve)
	api.POST("/main-goal-requests/:id/reject", mainGoalH.Reject)
	api.POST("/main-goal/contribute", mainGoalH.ContributeFromWallet)

	// ---- FITUR 2: Template Wedding Planning ----
	api.GET("/wedding/templates", weddingTmplH.ListTemplates)
	api.GET("/wedding/template-selection", weddingTmplH.GetSelection)
	api.POST("/wedding/apply-template", weddingTmplH.ApplyTemplate)
	api.GET("/wedding-template-requests", weddingTmplH.ListRequests)
	api.POST("/wedding-template-requests/:id/approve", weddingTmplH.Approve)
	api.POST("/wedding-template-requests/:id/reject", weddingTmplH.Reject)

	// ---- FITUR 3: Notif Habit (jadwal otomatis) ----
	api.GET("/habit/profile", habitH.GetProfile)
	api.POST("/habit/profile", habitH.SaveProfile)
	api.GET("/habit/schedule", habitH.GetSchedule)
	api.GET("/habit/schedule/partner", habitH.GetPartnerSchedule)
	api.POST("/habit/schedule", habitH.CreateEntry)
	api.PATCH("/habit/schedule/:id", habitH.UpdateEntry)
	api.DELETE("/habit/schedule/:id", habitH.DeleteEntry)

	// ---- Backup / restore (full JSON export & import + 28-day reminder) ----
	api.GET("/backup/export", backupH.Export)
	api.POST("/backup/import", backupH.Import)
	api.GET("/backup/status", backupH.Status)

	log.Printf("Goals API listening on :%s (MySQL: %s@%s:%s/%s)", cfg.AppPort, cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)
	e.Logger.Fatal(e.Start(":" + cfg.AppPort))
}