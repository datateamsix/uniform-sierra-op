package db

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	//Load project config and models
	"url-shortener/config"
	"url-shortener/models"
)

var DB *gorm.DB

func InitDatabase(cfg config.Config) {
	var err error
	DB, err = gorm.Open(postgres.Open(cfg.DBConnectionString), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate the models
	err = DB.AutoMigrate(&models.UrlMapping{}, &models.MaliciousLog{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
}
