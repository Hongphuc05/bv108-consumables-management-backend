package handlers

import "testing"

func TestCanRunInternalSupplySyncRole(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		role string
		want bool
	}{
		{name: "admin allowed", role: RoleAdmin, want: true},
		{name: "admin with spaces and casing allowed", role: " Admin ", want: true},
		{name: "chi huy khoa denied", role: RoleChiHuyKhoa, want: false},
		{name: "thu kho denied", role: RoleThuKho, want: false},
		{name: "nhan vien thau denied", role: RoleNhanVienThau, want: false},
		{name: "empty denied", role: "", want: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := canRunInternalSupplySyncRole(tc.role)
			if got != tc.want {
				t.Fatalf("canRunInternalSupplySyncRole(%q) = %v, want %v", tc.role, got, tc.want)
			}
		})
	}
}
