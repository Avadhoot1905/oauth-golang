package storage

import (
	"log"

	"github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Storage provides database operations using GORM
type Storage struct {
	DB *gorm.DB
}

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

	// Seed development client
	SeedDevClient(db)
}

// SeedDevClient inserts a development OAuth client for testing
func SeedDevClient(db *gorm.DB) {
	client := OAuthClient{
		ClientID:     "demo-frontend",
		ClientSecret: "dev-secret",
		ClientName:   "Demo Frontend App",
		ClientType:   "public",
		RedirectURIs: pq.StringArray{"http://localhost:3000/callback"},
		GrantTypes:   pq.StringArray{"authorization_code", "refresh_token"},
		Scope:        "openid profile email",
	}

	result := db.Where(OAuthClient{ClientID: client.ClientID}).FirstOrCreate(&client)
	if result.Error != nil {
		log.Printf("Warning: failed to seed dev client: %v", result.Error)
	} else {
		log.Println("Development OAuth client seeded successfully")
	}
}
