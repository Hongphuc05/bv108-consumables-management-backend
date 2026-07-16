package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bv108-consumables-management-backend/config"
	"bv108-consumables-management-backend/internal/models"
)

func TestInternalSupplySyncUsesOnlyRemoteRowsAndMapsPartnerSchema(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api_trangbi_thongtinvattu":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{
				{
					"id":                  101,
					"code":                "REMOTE001",
					"name":                "Vật tư từ API",
					"purchase_uom_name":   "Cái",
					"original_price":      12500,
					"manufacture_name":    "Nhà sản xuất/Nước sản xuất",
					"so_luong_ton_dau_ky": 10,
					"sl_nhap":             3,
					"sl_xuat":             2,
				},
			}})
		case "/partner_select_for_po":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{
				{
					"id":            "PARTNER-TAX",
					"name":          "Nhà cung cấp có mã số thuế",
					"tax_code":      "0101234567",
					"bank_account":  "00112233",
					"address":       "Hà Nội",
					"contact_email": "contact@example.test",
				},
				{
					"id":       "PARTNER-ID",
					"name":     "Nhà cung cấp chưa có mã số thuế",
					"tax_code": nil,
				},
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	supplyRepo := &captureInternalSupplyRepository{}
	contactRepo := &captureCompanyContactRepository{}
	service := NewInternalSupplySyncService(&config.Config{
		InternalSupplyAPIURL:            server.URL + "/",
		InternalSupplyAPIToken:          "Bearer test-token",
		InternalSupplyAPIBody:           "{}",
		InternalSupplyAPITimeoutSeconds: 5,
		InternalSupplySyncTimezone:      "UTC",
		SupplyMappingTable:              "mapping2",
	}, supplyRepo, contactRepo)

	count, err := service.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if count != 1 || len(supplyRepo.inputs) != 1 {
		t.Fatalf("synced count = %d, rows = %d; want only the one remote row", count, len(supplyRepo.inputs))
	}
	if supplyRepo.inputs[0].ID != "REMOTE001" {
		t.Fatalf("supply ID = %q", supplyRepo.inputs[0].ID)
	}
	if len(contactRepo.contacts) != 2 {
		t.Fatalf("company contact count = %d, want 2", len(contactRepo.contacts))
	}
	if contactRepo.contacts[0].MaSoThue != "0101234567" || contactRepo.contacts[0].Gmail != "contact@example.test" {
		t.Fatalf("tax-code partner mapping = %+v", contactRepo.contacts[0])
	}
	if contactRepo.contacts[1].MaSoThue != "PARTNER-ID" {
		t.Fatalf("fallback partner identity = %q", contactRepo.contacts[1].MaSoThue)
	}
}

func TestInternalSupplyMappingPrefersTypeNameAndFallsBackToID(t *testing.T) {
	t.Parallel()

	mappings := map[string]models.SupplyMapping{
		"TYPE-001_QD-01": {GroupName: "typename mapping"},
		"OLD-001_QD-01":  {GroupName: "legacy mapping"},
	}

	keys := uniqueNonEmptyMappingKeys("TYPE-001_QD-01", "OLD-001_QD-01")
	mapping, found := firstSupplyMapping(mappings, keys)
	if !found || mapping.GroupName != "typename mapping" {
		t.Fatalf("typename mapping = (%+v, %v), want typename mapping", mapping, found)
	}

	keys = uniqueNonEmptyMappingKeys("_QD-01", "OLD-001_QD-01")
	mapping, found = firstSupplyMapping(mappings, keys)
	if !found || mapping.GroupName != "legacy mapping" {
		t.Fatalf("legacy fallback mapping = (%+v, %v), want legacy mapping", mapping, found)
	}
}

type captureInternalSupplyRepository struct {
	inputs []models.SupplyUpsertInput
}

func (r *captureInternalSupplyRepository) ReplaceAll(inputs []models.SupplyUpsertInput) error {
	r.inputs = append([]models.SupplyUpsertInput(nil), inputs...)
	return nil
}

func (r *captureInternalSupplyRepository) GetSupplyMappings(string) (map[string]models.SupplyMapping, error) {
	return nil, nil
}

type captureCompanyContactRepository struct {
	contacts []models.CompanyContact
}

func (r *captureCompanyContactRepository) ReplaceAll(contacts []models.CompanyContact) error {
	r.contacts = append([]models.CompanyContact(nil), contacts...)
	return nil
}
