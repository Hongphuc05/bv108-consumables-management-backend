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

	supplyRepo := models.NewSupplyRepository(database.DB)
	companyContactRepo := models.NewCompanyContactRepository(database.DB)

	// Mode 1: mapping
	fmt.Println("=== TESTING CONFIGURATION MODE: mapping ===")
	config.AppConfig.SupplyMappingTable = "mapping"
	os.Setenv("SUPPLY_MAPPING_TABLE", "mapping")

	syncService1 := services.NewInternalSupplySyncService(config.AppConfig, supplyRepo, companyContactRepo)
	_, err := syncService1.RunOnce(context.Background())
	if err != nil {
		log.Fatalf("sync mode mapping error: %v", err)
	}

	printSupplies("TEST001")

	// Mode 2: mapping2
	fmt.Println("\n=== TESTING CONFIGURATION MODE: mapping2 ===")
	config.AppConfig.SupplyMappingTable = "mapping2"
	os.Setenv("SUPPLY_MAPPING_TABLE", "mapping2")

	syncService2 := services.NewInternalSupplySyncService(config.AppConfig, supplyRepo, companyContactRepo)
	_, err = syncService2.RunOnce(context.Background())
	if err != nil {
		log.Fatalf("sync mode mapping2 error: %v", err)
	}

	printSupplies("TEST001")
}

func printSupplies(id string) {
	var name, groupName, qcdg, qcgh, qctt, tt string
	var tkm int
	err := database.DB.QueryRow(`
		SELECT NAME, GROUPNAME, QUY_CACH_DONG_GOI, QUY_CACH_GIAO_HANG, QUY_CACH_TOI_THIEU, TON_KHO_MIN, TONGTHAU 
		FROM supplies WHERE ID = ?`, id).Scan(&name, &groupName, &qcdg, &qcgh, &qctt, &tkm, &tt)
	if err != nil {
		fmt.Printf("Error fetching %s: %v\n", id, err)
		return
	}
	fmt.Printf("ID: %s | Name: %q\n", id, name)
	fmt.Printf("  * GROUPNAME: %q\n", groupName)
	fmt.Printf("  * QUY_CACH_DONG_GOI: %q\n", qcdg)
	fmt.Printf("  * QUY_CACH_GIAO_HANG: %q\n", qcgh)
	fmt.Printf("  * QUY_CACH_TOI_THIEU: %q\n", qctt)
	fmt.Printf("  * TON_KHO_MIN: %d\n", tkm)
	fmt.Printf("  * TONGTHAU: %q\n", tt)
}
