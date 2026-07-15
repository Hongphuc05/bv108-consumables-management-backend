//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"log"

	"bv108-consumables-management-backend/config"
	"bv108-consumables-management-backend/internal/database"
)

func main() {
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("load config: %v", err)
	}

	if err := database.InitDB(); err != nil {
		log.Fatalf("init db: %v", err)
	}
	defer database.CloseDB()

	// 1. Get visibility setting
	var hideForOtherRoles int
	err := database.DB.QueryRow("SELECT hide_for_other_roles FROM supply_visibility_settings WHERE scope_key = 'global'").Scan(&hideForOtherRoles)
	if err != nil {
		fmt.Printf("Error getting settings: %v\n", err)
	} else {
		fmt.Printf("hide_for_other_roles: %d\n", hideForOtherRoles)
	}

	// 2. Count supplies
	var totalSupplies int
	database.DB.QueryRow("SELECT COUNT(*) FROM supplies").Scan(&totalSupplies)
	fmt.Printf("Total supplies in DB: %d\n", totalSupplies)

	// 3. Count assignments
	var totalAssignments int
	database.DB.QueryRow("SELECT COUNT(*) FROM supply_user_assignments").Scan(&totalAssignments)
	fmt.Printf("Total supply_user_assignments in DB: %d\n", totalAssignments)

	// 4. Count company contacts
	var totalContacts int
	database.DB.QueryRow("SELECT COUNT(*) FROM company_contacts").Scan(&totalContacts)
	fmt.Printf("Total company_contacts in DB: %d\n", totalContacts)

	// 5. List users and their roles
	rows, err := database.DB.Query("SELECT id, username, email, role FROM users")
	if err != nil {
		log.Fatalf("query users: %v", err)
	}
	defer rows.Close()

	fmt.Println("\nUsers in DB:")
	for rows.Next() {
		var id int64
		var username, email, role string
		rows.Scan(&id, &username, &email, &role)

		// Get assignment count for this user
		var userAssigns int
		database.DB.QueryRow("SELECT COUNT(*) FROM supply_user_assignments WHERE user_id = ?", id).Scan(&userAssigns)

		fmt.Printf("- ID=%d: %s (%s) - Role=%s - Assigned supplies=%d\n", id, username, email, role, userAssigns)
	}
}
