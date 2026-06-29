package handlers

import (
	"testing"

	"bv108-consumables-management-backend/internal/models"
)

func TestCanAssignManagedRole(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		requesterRole string
		requestedRole string
		want          bool
	}{
		{name: "admin can assign admin", requesterRole: RoleAdmin, requestedRole: RoleAdmin, want: true},
		{name: "admin can assign chi huy khoa", requesterRole: RoleAdmin, requestedRole: RoleChiHuyKhoa, want: true},
		{name: "chi huy khoa cannot assign admin", requesterRole: RoleChiHuyKhoa, requestedRole: RoleAdmin, want: false},
		{name: "chi huy khoa cannot assign chi huy khoa", requesterRole: RoleChiHuyKhoa, requestedRole: RoleChiHuyKhoa, want: false},
		{name: "chi huy khoa can assign thu kho", requesterRole: RoleChiHuyKhoa, requestedRole: RoleThuKho, want: true},
		{name: "chi huy khoa can assign legacy nhan vien role", requesterRole: RoleChiHuyKhoa, requestedRole: RoleNhanVien, want: true},
		{name: "thu kho cannot assign nhan vien thau", requesterRole: RoleThuKho, requestedRole: RoleNhanVienThau, want: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := canAssignManagedRole(tc.requesterRole, tc.requestedRole)
			if got != tc.want {
				t.Fatalf("canAssignManagedRole(%q, %q) = %v, want %v", tc.requesterRole, tc.requestedRole, got, tc.want)
			}
		})
	}
}

func TestCanManageTargetUserRole(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		requesterRole string
		targetRole    string
		want          bool
	}{
		{name: "admin can manage chi huy khoa", requesterRole: RoleAdmin, targetRole: RoleChiHuyKhoa, want: true},
		{name: "admin can manage legacy nhan vien", requesterRole: RoleAdmin, targetRole: RoleNhanVien, want: true},
		{name: "chi huy khoa cannot manage admin", requesterRole: RoleChiHuyKhoa, targetRole: RoleAdmin, want: false},
		{name: "chi huy khoa cannot manage chi huy khoa", requesterRole: RoleChiHuyKhoa, targetRole: RoleChiHuyKhoa, want: false},
		{name: "chi huy khoa can manage nhan vien ke toan", requesterRole: RoleChiHuyKhoa, targetRole: RoleNhanVienKeToan, want: true},
		{name: "nhan vien kho cannot manage thu kho", requesterRole: RoleNhanVienKho, targetRole: RoleThuKho, want: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := canManageTargetUserRole(tc.requesterRole, tc.targetRole)
			if got != tc.want {
				t.Fatalf("canManageTargetUserRole(%q, %q) = %v, want %v", tc.requesterRole, tc.targetRole, got, tc.want)
			}
		})
	}
}

func TestFilterManagedUserProfiles(t *testing.T) {
	t.Parallel()

	users := []models.UserProfile{
		{ID: 1, Username: "admin", Role: RoleAdmin},
		{ID: 2, Username: "chief", Role: RoleChiHuyKhoa},
		{ID: 3, Username: "warehouse", Role: RoleThuKho},
		{ID: 4, Username: "buyer", Role: RoleNhanVienThau},
		{ID: 5, Username: "legacy", Role: RoleNhanVien},
	}

	t.Run("admin sees all assignable roles", func(t *testing.T) {
		t.Parallel()

		filtered := filterManagedUserProfiles(RoleAdmin, users)
		if len(filtered) != len(users) {
			t.Fatalf("admin filtered length = %d, want %d", len(filtered), len(users))
		}
	})

	t.Run("chi huy khoa only sees operational roles", func(t *testing.T) {
		t.Parallel()

		filtered := filterManagedUserProfiles(RoleChiHuyKhoa, users)
		if len(filtered) != 3 {
			t.Fatalf("chi huy khoa filtered length = %d, want 3", len(filtered))
		}

		for _, user := range filtered {
			if !isOperationalManagedRole(user.Role) {
				t.Fatalf("unexpected role %q in filtered result", user.Role)
			}
		}
	})
}
