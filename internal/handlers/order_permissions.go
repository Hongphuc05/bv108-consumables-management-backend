package handlers

func canViewInvoiceWorkflowRole(role string) bool {
	switch normalizeRoleForPermissions(role) {
	case RoleAdmin, RoleChiHuyKhoa, RoleNhanVienKho, RoleThuKho, RoleNhanVienKeToan, RoleNhanVienThau:
		return true
	default:
		return false
	}
}

func canEditInvoiceWorkflowRole(role string) bool {
	switch normalizeRoleForPermissions(role) {
	case RoleThuKho:
		return true
	default:
		return false
	}
}

func canCreateManualOrderRole(role string) bool {
	switch normalizeRoleForPermissions(role) {
	case RoleAdmin, RoleChiHuyKhoa:
		return true
	default:
		return false
	}
}
