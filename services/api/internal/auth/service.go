package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jobright/api/internal/forge"
	"github.com/jobright/api/internal/middleware"
	"github.com/jobright/api/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrEmailTaken = errors.New("email taken")

type Service struct {
	db        *gorm.DB
	forge     *forge.Client
	jwtSecret string
	jwtExpiry time.Duration
}

func NewService(db *gorm.DB, forgeClient *forge.Client, jwtSecret string, jwtExpiry time.Duration) *Service {
	return &Service{db: db, forge: forgeClient, jwtSecret: jwtSecret, jwtExpiry: jwtExpiry}
}

func (s *Service) Signup(email, password, name string) (*models.User, string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}
	user := &models.User{
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
	}
	if err := s.db.Create(user).Error; err != nil {
		if isUniqueViolation(err) {
			return nil, "", ErrEmailTaken
		}
		return nil, "", err
	}
	s.linkForge(user, email, password, name)
	token, err := s.issueToken(user)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

func (s *Service) Login(email, password string) (*models.User, string, error) {
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", err
	}
	s.linkForge(&user, email, password, user.Name)
	token, err := s.issueToken(&user)
	if err != nil {
		return nil, "", err
	}
	return &user, token, nil
}

func (s *Service) GetByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Service) EnsureForgeToken(user *models.User) (string, error) {
	if user.ForgeAccessToken != "" {
		return user.ForgeAccessToken, nil
	}
	if user.ForgeRefreshToken != "" {
		auth, err := s.forge.Refresh(user.ForgeRefreshToken)
		if err == nil && auth.AccessToken != "" {
			s.saveForgeTokens(user, auth)
			return auth.AccessToken, nil
		}
	}
	return "", errors.New("resume forge session missing; log in again to reconnect")
}

func (s *Service) linkForge(user *models.User, email, password, name string) {
	if s.forge == nil {
		return
	}
	display := name
	if display == "" {
		display = strings.Split(email, "@")[0]
	}
	auth, err := s.forge.Login(email, password)
	if err != nil {
		auth, err = s.forge.Register(email, password, display)
		if err != nil {
			return
		}
	}
	if auth.AccessToken == "" {
		return
	}
	s.saveForgeTokens(user, auth)
}

func (s *Service) saveForgeTokens(user *models.User, auth *forge.AuthResult) {
	user.ForgeAccessToken = auth.AccessToken
	user.ForgeRefreshToken = auth.RefreshToken
	user.ForgeUserID = auth.UserID
	_ = s.db.Model(user).Updates(map[string]any{
		"forge_access_token":  user.ForgeAccessToken,
		"forge_refresh_token": user.ForgeRefreshToken,
		"forge_user_id":       user.ForgeUserID,
	}).Error
}

func (s *Service) issueToken(user *models.User) (string, error) {
	claims := middleware.Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID.String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func isUniqueViolation(err error) bool {
	return err != nil && (errors.Is(err, gorm.ErrDuplicatedKey) ||
		strings.Contains(strings.ToLower(err.Error()), "duplicate") ||
		strings.Contains(err.Error(), "UNIQUE"))
}
