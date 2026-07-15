package main

import (
	"context"
	"fmt"
	"log"
	"os"

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

	// Override config manually to test "mapping2" table mode
	fmt.Println("--- OVERRIDING SUPPLY_MAPPING_TABLE TO 'mapping2' ---")
	config.AppConfig.SupplyMappingTable = "mapping2"
	os.Setenv("SUPPLY_MAPPING_TABLE", "mapping2")

	supplyRepo := models.NewSupplyRepository(database.DB)
	companyContactRepo := models.NewCompanyContactRepository(database.DB)

	// Trigger sync
	syncService := services.NewInternalSupplySyncService(config.AppConfig, supplyRepo, companyContactRepo)
	count, err := syncService.RunOnce(context.Background())
	if err != nil {
		log.Fatalf("sync error: %v", err)
	}
	fmt.Printf("Sync completed. Total synced supplies: %d\n", count)

	// Fetch fake items to check they are filled from the 'mapping' table
	fmt.Println("\n--- CHECKING INJECTED TEST SUPPLIES IN DB (MAPPING MODE) ---")
	testIDs := []string{"TEST001", "TEST002", "TEST003"}
	for _, id := range testIDs {
		var name, groupName, qcdg, qcgh, qctt, tt string
		var tkm int
		err = database.DB.QueryRow(`
			SELECT NAME, GROUPNAME, QUY_CACH_DONG_GOI, QUY_CACH_GIAO_HANG, QUY_CACH_TOI_THIEU, TON_KHO_MIN, TONGTHAU 
			FROM supplies WHERE ID = ?`, id).Scan(&name, &groupName, &qcdg, &qcgh, &qctt, &tkm, &tt)
		if err != nil {
			fmt.Printf("Error fetching %s: %v\n", id, err)
			continue
		}
		fmt.Printf("ID: %s | Name: %q\n", id, name)
		fmt.Printf("  * GROUPNAME: %q\n", groupName)
		fmt.Printf("  * QUY_CACH_DONG_GOI: %q\n", qcdg)
		fmt.Printf("  * QUY_CACH_GIAO_HANG: %q\n", qcgh)
		fmt.Printf("  * QUY_CACH_TOI_THIEU: %q\n", qctt)
		fmt.Printf("  * TON_KHO_MIN: %d\n", tkm)
		fmt.Printf("  * TONGTHAU: %q (Should be empty in mapping table)\n", tt)
	}
}
