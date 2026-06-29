package handlers

import "testing"

func TestCanManageInvoiceWorkflowRole(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		role string
		want bool
	}{
		{name: "admin allowed", role: RoleAdmin, want: true},
		{name: "chi huy khoa allowed", role: RoleChiHuyKhoa, want: true},
		{name: "nhan vien ke toan allowed", role: RoleNhanVienKeToan, want: true},
		{name: "thu kho denied", role: RoleThuKho, want: false},
		{name: "nhan vien thau denied", role: RoleNhanVienThau, want: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := canManageInvoiceWorkflowRole(tc.role)
			if got != tc.want {
				t.Fatalf("canManageInvoiceWorkflowRole(%q) = %v, want %v", tc.role, got, tc.want)
			}
		})
	}
}

func TestCanCreateManualOrderRole(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		role string
		want bool
	}{
		{name: "admin allowed", role: RoleAdmin, want: true},
		{name: "chi huy khoa allowed", role: RoleChiHuyKhoa, want: true},
		{name: "nhan vien ke toan denied", role: RoleNhanVienKeToan, want: false},
		{name: "thu kho denied", role: RoleThuKho, want: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := canCreateManualOrderRole(tc.role)
			if got != tc.want {
				t.Fatalf("canCreateManualOrderRole(%q) = %v, want %v", tc.role, got, tc.want)
			}
		})
	}
}
