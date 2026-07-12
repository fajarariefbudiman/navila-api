package handlers

import (
	"net/http"
	"time"

	"goals-api/models"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type BackupHandler struct{ DB *gorm.DB }

const BackupReminderDays = 28

type backupPayload struct {
	ExportedAt   time.Time                `json:"exportedAt"`
	User         models.User              `json:"user"`
	Partner      *models.User             `json:"partner"`
	Wallets      []models.Wallet          `json:"wallets"`
	Budgets      []models.Budget          `json:"budgets"`
	Transactions []models.Transaction     `json:"transactions"`
	Cashflows    []models.CashFlow        `json:"cashflows"`
	Wedding      []models.WeddingCategory `json:"wedding"`
	Plans        []models.Plan            `json:"plans"`
	Goals        []models.Goal            `json:"goals"`
}

func (h *BackupHandler) collectUserIDs(userID string) (models.User, []string) {
	var u models.User
	h.DB.First(&u, "id = ?", userID)
	ids := []string{userID}
	if u.PartnerID != nil {
		ids = append(ids, *u.PartnerID)
	}
	return u, ids
}

// Export bundles every table's rows for the logged-in user (and partner,
// where data is shared) into a single JSON document the frontend can
// download as a backup file, and can later re-import with Import below.
func (h *BackupHandler) Export(c echo.Context) error {
	userID := c.Get("userId").(string)
	u, ids := h.collectUserIDs(userID)

	var partner *models.User
	if u.PartnerID != nil {
		var p models.User
		if err := h.DB.First(&p, "id = ?", *u.PartnerID).Error; err == nil {
			partner = &p
		}
	}

	payload := backupPayload{ExportedAt: time.Now(), User: u, Partner: partner}
	h.DB.Where("user_id IN ?", ids).Find(&payload.Wallets)
	h.DB.Where("user_id = ?", userID).Find(&payload.Budgets)
	h.DB.Where("user_id IN ?", ids).Find(&payload.Transactions)
	h.DB.Where("user_id IN ?", ids).Find(&payload.Cashflows)
	h.DB.Where("couple_id IN ?", ids).Find(&payload.Wedding)
	h.DB.Where("user_id IN ?", ids).Find(&payload.Plans)
	h.DB.Where("user_id IN ?", ids).Find(&payload.Goals)

	now := time.Now()
	h.DB.Model(&models.User{}).Where("id = ?", userID).Update("last_backup_at", now)
	h.DB.Create(&models.BackupLog{UserID: userID, Action: "export", CreatedAt: now})

	return c.JSON(http.StatusOK, payload)
}

// Import accepts a previously exported JSON document and re-inserts every
// row that belongs to the current user (rows are matched/replaced by ID, so
// importing twice is safe / idempotent).
func (h *BackupHandler) Import(c echo.Context) error {
	userID := c.Get("userId").(string)
	var body backupPayload
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid backup file"})
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		for i := range body.Wallets {
			body.Wallets[i].UserID = userID
			if err := tx.Save(&body.Wallets[i]).Error; err != nil {
				return err
			}
		}
		for i := range body.Budgets {
			body.Budgets[i].UserID = userID
			if err := tx.Save(&body.Budgets[i]).Error; err != nil {
				return err
			}
		}
		for i := range body.Transactions {
			body.Transactions[i].UserID = userID
			if err := tx.Save(&body.Transactions[i]).Error; err != nil {
				return err
			}
		}
		for i := range body.Cashflows {
			body.Cashflows[i].UserID = userID
			if err := tx.Save(&body.Cashflows[i]).Error; err != nil {
				return err
			}
		}
		for i := range body.Wedding {
			if err := tx.Save(&body.Wedding[i]).Error; err != nil {
				return err
			}
		}
		for i := range body.Plans {
			if err := tx.Save(&body.Plans[i]).Error; err != nil {
				return err
			}
		}
		for i := range body.Goals {
			if err := tx.Save(&body.Goals[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to import backup"})
	}

	now := time.Now()
	h.DB.Create(&models.BackupLog{UserID: userID, Action: "import", CreatedAt: now})

	return c.JSON(http.StatusOK, echo.Map{"imported": true})
}

// Status tells the frontend when the last backup happened and whether the
// 28-day reminder should be shown.
func (h *BackupHandler) Status(c echo.Context) error {
	userID := c.Get("userId").(string)
	var u models.User
	if err := h.DB.First(&u, "id = ?", userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "user not found"})
	}

	dueForBackup := true
	var nextDue *time.Time
	if u.LastBackupAt != nil {
		due := u.LastBackupAt.AddDate(0, 0, BackupReminderDays)
		nextDue = &due
		dueForBackup = time.Now().After(due)
	}

	return c.JSON(http.StatusOK, echo.Map{
		"lastBackupAt":      u.LastBackupAt,
		"nextDueAt":         nextDue,
		"reminderDays":      BackupReminderDays,
		"shouldShowReminder": dueForBackup,
	})
}
