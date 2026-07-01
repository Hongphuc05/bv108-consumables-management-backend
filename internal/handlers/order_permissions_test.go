package handlers

import "testing"

func TestCanViewInvoiceWorkflowRole(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		role string
		want bool
	}{
		{name: "admin allowed", role: RoleAdmin, want: true},
		{name: "chi huy khoa allowed", role: RoleChiHuyKhoa, want: true},
		{name: "nhan vien kho allowed", role: RoleNhanVienKho, want: true},
		{name: "thu kho allowed", role: RoleThuKho, want: true},
		{name: "nhan vien ke toan allowed", role: RoleNhanVienKeToan, want: true},
		{name: "nhan vien thau allowed", role: RoleNhanVienThau, want: true},
		{name: "legacy nhan vien allowed", role: RoleNhanVien, want: true},
		{name: "unknown denied", role: "guest", want: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := canViewInvoiceWorkflowRole(tc.role)
			if got != tc.want {
				t.Fatalf("canViewInvoiceWorkflowRole(%q) = %v, want %v", tc.role, got, tc.want)
			}
		})
	}
}

func TestCanEditInvoiceWorkflowRole(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		role string
		want bool
	}{
		{name: "thu kho allowed", role: RoleThuKho, want: true},
		{name: "admin denied", role: RoleAdmin, want: false},
		{name: "chi huy khoa denied", role: RoleChiHuyKhoa, want: false},
		{name: "nhan vien ke toan denied", role: RoleNhanVienKeToan, want: false},
		{name: "legacy nhan vien denied", role: RoleNhanVien, want: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := canEditInvoiceWorkflowRole(tc.role)
			if got != tc.want {
				t.Fatalf("canEditInvoiceWorkflowRole(%q) = %v, want %v", tc.role, got, tc.want)
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
