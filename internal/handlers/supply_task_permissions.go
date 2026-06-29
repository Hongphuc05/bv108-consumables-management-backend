package handlers

import (
	"fmt"

	"bv108-consumables-management-backend/internal/models"
)

func isSupplyAssignmentEligibleRole(role string) bool {
	return normalizeRoleForPermissions(role) == RoleNhanVienThau
}

func loadSupplyAssignmentUser(userRepo *models.UserRepository, userID int64) (*models.User, error) {
	user, err := loadActiveUserByID(userRepo, userID)
	if err != nil {
		return nil, err
	}

	if !isSupplyAssignmentEligibleRole(user.Role) {
		return nil, fmt.Errorf("user is not eligible for supply assignments")
	}

	return user, nil
}
