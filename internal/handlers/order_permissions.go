package handlers

func canManageInvoiceWorkflowRole(role string) bool {
	switch normalizeRoleForPermissions(role) {
	case RoleAdmin, RoleChiHuyKhoa, RoleNhanVienKeToan:
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
