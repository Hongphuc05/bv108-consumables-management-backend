package main

import (
	"context"
	"fmt"
	"log"

	"bv108-consumables-management-backend/config"
	"bv108-consumables-management-backend/internal/database"
	"bv108-consumables-management-backend/internal/models"
	"bv108-consumables-management-backend/internal/services"
)

func main() {
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("load config: %v", err)
	}

	if err := database.InitDB(); err != nil {
		log.Fatalf("init db: %v", err)
	}
	defer database.CloseDB()

	supplyRepo := models.NewSupplyRepository(database.DB)
	companyContactRepo := models.NewCompanyContactRepository(database.DB)
	
	syncService := services.NewInternalSupplySyncService(config.AppConfig, supplyRepo, companyContactRepo)

	fmt.Println("Starting sync service manually...")
	count, err := syncService.RunOnce(context.Background())
	if err != nil {
		log.Fatalf("sync error: %v", err)
	}

	fmt.Printf("Sync completed successfully! Synced %d supplies.\n", count)
}
