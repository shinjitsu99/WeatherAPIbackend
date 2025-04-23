package utils

import (
	"fmt"
	"log"
	"os"
	"time"

	"GoFiber/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	var err error
	var db *gorm.DB

	// Retry logic with exponential backoff
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}

		waitTime := time.Duration(i*i) * time.Second
		log.Printf("Connection attempt %d failed, retrying in %v...", i+1, waitTime)
		time.Sleep(waitTime)
	}

	if err != nil {
		log.Fatal("Failed to connect to database after retries:", err)
	}

	DB = db

	// Connection pool settings
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatal("Failed to get DB instance:", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("Successfully connected to database!")

	// Run migrations
	err = DB.AutoMigrate(&models.Translation{})
	if err != nil {
		log.Fatal("Database migration failed:", err)
	}
}
