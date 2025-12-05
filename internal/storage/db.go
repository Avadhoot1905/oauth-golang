package storage

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB(dsn string) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	DB = db

	// Auto-migrate the schemas
	err = db.AutoMigrate(&User{}, &OAuthClient{}, &RefreshToken{})
	if err != nil {
		log.Fatal("failed to migrate database:", err)
	}

	log.Println("Database connected and tables migrated successfully")
}
