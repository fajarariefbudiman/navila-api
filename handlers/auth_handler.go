package handlers

import (
	"math/rand"
	"net/http"
	"strings"
	"time"

	"goals-api/middleware"
	"goals-api/models"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	DB        *gorm.DB
	JWTSecret string
}

// hashPassword hashes a raw password with bcrypt (cost 12). bcrypt salts
// automatically and is deliberately slow, unlike a bare SHA-256 digest.
func hashPassword(pw string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// checkPassword compares a raw password against a bcrypt hash.
func checkPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

func genID(prefix string) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 10)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return prefix + "-" + string(b)
}

func genPartnerCode() string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

type registerReq struct {
	Name              string  `json:"name"`
	Email             string  `json:"email"`
	Password          string  `json:"password"`
	Gender            string  `json:"gender"`
	Mode              string  `json:"mode"`
	WeddingDate       string  `json:"weddingDate"`
	TargetWeddingFund float64 `json:"targetWeddingFund"`
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req registerReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	var existing models.User
	if err := h.DB.Where("email = ?", strings.ToLower(req.Email)).First(&existing).Error; err == nil {
		return c.JSON(http.StatusConflict, echo.Map{"error": "Email sudah terdaftar"})
	}
	mode := req.Mode
	if mode != "myKisah" {
		mode = "jomblo"
	}
	hashed, err := hashPassword(req.Password)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to process password"})
	}
	u := models.User{
		ID:                genID("usr"),
		Name:              req.Name,
		Email:             strings.ToLower(req.Email),
		Password:          hashed,
		Gender:            req.Gender,
		Mode:              mode,
		PartnerCode:       genPartnerCode(),
		WeddingDate:       &req.WeddingDate,
		TargetWeddingFund: req.TargetWeddingFund,
		CreatedAt:         time.Now(),
	}
	if err := h.DB.Create(&u).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create user"})
	}
	token, _ := middleware.GenerateToken(u.ID, h.JWTSecret)
	return c.JSON(http.StatusCreated, echo.Map{"user": u, "token": token})
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req loginReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	var u models.User
	if err := h.DB.Where("email = ?", strings.ToLower(req.Email)).First(&u).Error; err != nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Email atau password salah"})
	}
	if !checkPassword(u.Password, req.Password) {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "Email atau password salah"})
	}
	token, _ := middleware.GenerateToken(u.ID, h.JWTSecret)
	return c.JSON(http.StatusOK, echo.Map{"user": u, "token": token})
}

func (h *AuthHandler) Me(c echo.Context) error {
	userID := c.Get("userId").(string)
	var u models.User
	if err := h.DB.First(&u, "id = ?", userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "user not found"})
	}
	return c.JSON(http.StatusOK, u)
}

func (h *AuthHandler) GetPartner(c echo.Context) error {
	userID := c.Get("userId").(string)
	var me models.User
	if err := h.DB.First(&me, "id = ?", userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "user not found"})
	}
	if me.PartnerID == nil || *me.PartnerID == "" {
		return c.JSON(http.StatusOK, nil)
	}
	var partner models.User
	if err := h.DB.First(&partner, "id = ?", *me.PartnerID).Error; err != nil {
		return c.JSON(http.StatusOK, nil)
	}
	return c.JSON(http.StatusOK, partner)
}

type linkPartnerReq struct {
	PartnerCode string `json:"partnerCode"`
}

func (h *AuthHandler) LinkPartner(c echo.Context) error {
	userID := c.Get("userId").(string)
	var req linkPartnerReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	var partner models.User
	code := strings.ToUpper(req.PartnerCode)
	if err := h.DB.Where("partner_code = ? AND id <> ?", code, userID).First(&partner).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "Kode pasangan tidak ditemukan"})
	}
	if partner.PartnerID != nil {
		return c.JSON(http.StatusConflict, echo.Map{"error": "Pasangan sudah terhubung dengan akun lain"})
	}
	if err := h.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{"partner_id": partner.ID, "mode": "myKisah"}).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to link partner"})
	}
	if err := h.DB.Model(&models.User{}).Where("id = ?", partner.ID).Updates(map[string]interface{}{"partner_id": userID, "mode": "myKisah"}).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to link partner"})
	}
	var updated models.User
	h.DB.First(&updated, "id = ?", userID)
	return c.JSON(http.StatusOK, updated)
}

func (h *AuthHandler) UnlinkPartner(c echo.Context) error {
	userID := c.Get("userId").(string)
	var me models.User
	if err := h.DB.First(&me, "id = ?", userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "user not found"})
	}
	if me.PartnerID != nil {
		h.DB.Model(&models.User{}).Where("id = ?", *me.PartnerID).
			Updates(map[string]interface{}{"partner_id": nil, "mode": "jomblo"})
	}
	if err := h.DB.Model(&models.User{}).Where("id = ?", userID).
		Updates(map[string]interface{}{"partner_id": nil, "mode": "jomblo"}).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to unlink partner"})
	}
	var updated models.User
	h.DB.First(&updated, "id = ?", userID)
	return c.JSON(http.StatusOK, updated)
}

func (h *AuthHandler) UpdateProfile(c echo.Context) error {
	userID := c.Get("userId").(string)
	var patch map[string]interface{}
	if err := c.Bind(&patch); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
	}
	delete(patch, "id")
	delete(patch, "email")
	delete(patch, "password")
	delete(patch, "partnerCode")
	patch = camelKeysToSnake(patch)
	if err := h.DB.Model(&models.User{}).Where("id = ?", userID).Updates(patch).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update"})
	}
	var updated models.User
	h.DB.First(&updated, "id = ?", userID)
	return c.JSON(http.StatusOK, updated)
}