package database

import (
	"fmt"
	"log"
	"os"
	"obitel-wiki/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	dbHost := GetEnv("DB_HOST", "localhost")
	dbUser := GetEnv("DB_USER", "postgres")
	dbPass := GetEnv("DB_PASSWORD", "postgres")
	dbName := GetEnv("DB_NAME", "wiki")
	dbPort := GetEnv("DB_PORT", "5432")

	adminUser := GetEnv("ADMIN_USER", "admin")
	adminPass := GetEnv("ADMIN_PASSWORD", "admin123")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC", 
		dbHost, dbUser, dbPass, dbName, dbPort)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// миграция
	err = DB.AutoMigrate(&models.User{}, &models.Article{}, &models.Block{}, &models.Image{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
	
	log.Println("Database connected and migrated successfully")

	// создаем первого админа при запуске, если его нет
	var count int64
	DB.Model(&models.User{}).Where("role = ?", "admin").Count(&count)
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
		if err != nil {
			log.Fatal("Failed to hash password:", err)
		}

		admin := models.User{
			Username: adminUser, 
			Password: string(hash), 
			Role:     "admin",
		}
		DB.Create(&admin)
		log.Println("==================================================")
		log.Println("Default admin created!")
		log.Println("Login: " + adminUser)
		log.Println("Password: " + adminPass)
		log.Println("==================================================")
	}
}

func GetEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}