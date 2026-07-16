package database

import (
	"sms-identity/internal/domain"
	"sms-identity/internal/infrastructure/logger"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func SeedUsers(db *gorm.DB, adminEmail, adminPassword, userEmail, userPassword string) {
	if adminEmail == "" || adminPassword == "" {
		logger.Log.Sugar().Info("Admin credentials not set, skipping admin seeder.")
	} else {
		seedSingleUser(db, adminEmail, adminPassword, domain.RoleCodeAdmin)
	}

	if userEmail == "" || userPassword == "" {
		logger.Log.Sugar().Info("User credentials not set, skipping user seeder.")
	} else {
		seedSingleUser(db, userEmail, userPassword, domain.RoleCodeUser)
	}
}

func seedSingleUser(db *gorm.DB, email, password string, role domain.RoleCode) {
	var count int64
	db.Model(&domain.User{}).Where("email = ?", email).Count(&count)
	if count == 0 {
		logger.Log.Sugar().Infof("Seeding default %s user...", role)
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		db.Create(&domain.User{Email: email, Password: string(hash), RoleCode: role})
	}
}
