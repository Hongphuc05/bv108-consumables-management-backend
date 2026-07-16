package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"bv108-consumables-management-backend/internal/models"

	"golang.org/x/text/unicode/norm"
)

const (
	defaultVinmesCatalogTimeout = 60 * time.Second
)

var vinmesTenderCodePattern = regexp.MustCompile(`\b(2233|4418|7313|9528|9530|9532|9534)\b`)

type VinmesCatalogConfig struct {
	APIBaseURL     string
	APIToken       string
	TimeoutSeconds int
	CatalogStore   VinmesCatalogStore
}

type VinmesCatalogStore interface {
	ReplaceAll(items []models.VinmesCatalogItem, syncedAt time.Time) error
	ListAll() ([]models.VinmesCatalogItem, error)
}

type VinmesCatalogService struct {
	apiBaseURL   string
	apiToken     string
	httpClient   *http.Client
	catalogStore VinmesCatalogStore
}

type vinmesStorage struct {
	ID   int64  `json:"msl_storage_id"`
	Name string `json:"msl_name"`
}

type vinmesPartner struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	TaxCode     *string `json:"tax_code"`
	BankAccount *string `json:"bank_account"`
}

type vinmesResource struct {
	ID   int64  `json:"mpr_product_resource_id"`
	Name string `json:"mpr_name"`
}

type vinmesTax struct {
	ID   int64   `json:"adt_taxrate_id"`
	Rate float64 `json:"adt_rate"`
}

type vinmesContractPackage struct {
	ID          string `json:"adcp_contract_package_id"`
	Description string `json:"adcp_description"`
}

type vinmesProduct struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type vinmesStringID string

func (id *vinmesStringID) UnmarshalJSON(data []byte) error {
	var stringValue string
	if err := json.Unmarshal(data, &stringValue); err == nil {
		*id = vinmesStringID(stringValue)
		return nil
	}

	var numberValue json.Number
	if err := json.Unmarshal(data, &numberValue); err != nil {
		return fmt.Errorf("expected string or number ID: %w", err)
	}
	*id = vinmesStringID(numberValue.String())
	return nil
}

type vinmesContract struct {
	ID vinmesStringID `json:"adc_contract_id"`
	No string         `json:"adc_contract_no"`
}

type vinmesCatalogs struct {
	Storages         []vinmesStorage
	StoragePayloads  []json.RawMessage
	Partners         []vinmesPartner
	PartnerPayloads  []json.RawMessage
	Resources        []vinmesResource
	ResourcePayloads []json.RawMessage
	Taxes            []vinmesTax
	TaxPayloads      []json.RawMessage
	ContractPackages []vinmesContractPackage
	PackagePayloads  []json.RawMessage
	Products         []vinmesProduct
	ProductPayloads  []json.RawMessage
	Contracts        []vinmesContract
	ContractPayloads []json.RawMessage
}

type VinmesCatalogRefreshResult struct {
	Total    int            `json:"total"`
	Counts   map[string]int `json:"counts"`
	SyncedAt time.Time      `json:"syncedAt"`
}

type VinmesC10Options struct {
	DML bool `json:"dml"`
}

type VinmesC10MasterBinds struct {
	UserID            string  `json:"p_user_id"`
	StorageID         *int64  `json:"p_storage_id"`
	ResourceID        *int64  `json:"p_resource_id"`
	PartnerID         *string `json:"p_partner_id"`
	InvoiceType       string  `json:"p_invoicetype"`
	TaxID             *int64  `json:"p_tax_id"`
	KyHieu            string  `json:"p_kyhieu"`
	InvoiceNo         string  `json:"p_invoiceno"`
	ContractPackageID *int64  `json:"p_contractpkg_id"`
	ContractID        *string `json:"p_contract_id"`
	OrderDate         string  `json:"p_orderdate"`
	InvoiceDate       string  `json:"p_invoicedate"`
	Description       string  `json:"p_description"`
}

type VinmesC10MasterRequest struct {
	Options VinmesC10Options     `json:"options"`
	Binds   VinmesC10MasterBinds `json:"binds"`
}

type VinmesC10DetailBinds struct {
	PurchaseOrderID *int64  `json:"p_po_id"`
	UserID          string  `json:"p_user_id"`
	ProductID       *int64  `json:"p_product_id"`
	Quantity        float64 `json:"p_qtyorder"`
	LotNo           *string `json:"p_lotno"`
	ExpiryDate      *string `json:"p_expdate"`
}

type VinmesC10DetailRequest struct {
	Options VinmesC10Options     `json:"options"`
	Binds   VinmesC10DetailBinds `json:"binds"`
}

type VinmesMappingValidationError struct {
	Field       string `json:"field"`
	SourceValue string `json:"sourceValue"`
	Message     string `json:"message"`
}

type VinmesMappingSource struct {
	SoPhieu                string  `json:"soPhieu"`
	SoHoaDon               string  `json:"soHoaDon"`
	NhaCungCap             string  `json:"nhaCungCap"`
	MaSoThueNhaCungCap     string  `json:"maSoThueNhaCungCap"`
	SoTKNganHangNhaCungCap string  `json:"soTkNganHangNhaCungCap"`
	ReconciliationIDs      []int64 `json:"reconciliationIds"`
}

type VinmesMappedPurchaseOrder struct {
	Master             VinmesC10MasterRequest         `json:"master"`
	Details            []VinmesC10DetailRequest       `json:"details"`
	Source             VinmesMappingSource            `json:"source"`
	PartnerMatchMethod string                         `json:"partnerMatchMethod,omitempty"`
	ValidationErrors   []VinmesMappingValidationError `json:"validationErrors"`
}

func NewVinmesCatalogService(cfg VinmesCatalogConfig) *VinmesCatalogService {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = defaultVinmesCatalogTimeout
	}

	return &VinmesCatalogService{
		apiBaseURL:   strings.TrimRight(strings.TrimSpace(cfg.APIBaseURL), "/"),
		apiToken:     strings.TrimSpace(cfg.APIToken),
		httpClient:   &http.Client{Timeout: timeout},
		catalogStore: cfg.CatalogStore,
	}
}

func (s *VinmesCatalogService) IsConfigured() bool {
	return s != nil && s.apiBaseURL != ""
}

func (s *VinmesCatalogService) BuildMappingPreview(ctx context.Context, masters []models.VinmesExportMaster) ([]VinmesMappedPurchaseOrder, error) {
	if !s.IsConfigured() {
		return nil, fmt.Errorf("Vinmes catalog API is not configured")
	}

	catalogs, err := s.loadCatalogs(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]VinmesMappedPurchaseOrder, 0, len(masters))
	for _, master := range masters {
		result = append(result, mapVinmesMaster(master, catalogs))
	}
	return result, nil
}

func (s *VinmesCatalogService) loadCatalogs(ctx context.Context) (*vinmesCatalogs, error) {
	if s.catalogStore != nil {
		items, err := s.catalogStore.ListAll()
		if err != nil {
			return nil, err
		}
		if len(items) > 0 {
			return catalogsFromStoredItems(items)
		}
		_, catalogs, err := s.refreshCatalogs(ctx)
		return catalogs, err
	}
	return s.loadRemoteCatalogs(ctx)
}

func (s *VinmesCatalogService) RefreshCatalogs(ctx context.Context) (*VinmesCatalogRefreshResult, error) {
	if !s.IsConfigured() {
		return nil, fmt.Errorf("Vinmes catalog API is not configured")
	}
	result, _, err := s.refreshCatalogs(ctx)
	return result, err
}

func (s *VinmesCatalogService) refreshCatalogs(ctx context.Context) (*VinmesCatalogRefreshResult, *vinmesCatalogs, error) {
	catalogs, err := s.loadRemoteCatalogs(ctx)
	if err != nil {
		return nil, nil, err
	}
	syncedAt := time.Now()
	items, counts, err := storedItemsFromCatalogs(catalogs, syncedAt)
	if err != nil {
		return nil, nil, err
	}
	if s.catalogStore != nil {
		if err := s.catalogStore.ReplaceAll(items, syncedAt); err != nil {
			return nil, nil, err
		}
	}
	return &VinmesCatalogRefreshResult{Total: len(items), Counts: counts, SyncedAt: syncedAt}, catalogs, nil
}

func (s *VinmesCatalogService) loadRemoteCatalogs(ctx context.Context) (*vinmesCatalogs, error) {
	catalogs := &vinmesCatalogs{}
	tasks := []struct {
		resource string
		target   any
		payloads *[]json.RawMessage
	}{
		{resource: "storage_select_for_po", target: &catalogs.Storages, payloads: &catalogs.StoragePayloads},
		{resource: "partner_select_for_po", target: &catalogs.Partners, payloads: &catalogs.PartnerPayloads},
		{resource: "resource_select_for_po", target: &catalogs.Resources, payloads: &catalogs.ResourcePayloads},
		{resource: "tax_select_for_po", target: &catalogs.Taxes, payloads: &catalogs.TaxPayloads},
		{resource: "contractpkg_select_for_po", target: &catalogs.ContractPackages, payloads: &catalogs.PackagePayloads},
		{resource: "product_select_for_po", target: &catalogs.Products, payloads: &catalogs.ProductPayloads},
		{resource: "contract_select_for_po", target: &catalogs.Contracts, payloads: &catalogs.ContractPayloads},
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(tasks))
	for _, task := range tasks {
		task := task
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.fetchCatalog(ctx, task.resource, task.target, task.payloads); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	if err := <-errCh; err != nil {
		return nil, err
	}
	if err := validateRequiredVinmesCatalogs(catalogs); err != nil {
		return nil, err
	}

	return catalogs, nil
}

func validateRequiredVinmesCatalogs(catalogs *vinmesCatalogs) error {
	if catalogs == nil {
		return fmt.Errorf("Vinmes catalog response is empty")
	}

	required := []struct {
		resource string
		count    int
	}{
		{resource: "storage_select_for_po", count: len(catalogs.Storages)},
		{resource: "partner_select_for_po", count: len(catalogs.Partners)},
		{resource: "resource_select_for_po", count: len(catalogs.Resources)},
		{resource: "tax_select_for_po", count: len(catalogs.Taxes)},
		{resource: "contractpkg_select_for_po", count: len(catalogs.ContractPackages)},
		{resource: "product_select_for_po", count: len(catalogs.Products)},
	}
	for _, catalog := range required {
		if catalog.count == 0 {
			return fmt.Errorf("Vinmes %s returned an empty required catalog", catalog.resource)
		}
	}
	return nil
}

func (s *VinmesCatalogService) fetchCatalog(ctx context.Context, resource string, target any, payloads *[]json.RawMessage) error {
	endpoint := fmt.Sprintf("%s/%s?method=select", s.apiBaseURL, url.PathEscape(resource))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, http.NoBody)
	if err != nil {
		return fmt.Errorf("create Vinmes %s request: %w", resource, err)
	}
	if s.apiToken != "" {
		req.Header.Set("Authorization", bearerAuthorization(s.apiToken))
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request Vinmes %s: %w", resource, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("Vinmes %s returned HTTP %d: %s", resource, resp.StatusCode, strings.TrimSpace(string(message)))
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("decode Vinmes %s response: %w", resource, err)
	}
	if len(envelope.Data) == 0 {
		return fmt.Errorf("Vinmes %s response has no data field", resource)
	}
	if err := json.Unmarshal(envelope.Data, target); err != nil {
		return fmt.Errorf("decode Vinmes %s catalog: %w", resource, err)
	}
	if err := json.Unmarshal(envelope.Data, payloads); err != nil {
		return fmt.Errorf("preserve Vinmes %s catalog payloads: %w", resource, err)
	}
	return nil
}

func storedItemsFromCatalogs(catalogs *vinmesCatalogs, syncedAt time.Time) ([]models.VinmesCatalogItem, map[string]int, error) {
	items := make([]models.VinmesCatalogItem, 0,
		len(catalogs.Storages)+len(catalogs.Partners)+len(catalogs.Resources)+
			len(catalogs.Taxes)+len(catalogs.ContractPackages)+len(catalogs.Contracts)+len(catalogs.Products),
	)
	counts := make(map[string]int)

	appendItem := func(catalogType, externalID, code, name, taxCode, bankAccount string, taxRate *float64, raw json.RawMessage, value any) error {
		if len(raw) == 0 {
			encoded, err := json.Marshal(value)
			if err != nil {
				return fmt.Errorf("encode Vinmes %s item %s: %w", catalogType, externalID, err)
			}
			raw = encoded
		}
		items = append(items, models.VinmesCatalogItem{
			CatalogType: catalogType,
			ExternalID:  externalID,
			Code:        code,
			Name:        name,
			TaxCode:     taxCode,
			BankAccount: bankAccount,
			TaxRate:     taxRate,
			RawPayload:  string(raw),
			SyncedAt:    syncedAt,
		})
		counts[catalogType]++
		return nil
	}

	for index, item := range catalogs.Storages {
		if err := appendItem("storage", strconv.FormatInt(item.ID, 10), "", item.Name, "", "", nil, payloadAt(catalogs.StoragePayloads, index), item); err != nil {
			return nil, nil, err
		}
	}
	for index, item := range catalogs.Partners {
		taxCode := ""
		if item.TaxCode != nil {
			taxCode = strings.TrimSpace(*item.TaxCode)
		}
		bankAccount := ""
		if item.BankAccount != nil {
			bankAccount = strings.TrimSpace(*item.BankAccount)
		}
		if err := appendItem("partner", item.ID, "", item.Name, taxCode, bankAccount, nil, payloadAt(catalogs.PartnerPayloads, index), item); err != nil {
			return nil, nil, err
		}
	}
	for index, item := range catalogs.Resources {
		if err := appendItem("resource", strconv.FormatInt(item.ID, 10), "", item.Name, "", "", nil, payloadAt(catalogs.ResourcePayloads, index), item); err != nil {
			return nil, nil, err
		}
	}
	for index, item := range catalogs.Taxes {
		rate := item.Rate
		if err := appendItem("tax", strconv.FormatInt(item.ID, 10), strconv.FormatFloat(item.Rate, 'f', -1, 64), "", "", "", &rate, payloadAt(catalogs.TaxPayloads, index), item); err != nil {
			return nil, nil, err
		}
	}
	for index, item := range catalogs.ContractPackages {
		if err := appendItem("contract_package", item.ID, item.ID, item.Description, "", "", nil, payloadAt(catalogs.PackagePayloads, index), item); err != nil {
			return nil, nil, err
		}
	}
	for index, item := range catalogs.Contracts {
		if err := appendItem("contract", string(item.ID), item.No, item.No, "", "", nil, payloadAt(catalogs.ContractPayloads, index), item); err != nil {
			return nil, nil, err
		}
	}
	for index, item := range catalogs.Products {
		if err := appendItem("product", strconv.FormatInt(item.ID, 10), item.Code, item.Name, "", "", nil, payloadAt(catalogs.ProductPayloads, index), item); err != nil {
			return nil, nil, err
		}
	}

	return items, counts, nil
}

func payloadAt(payloads []json.RawMessage, index int) json.RawMessage {
	if index < 0 || index >= len(payloads) {
		return nil
	}
	return payloads[index]
}

func catalogsFromStoredItems(items []models.VinmesCatalogItem) (*vinmesCatalogs, error) {
	catalogs := &vinmesCatalogs{}
	for _, item := range items {
		var target any
		switch item.CatalogType {
		case "storage":
			catalogs.Storages = append(catalogs.Storages, vinmesStorage{})
			target = &catalogs.Storages[len(catalogs.Storages)-1]
		case "partner":
			catalogs.Partners = append(catalogs.Partners, vinmesPartner{})
			target = &catalogs.Partners[len(catalogs.Partners)-1]
		case "resource":
			catalogs.Resources = append(catalogs.Resources, vinmesResource{})
			target = &catalogs.Resources[len(catalogs.Resources)-1]
		case "tax":
			catalogs.Taxes = append(catalogs.Taxes, vinmesTax{})
			target = &catalogs.Taxes[len(catalogs.Taxes)-1]
		case "contract_package":
			catalogs.ContractPackages = append(catalogs.ContractPackages, vinmesContractPackage{})
			target = &catalogs.ContractPackages[len(catalogs.ContractPackages)-1]
		case "contract":
			catalogs.Contracts = append(catalogs.Contracts, vinmesContract{})
			target = &catalogs.Contracts[len(catalogs.Contracts)-1]
		case "product":
			catalogs.Products = append(catalogs.Products, vinmesProduct{})
			target = &catalogs.Products[len(catalogs.Products)-1]
		default:
			continue
		}
		if err := json.Unmarshal([]byte(item.RawPayload), target); err != nil {
			return nil, fmt.Errorf("decode stored Vinmes %s item %s: %w", item.CatalogType, item.ExternalID, err)
		}
	}
	return catalogs, nil
}

func bearerAuthorization(token string) string {
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		return token
	}
	return "Bearer " + token
}

func mapVinmesMaster(master models.VinmesExportMaster, catalogs *vinmesCatalogs) VinmesMappedPurchaseOrder {
	mapped := VinmesMappedPurchaseOrder{
		Master: VinmesC10MasterRequest{
			Options: VinmesC10Options{DML: true},
			Binds: VinmesC10MasterBinds{
				UserID:      master.UserID,
				InvoiceType: "P",
				KyHieu:      strings.TrimSpace(master.KyHieu),
				InvoiceNo:   strings.TrimSpace(master.SoHoaDon),
				OrderDate:   convertVinmesDate(master.NgayYeuCau),
				InvoiceDate: convertVinmesDate(master.NgayHoaDon),
				Description: "Số phiếu BV108: " + strings.TrimSpace(master.SoPhieu),
			},
		},
		Details:          make([]VinmesC10DetailRequest, 0, len(master.Details)),
		ValidationErrors: make([]VinmesMappingValidationError, 0),
		Source: VinmesMappingSource{
			SoPhieu:                master.SoPhieu,
			SoHoaDon:               master.SoHoaDon,
			NhaCungCap:             master.NhaCungCap,
			MaSoThueNhaCungCap:     master.MaSoThueNhaCungCap,
			SoTKNganHangNhaCungCap: master.SoTKNganHangNhaCungCap,
			ReconciliationIDs:      make([]int64, 0, len(master.Details)),
		},
	}

	mapStorage(master, catalogs, &mapped)
	mapResource(master, catalogs, &mapped)
	mapPartner(master, catalogs, &mapped)
	mapTax(master, catalogs, &mapped)
	mapContractPackage(master, catalogs, &mapped)
	validateMasterFields(master, &mapped)

	productsByCode := make(map[string][]vinmesProduct, len(catalogs.Products))
	for _, product := range catalogs.Products {
		key := normalizeCode(product.Code)
		if key != "" {
			productsByCode[key] = append(productsByCode[key], product)
		}
	}
	for index, detail := range master.Details {
		mapped.Source.ReconciliationIDs = append(mapped.Source.ReconciliationIDs, detail.ReconciliationID)
		request := VinmesC10DetailRequest{
			Options: VinmesC10Options{DML: true},
			Binds: VinmesC10DetailBinds{
				UserID:   master.UserID,
				Quantity: detail.SoLuong,
			},
		}
		matches := productsByCode[normalizeCode(detail.MaHang)]
		switch len(matches) {
		case 1:
			request.Binds.ProductID = int64Pointer(matches[0].ID)
		case 0:
			mapped.addError(fmt.Sprintf("details[%d].p_product_id", index), detail.MaHang, "Không tìm thấy mã hàng trong danh mục Vinmes")
		default:
			mapped.addError(fmt.Sprintf("details[%d].p_product_id", index), detail.MaHang, "Mã hàng khớp nhiều sản phẩm Vinmes")
		}
		if detail.SoLuong <= 0 {
			mapped.addError(fmt.Sprintf("details[%d].p_qtyorder", index), strconv.FormatFloat(detail.SoLuong, 'f', -1, 64), "Số lượng phải lớn hơn 0")
		}
		mapped.Details = append(mapped.Details, request)
	}
	if len(master.Details) == 0 {
		mapped.addError("details", "", "Đơn hàng không có dòng chi tiết")
	}

	return mapped
}

func mapStorage(master models.VinmesExportMaster, catalogs *vinmesCatalogs, mapped *VinmesMappedPurchaseOrder) {
	matches := make([]vinmesStorage, 0, 1)
	key := normalizeLookup(master.KhoHang)
	for _, item := range catalogs.Storages {
		if normalizeLookup(item.Name) == key {
			matches = append(matches, item)
		}
	}
	if len(matches) == 1 {
		mapped.Master.Binds.StorageID = int64Pointer(matches[0].ID)
		return
	}
	message := "Không tìm thấy kho trong danh mục Vinmes"
	if len(matches) > 1 {
		message = "Tên kho khớp nhiều kho Vinmes"
	}
	mapped.addError("p_storage_id", master.KhoHang, message)
}

func mapResource(master models.VinmesExportMaster, catalogs *vinmesCatalogs, mapped *VinmesMappedPurchaseOrder) {
	matches := make([]vinmesResource, 0, 1)
	key := normalizeLookup(master.Nguon)
	for _, item := range catalogs.Resources {
		if normalizeLookup(item.Name) == key {
			matches = append(matches, item)
		}
	}
	if len(matches) == 1 {
		mapped.Master.Binds.ResourceID = int64Pointer(matches[0].ID)
		return
	}
	message := "Không tìm thấy nguồn trong danh mục Vinmes"
	if len(matches) > 1 {
		message = "Tên nguồn khớp nhiều nguồn Vinmes"
	}
	mapped.addError("p_resource_id", master.Nguon, message)
}

func mapPartner(master models.VinmesExportMaster, catalogs *vinmesCatalogs, mapped *VinmesMappedPurchaseOrder) {
	taxCode := normalizeCode(master.MaSoThueNhaCungCap)
	if taxCode != "" {
		matches := make([]vinmesPartner, 0, 1)
		for _, partner := range catalogs.Partners {
			if partner.TaxCode != nil && normalizeCode(*partner.TaxCode) == taxCode {
				matches = append(matches, partner)
			}
		}
		if len(matches) == 1 {
			mapped.Master.Binds.PartnerID = stringPointer(matches[0].ID)
			mapped.PartnerMatchMethod = "tax_code"
			return
		}
		if len(matches) > 1 {
			mapped.addError("p_partner_id", master.MaSoThueNhaCungCap, "Mã số thuế khớp nhiều nhà cung cấp Vinmes")
			return
		}
	}

	bankAccount := normalizeCode(master.SoTKNganHangNhaCungCap)
	if bankAccount != "" {
		matches := make([]vinmesPartner, 0, 1)
		for _, partner := range catalogs.Partners {
			if partner.BankAccount != nil && normalizeCode(*partner.BankAccount) == bankAccount {
				matches = append(matches, partner)
			}
		}
		if len(matches) == 1 {
			mapped.Master.Binds.PartnerID = stringPointer(matches[0].ID)
			mapped.PartnerMatchMethod = "bank_account"
			return
		}
		if len(matches) > 1 {
			mapped.addError("p_partner_id", master.SoTKNganHangNhaCungCap, "Số tài khoản khớp nhiều nhà cung cấp Vinmes")
			return
		}
	}

	nameKey := normalizeLookup(master.NhaCungCap)
	matches := make([]vinmesPartner, 0, 1)
	for _, partner := range catalogs.Partners {
		if normalizeLookup(partner.Name) == nameKey {
			matches = append(matches, partner)
		}
	}
	if len(matches) == 1 {
		mapped.Master.Binds.PartnerID = stringPointer(matches[0].ID)
		mapped.PartnerMatchMethod = "normalized_name"
		return
	}
	message := "Không tìm thấy nhà cung cấp theo mã số thuế hoặc tên"
	if len(matches) > 1 {
		message = "Tên nhà cung cấp khớp nhiều đối tác Vinmes; cần mapping thủ công"
	}
	mapped.addError("p_partner_id", master.NhaCungCap, message)
}

func mapTax(master models.VinmesExportMaster, catalogs *vinmesCatalogs, mapped *VinmesMappedPurchaseOrder) {
	rate, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(master.Thue, "%")), 64)
	if err != nil {
		mapped.addError("p_tax_id", master.Thue, "Thuế suất không đúng định dạng")
		return
	}
	matches := make([]vinmesTax, 0, 1)
	for _, item := range catalogs.Taxes {
		if item.Rate == rate {
			matches = append(matches, item)
		}
	}
	if len(matches) == 1 {
		mapped.Master.Binds.TaxID = int64Pointer(matches[0].ID)
		return
	}
	message := "Không tìm thấy thuế suất trong danh mục Vinmes"
	if len(matches) > 1 {
		message = "Thuế suất khớp nhiều mã thuế Vinmes"
	}
	mapped.addError("p_tax_id", master.Thue, message)
}

func mapContractPackage(master models.VinmesExportMaster, catalogs *vinmesCatalogs, mapped *VinmesMappedPurchaseOrder) {
	key := normalizeLookup(master.GoiThau)
	tenderCode := ""
	if match := vinmesTenderCodePattern.FindString(master.GoiThau); match != "" {
		tenderCode = match
	}
	matches := make([]vinmesContractPackage, 0, 1)
	for _, item := range catalogs.ContractPackages {
		text := item.ID + " " + item.Description
		if (tenderCode != "" && strings.Contains(text, tenderCode)) || (tenderCode == "" && normalizeLookup(text) == key) {
			matches = append(matches, item)
		}
	}
	if len(matches) == 1 {
		packageID, err := strconv.ParseInt(strings.TrimSpace(matches[0].ID), 10, 64)
		if err == nil {
			mapped.Master.Binds.ContractPackageID = int64Pointer(packageID)
			return
		}
		mapped.addError("p_contractpkg_id", master.GoiThau, "ID gói thầu Vinmes không phải dạng số C10 yêu cầu; không tự suy đoán ID từ số quyết định")
		return
	}
	message := "Không tìm thấy gói thầu trong danh mục Vinmes"
	if len(matches) > 1 {
		message = "Gói thầu khớp nhiều bản ghi Vinmes"
	}
	mapped.addError("p_contractpkg_id", master.GoiThau, message)
}

func validateMasterFields(master models.VinmesExportMaster, mapped *VinmesMappedPurchaseOrder) {
	if strings.TrimSpace(master.KyHieu) == "" || master.KyHieu == "không hiểu" {
		mapped.addError("p_kyhieu", master.KyHieu, "Thiếu ký hiệu hóa đơn")
	}
	if strings.TrimSpace(master.SoHoaDon) == "" {
		mapped.addError("p_invoiceno", master.SoHoaDon, "Thiếu số hóa đơn")
	}
	if mapped.Master.Binds.OrderDate == "" {
		mapped.addError("p_orderdate", master.NgayYeuCau, "Ngày yêu cầu không đúng định dạng DD/MM/YYYY")
	}
	if mapped.Master.Binds.InvoiceDate == "" {
		mapped.addError("p_invoicedate", master.NgayHoaDon, "Ngày hóa đơn không đúng định dạng DD/MM/YYYY")
	}
}

func (m *VinmesMappedPurchaseOrder) addError(field, sourceValue, message string) {
	m.ValidationErrors = append(m.ValidationErrors, VinmesMappingValidationError{
		Field:       field,
		SourceValue: strings.TrimSpace(sourceValue),
		Message:     message,
	})
}

func convertVinmesDate(value string) string {
	parsed, err := time.Parse("02/01/2006", strings.TrimSpace(value))
	if err != nil {
		return ""
	}
	return parsed.Format("2006-01-02")
}

func normalizeCode(value string) string {
	var builder strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func normalizeLookup(value string) string {
	decomposed := norm.NFD.String(strings.ToLower(strings.TrimSpace(value)))
	var builder strings.Builder
	lastSpace := true
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if r == 'đ' {
			r = 'd'
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace {
			builder.WriteByte(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(builder.String())
}

func int64Pointer(value int64) *int64 {
	return &value
}

func stringPointer(value string) *string {
	return &value
}
