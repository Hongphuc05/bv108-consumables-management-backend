package handlers

import "testing"

func TestIsSupplyAssignmentEligibleRole(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		role string
		want bool
	}{
		{name: "nhan vien thau is eligible", role: RoleNhanVienThau, want: true},
		{name: "legacy nhan vien is not eligible", role: RoleNhanVien, want: false},
		{name: "thu kho is not eligible", role: RoleThuKho, want: false},
		{name: "admin is not eligible", role: RoleAdmin, want: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isSupplyAssignmentEligibleRole(tc.role)
			if got != tc.want {
				t.Fatalf("isSupplyAssignmentEligibleRole(%q) = %v, want %v", tc.role, got, tc.want)
			}
		})
	}
}
