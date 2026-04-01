package handlers

import (
	"strings"

	"bv108-consumables-management-backend/internal/models"
)

func normalizeRoleForPermissions(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case RoleTruongKhoa:
		return RoleAdmin
	case RoleNhanVien:
		return RoleNhanVienKho
	default:
		return strings.ToLower(strings.TrimSpace(role))
	}
}

func userHasAnyRole(user *models.UserProfile, roles ...string) bool {
	if user == nil {
		return false
	}

	normalizedUserRole := normalizeRoleForPermissions(user.Role)
	for _, role := range roles {
		if normalizedUserRole == normalizeRoleForPermissions(role) {
			return true
		}
	}

	return false
}
