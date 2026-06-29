package handlers

import "bv108-consumables-management-backend/internal/models"

func isOperationalManagedRole(role string) bool {
	switch normalizeRoleForPermissions(role) {
	case RoleNhanVienKho, RoleThuKho, RoleNhanVienKeToan, RoleNhanVienThau:
		return true
	default:
		return false
	}
}

func canAssignManagedRole(requesterRole, requestedRole string) bool {
	normalizedRequesterRole := normalizeRoleForPermissions(requesterRole)
	normalizedRequestedRole := normalizeRoleForPermissions(requestedRole)

	if !isAssignableRole(normalizedRequestedRole) {
		return false
	}

	switch normalizedRequesterRole {
	case RoleAdmin:
		return true
	case RoleChiHuyKhoa:
		return isOperationalManagedRole(normalizedRequestedRole)
	default:
		return false
	}
}

func canManageTargetUserRole(requesterRole, targetRole string) bool {
	normalizedRequesterRole := normalizeRoleForPermissions(requesterRole)
	normalizedTargetRole := normalizeRoleForPermissions(targetRole)

	switch normalizedRequesterRole {
	case RoleAdmin:
		return isAssignableRole(normalizedTargetRole)
	case RoleChiHuyKhoa:
		return isOperationalManagedRole(normalizedTargetRole)
	default:
		return false
	}
}

func filterManagedUserProfiles(requesterRole string, users []models.UserProfile) []models.UserProfile {
	filtered := make([]models.UserProfile, 0, len(users))
	for _, user := range users {
		if canManageTargetUserRole(requesterRole, user.Role) {
			filtered = append(filtered, user)
		}
	}

	return filtered
}
