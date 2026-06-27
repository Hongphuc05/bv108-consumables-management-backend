package handlers

import (
	"strings"

	"bv108-consumables-management-backend/internal/models"
)

func normalizeRoleForPermissions(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case RoleNhanVien:
		return RoleNhanVienKho
	default:
		return strings.ToLower(strings.TrimSpace(role))
	}
}

func formatRoleLabelForPermissions(role string) string {
	switch normalizeRoleForPermissions(role) {
	case RoleAdmin:
		return "Admin"
	case RoleChiHuyKhoa:
		return "Chỉ huy khoa"
	case RoleNhanVienKho:
		return "Nhân viên kho"
	case RoleThuKho:
		return "Thủ kho"
	case RoleNhanVienKeToan:
		return "Nhân viên kế toán"
	case RoleNhanVienThau:
		return "Nhân viên thầu"
	default:
		return strings.TrimSpace(role)
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
