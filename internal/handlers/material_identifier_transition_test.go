package handlers

import (
	"testing"

	"bv108-consumables-management-backend/internal/models"
)

func TestBuildForecastApprovalInputAllowsTypeNameWithoutLegacyID(t *testing.T) {
	t.Parallel()

	input, err := buildForecastApprovalInput(SaveForecastApprovalRequest{
		ForecastMonth: 7,
		ForecastYear:  2030,
		MaQuanLy:      " TYPE-001 ",
		TenVtytBv:     "Vật tư mới",
		Status:        models.ForecastApprovalStatusEdited,
	}, &models.UserProfile{ID: 1, Username: "tester"})
	if err != nil {
		t.Fatalf("buildForecastApprovalInput() error = %v", err)
	}
	if input.MaQuanLy != "TYPE-001" || input.MaVtytCu != "" {
		t.Fatalf("identifiers = (%q, %q), want (TYPE-001, empty)", input.MaQuanLy, input.MaVtytCu)
	}
}

func TestBuildForecastApprovalInputFallsBackToLegacyID(t *testing.T) {
	t.Parallel()

	input, err := buildForecastApprovalInput(SaveForecastApprovalRequest{
		ForecastMonth: 7,
		ForecastYear:  2030,
		MaVtytCu:      " OLD-001 ",
		TenVtytBv:     "Vật tư cũ",
		Status:        models.ForecastApprovalStatusEdited,
	}, &models.UserProfile{ID: 1, Username: "tester"})
	if err != nil {
		t.Fatalf("buildForecastApprovalInput() error = %v", err)
	}
	if input.MaQuanLy != "OLD-001" || input.MaVtytCu != "OLD-001" {
		t.Fatalf("identifiers = (%q, %q), want legacy fallback in both fields", input.MaQuanLy, input.MaVtytCu)
	}
}
