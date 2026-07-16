package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bv108-consumables-management-backend/internal/models"
)

func TestVinmesMappingPreviewFallsBackToNormalizedPartnerName(t *testing.T) {
	t.Parallel()

	server := newVinmesCatalogTestServer(t)
	defer server.Close()

	service := NewVinmesCatalogService(VinmesCatalogConfig{
		APIBaseURL: server.URL,
		APIToken:   "test-token",
	})
	preview, err := service.BuildMappingPreview(context.Background(), []models.VinmesExportMaster{
		{
			UserID:             "trangbi",
			GoiThau:            "9530/QĐ-BV",
			KhoHang:            "Kho vật tư tiêu hao",
			Nguon:              "Mua",
			NhaCungCap:         "CÔNG TY TNHH ĐẦU TƯ VÀ PHÁT TRIỂN TBYT TRƯỜNG TIỀN",
			MaSoThueNhaCungCap: "0109999999",
			SoPhieu:            "PN20260707000088",
			KyHieu:             "1C26TKK",
			SoHoaDon:           "00000315",
			NgayYeuCau:         "07/07/2026",
			NgayHoaDon:         "06/07/2026",
			Thue:               "0%",
			Details: []models.VinmesExportDetail{
				{MaHang: "B001142.1", SoLuong: 15, ReconciliationID: 88},
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildMappingPreview() error = %v", err)
	}
	if len(preview) != 1 {
		t.Fatalf("expected 1 preview item, got %d", len(preview))
	}

	item := preview[0]
	if item.PartnerMatchMethod != "normalized_name" {
		t.Fatalf("partnerMatchMethod = %q", item.PartnerMatchMethod)
	}
	if item.Master.Binds.PartnerID == nil || *item.Master.Binds.PartnerID != "TB.TRGTIEN" {
		t.Fatalf("partner ID = %v", item.Master.Binds.PartnerID)
	}
	if item.Master.Binds.StorageID == nil || *item.Master.Binds.StorageID != 5 {
		t.Fatalf("storage ID = %v", item.Master.Binds.StorageID)
	}
	if item.Master.Binds.ResourceID == nil || *item.Master.Binds.ResourceID != 1 {
		t.Fatalf("resource ID = %v", item.Master.Binds.ResourceID)
	}
	if item.Master.Binds.TaxID == nil || *item.Master.Binds.TaxID != 0 {
		t.Fatalf("tax ID = %v", item.Master.Binds.TaxID)
	}
	if item.Master.Binds.ContractPackageID == nil || *item.Master.Binds.ContractPackageID != 123 {
		t.Fatalf("contract package ID = %v", item.Master.Binds.ContractPackageID)
	}
	if len(item.Details) != 1 || item.Details[0].Binds.ProductID == nil || *item.Details[0].Binds.ProductID != 12345 {
		t.Fatalf("product mapping = %+v", item.Details)
	}
	if item.Master.Binds.OrderDate != "2026-07-07" || item.Master.Binds.InvoiceDate != "2026-07-06" {
		t.Fatalf("mapped dates = %q, %q", item.Master.Binds.OrderDate, item.Master.Binds.InvoiceDate)
	}
	if len(item.ValidationErrors) != 0 {
		t.Fatalf("unexpected validation errors: %+v", item.ValidationErrors)
	}
}

func TestVinmesMappingPreviewPrefersTaxCodeAndReportsMissingProduct(t *testing.T) {
	t.Parallel()

	server := newVinmesCatalogTestServer(t)
	defer server.Close()
	service := NewVinmesCatalogService(VinmesCatalogConfig{APIBaseURL: server.URL, APIToken: "test-token"})

	preview, err := service.BuildMappingPreview(context.Background(), []models.VinmesExportMaster{
		{
			UserID:             "trangbi",
			GoiThau:            "9530/QĐ-BV",
			KhoHang:            "Kho vật tư tiêu hao",
			Nguon:              "Mua",
			NhaCungCap:         "Tên không giống Vinmes",
			MaSoThueNhaCungCap: "0101234567",
			KyHieu:             "1C26ABC",
			SoHoaDon:           "0001",
			NgayYeuCau:         "07/07/2026",
			NgayHoaDon:         "07/07/2026",
			Thue:               "0%",
			Details:            []models.VinmesExportDetail{{MaHang: "NOT-FOUND", SoLuong: 1}},
		},
	})
	if err != nil {
		t.Fatalf("BuildMappingPreview() error = %v", err)
	}
	item := preview[0]
	if item.PartnerMatchMethod != "tax_code" {
		t.Fatalf("partnerMatchMethod = %q", item.PartnerMatchMethod)
	}
	if item.Master.Binds.PartnerID == nil || *item.Master.Binds.PartnerID != "PARTNER-TAX" {
		t.Fatalf("partner ID = %v", item.Master.Binds.PartnerID)
	}
	if len(item.ValidationErrors) != 1 || item.ValidationErrors[0].Field != "details[0].p_product_id" {
		t.Fatalf("validation errors = %+v", item.ValidationErrors)
	}
}

func TestVinmesMappingPreviewUsesBankAccountBeforeName(t *testing.T) {
	t.Parallel()

	server := newVinmesCatalogTestServer(t)
	defer server.Close()
	service := NewVinmesCatalogService(VinmesCatalogConfig{APIBaseURL: server.URL, APIToken: "test-token"})

	preview, err := service.BuildMappingPreview(context.Background(), []models.VinmesExportMaster{
		{
			UserID:                 "trangbi",
			GoiThau:                "9530/QĐ-BV",
			KhoHang:                "Kho vật tư tiêu hao",
			Nguon:                  "Mua",
			NhaCungCap:             "Tên không giống Vinmes",
			MaSoThueNhaCungCap:     "MST-KHONG-KHOP",
			SoTKNganHangNhaCungCap: "0031-10133 7009",
			KyHieu:                 "1C26ABC",
			SoHoaDon:               "0002",
			NgayYeuCau:             "07/07/2026",
			NgayHoaDon:             "07/07/2026",
			Thue:                   "0%",
			Details:                []models.VinmesExportDetail{{MaHang: "B001142.1", SoLuong: 1}},
		},
	})
	if err != nil {
		t.Fatalf("BuildMappingPreview() error = %v", err)
	}
	item := preview[0]
	if item.PartnerMatchMethod != "bank_account" {
		t.Fatalf("partnerMatchMethod = %q", item.PartnerMatchMethod)
	}
	if item.Master.Binds.PartnerID == nil || *item.Master.Binds.PartnerID != "TB.TRGTIEN" {
		t.Fatalf("partner ID = %v", item.Master.Binds.PartnerID)
	}
	if len(item.ValidationErrors) != 0 {
		t.Fatalf("unexpected validation errors: %+v", item.ValidationErrors)
	}
}

func TestMapContractPackageUsesCatalogIDInsteadOfTenderReference(t *testing.T) {
	t.Parallel()

	mapped := VinmesMappedPurchaseOrder{}
	mapContractPackage(
		models.VinmesExportMaster{GoiThau: "4418/QĐ-BV"},
		&vinmesCatalogs{ContractPackages: []vinmesContractPackage{{ID: "123", Description: "Quyết định 4418/QĐ-BV"}}},
		&mapped,
	)
	if mapped.Master.Binds.ContractPackageID == nil || *mapped.Master.Binds.ContractPackageID != 123 {
		t.Fatalf("contract package ID = %v", mapped.Master.Binds.ContractPackageID)
	}
	if len(mapped.ValidationErrors) != 0 {
		t.Fatalf("validation errors = %+v", mapped.ValidationErrors)
	}
}

func TestMapContractPackageRejectsNonnumericCatalogID(t *testing.T) {
	t.Parallel()

	mapped := VinmesMappedPurchaseOrder{}
	mapContractPackage(
		models.VinmesExportMaster{GoiThau: "4418/QĐ-BV"},
		&vinmesCatalogs{ContractPackages: []vinmesContractPackage{{ID: "QĐBV4418", Description: "Quyết định 4418/QĐ-BV"}}},
		&mapped,
	)
	if mapped.Master.Binds.ContractPackageID != nil {
		t.Fatalf("contract package ID = %v, want nil", mapped.Master.Binds.ContractPackageID)
	}
	if len(mapped.ValidationErrors) != 1 || mapped.ValidationErrors[0].Field != "p_contractpkg_id" {
		t.Fatalf("validation errors = %+v", mapped.ValidationErrors)
	}
}

func TestValidateRequiredVinmesCatalogsRejectsMissingRequiredData(t *testing.T) {
	t.Parallel()

	catalogs := &vinmesCatalogs{
		Storages:         []vinmesStorage{{ID: 5}},
		Partners:         []vinmesPartner{{ID: "partner"}},
		Resources:        []vinmesResource{{ID: 1}},
		Taxes:            []vinmesTax{{ID: 1}},
		ContractPackages: []vinmesContractPackage{{ID: "123"}},
		Products:         nil,
	}
	if err := validateRequiredVinmesCatalogs(catalogs); err == nil || !strings.Contains(err.Error(), "product_select_for_po") {
		t.Fatalf("validateRequiredVinmesCatalogs() error = %v", err)
	}
}

func TestVinmesCatalogServiceAllowsNetworkOnlyConfiguration(t *testing.T) {
	t.Parallel()

	service := NewVinmesCatalogService(VinmesCatalogConfig{APIBaseURL: "http://vinmes.internal"})
	if !service.IsConfigured() {
		t.Fatal("expected API base URL without token to be configured")
	}
}

func TestVinmesCatalogServicePersistsAndReusesCatalogStore(t *testing.T) {
	t.Parallel()

	server := newVinmesCatalogTestServer(t)
	store := &memoryVinmesCatalogStore{}
	service := NewVinmesCatalogService(VinmesCatalogConfig{
		APIBaseURL:   server.URL,
		APIToken:     "test-token",
		CatalogStore: store,
	})

	result, err := service.RefreshCatalogs(context.Background())
	if err != nil {
		t.Fatalf("RefreshCatalogs() error = %v", err)
	}
	if result.Total != 8 {
		t.Fatalf("refresh total = %d, want 8", result.Total)
	}
	if len(store.items) != 8 || store.replaceCalls != 1 {
		t.Fatalf("stored items = %d, replace calls = %d", len(store.items), store.replaceCalls)
	}
	var storedPartner map[string]any
	for _, item := range store.items {
		if item.CatalogType == "partner" && item.ExternalID == "PARTNER-TAX" {
			if err := json.Unmarshal([]byte(item.RawPayload), &storedPartner); err != nil {
				t.Fatalf("decode stored partner payload: %v", err)
			}
		}
	}
	if storedPartner["address"] != "Địa chỉ chỉ có trong payload gốc" {
		t.Fatalf("stored partner payload did not preserve unknown fields: %#v", storedPartner)
	}

	server.Close()
	preview, err := service.BuildMappingPreview(context.Background(), []models.VinmesExportMaster{
		{
			UserID:     "trangbi",
			GoiThau:    "9530/QĐ-BV",
			KhoHang:    "Kho vật tư tiêu hao",
			Nguon:      "Mua",
			NhaCungCap: "Nhà cung cấp theo thuế",
			KyHieu:     "1C26ABC",
			SoHoaDon:   "0003",
			NgayYeuCau: "07/07/2026",
			NgayHoaDon: "07/07/2026",
			Thue:       "0%",
			Details:    []models.VinmesExportDetail{{MaHang: "B001142.1", SoLuong: 1}},
		},
	})
	if err != nil {
		t.Fatalf("BuildMappingPreview() from store error = %v", err)
	}
	if len(preview) != 1 {
		t.Fatalf("preview count = %d", len(preview))
	}
}

type memoryVinmesCatalogStore struct {
	items        []models.VinmesCatalogItem
	replaceCalls int
}

func (s *memoryVinmesCatalogStore) ReplaceAll(items []models.VinmesCatalogItem, _ time.Time) error {
	s.items = append([]models.VinmesCatalogItem(nil), items...)
	s.replaceCalls++
	return nil
}

func (s *memoryVinmesCatalogStore) ListAll() ([]models.VinmesCatalogItem, error) {
	return append([]models.VinmesCatalogItem(nil), s.items...), nil
}

func newVinmesCatalogTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	catalogs := map[string]any{
		"storage_select_for_po": []map[string]any{{"msl_storage_id": 5, "msl_name": "Kho vật tư tiêu hao"}},
		"partner_select_for_po": []map[string]any{
			{"id": "TB.TRGTIEN", "name": " Công ty  TNHH đầu tư và phát triển TBYT Trường Tiền ", "tax_code": nil, "bank_account": "0031 10133 7009"},
			{"id": "PARTNER-TAX", "name": "Nhà cung cấp theo thuế", "tax_code": "0101234567", "address": "Địa chỉ chỉ có trong payload gốc"},
		},
		"resource_select_for_po":    []map[string]any{{"mpr_product_resource_id": 1, "mpr_name": "Mua"}},
		"tax_select_for_po":         []map[string]any{{"adt_taxrate_id": 0, "adt_rate": 0}},
		"contractpkg_select_for_po": []map[string]any{{"adcp_contract_package_id": "123", "adcp_description": "Quyết định 9530"}},
		"contract_select_for_po":    []map[string]any{{"adc_contract_id": 101, "adc_contract_no": "HD-1"}},
		"product_select_for_po":     []map[string]any{{"id": 12345, "code": "B001142.1", "name": "Nẹp 2.0mm"}},
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Authorization header missing")
		}
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		resource := parts[len(parts)-1]
		data, ok := catalogs[resource]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"data": data}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
}
