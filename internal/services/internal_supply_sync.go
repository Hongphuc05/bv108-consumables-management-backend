package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"bv108-consumables-management-backend/config"
	"bv108-consumables-management-backend/internal/models"
)

type internalSupplySyncRepository interface {
	ReplaceAll(inputs []models.SupplyUpsertInput) error
}

type companyContactSyncRepository interface {
	ReplaceAll(contacts []models.CompanyContact) error
}

type InternalSupplySyncService struct {
	config      *config.Config
	repo        internalSupplySyncRepository
	contactRepo companyContactSyncRepository
	httpClient  *http.Client
	location    *time.Location
	mu          sync.Mutex
}

type internalSupplyAPIRow struct {
	Idx1            int     `json:"IDX1"`
	ProductID       int     `json:"PRODUCTID"`
	GroupName       string  `json:"GROUPNAME"`
	MaVtyt          string  `json:"MA_VTYT"`
	ID              string  `json:"ID"`
	Idx2            string  `json:"IDX2"`
	MaHieu          string  `json:"MA_HIEU"`
	TypeName        string  `json:"TYPENAME"`
	TenVatTuBV      string  `json:"TEN_VTYT_BV"`
	Name            string  `json:"NAME"`
	DonViTinh       string  `json:"DON_VI_TINH"`
	Unit            string  `json:"UNIT"`
	QuyCach         string  `json:"QUY_CACH"`
	QuyCachDongGoi  string  `json:"QUY_CACH_DONG_GOI"`
	QuyCachGiaoHang string  `json:"QUY_CACH_GIAO_HANG"`
	QuyCachToiThieu string  `json:"QUY_CACH_TOI_THIEU"`
	QuyetDinh       string  `json:"QUYET_DINH"`
	ThongTinThau    string  `json:"THONG_TIN_THAU"`
	TongThau        string  `json:"TONGTHAU"`
	HangSX          string  `json:"HANG_SX"`
	HangSXAlt       string  `json:"HANGSX"`
	NuocSX          string  `json:"NUOC_SX"`
	NuocSXAlt       string  `json:"NUOCSX"`
	NhaThau         string  `json:"NHA_THAU"`
	NhaCungCap      string  `json:"NHA_CUNG_CAP"`
	DonGia          float64 `json:"DON_GIA"`
	Price           float64 `json:"PRICE"`
	SoLuongTonKho   int     `json:"SO_LUONG_TON_KHO"`
	SlTon           int     `json:"SL_TON"`
	TonDauKy        int     `json:"TONDAUKY"`
	SlNhap          int     `json:"SL_NHAP"`
	NhapTrongKy     int     `json:"NHAPTRONGKY"`
	SlXuat          int     `json:"SL_XUAT"`
	XuatTrongKy     int     `json:"XUATTRONGKY"`
	TongNhap        int     `json:"TONGNHAP"`
	TonKhoMin       int     `json:"TON_KHO_MIN"`
	// New lowercase fields for product_select API compatibility
	NewID           int     `json:"id"`
	NewCode         string  `json:"code"`
	NewName         string  `json:"name"`
	PurchaseUomId   int     `json:"purchase_uom_id"`
	PurchaseUomName string  `json:"purchase_uom_name"`
	UomId           int     `json:"uom_id"`
	UomName         string  `json:"uom_name"`
	OriginalPrice   float64 `json:"original_price"`
	ManufactureName string  `json:"manufacture_name"`

	// New lowercase fields for api_trangbi_thongtinvattu compatibility
	DonGiaLower          float64 `json:"don_gia"`
	DonViTinhLower       string  `json:"don_vi_tinh"`
	HangSXLower          string  `json:"hang_sx"`
	MaHieuLower          string  `json:"ma_hieu"`
	MaVtytLower          string  `json:"ma_vtyt"`
	NhaThauLower         string  `json:"nha_thau"`
	NuocSXLower          string  `json:"nuoc_sx"`
	QuyCachLower         string  `json:"quy_cach"`
	QuyetDinhLower       string  `json:"quyet_dinh"`
	SlNhapLower          int     `json:"sl_nhap"`
	SlXuatLower          int     `json:"sl_xuat"`
	SoLuongTonDauKyLower int     `json:"so_luong_ton_dau_ky"`
	TenVtytBvLower       string  `json:"ten_vtyt_bv"`
}

type internalSupplyAPIResponse struct {
	Data  []internalSupplyAPIRow `json:"data"`
	Rows  []internalSupplyAPIRow `json:"rows"`
	Items []internalSupplyAPIRow `json:"items"`
}

func NewInternalSupplySyncService(
	cfg *config.Config,
	repo internalSupplySyncRepository,
	contactRepo companyContactSyncRepository,
) *InternalSupplySyncService {
	location, err := time.LoadLocation(strings.TrimSpace(cfg.InternalSupplySyncTimezone))
	if err != nil {
		log.Printf("[internal-supply-sync] invalid timezone %q, fallback to local: %v", cfg.InternalSupplySyncTimezone, err)
		location = time.Local
	}

	timeout := time.Duration(cfg.InternalSupplyAPITimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}

	return &InternalSupplySyncService{
		config:      cfg,
		repo:        repo,
		contactRepo: contactRepo,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		location: location,
	}
}

func (s *InternalSupplySyncService) Start(ctx context.Context) {
	if !s.config.InternalSupplySyncEnabled {
		log.Println("[internal-supply-sync] disabled by INTERNAL_SUPPLY_SYNC_ENABLED")
		return
	}
	if strings.TrimSpace(s.config.InternalSupplyAPIURL) == "" {
		log.Println("[internal-supply-sync] skipped because INTERNAL_SUPPLY_API_URL is empty")
		return
	}

	go s.runScheduler(ctx)
}

func (s *InternalSupplySyncService) RunOnce(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Sync supplies from API
	rows, err := s.fetchRows(ctx, "/api_trangbi_thongtinvattu?method=select")
	if err != nil {
		return 0, fmt.Errorf("error syncing supplies: %w", err)
	}
	if len(rows) == 0 {
		return 0, fmt.Errorf("internal supply API returned no product rows (ensure INTERNAL_SUPPLY_API_BODY is set correctly)")
	}

	inputs := make([]models.SupplyUpsertInput, 0, len(rows))
	for index, row := range rows {
		inputs = append(inputs, mapInternalSupplyRow(row, index))
	}

	if err := s.repo.ReplaceAll(inputs); err != nil {
		return 0, fmt.Errorf("error updating supplies database: %w", err)
	}

	log.Printf("[internal-supply-sync] synced %d supply rows successfully", len(inputs))

	// 2. Sync company contacts from partner select API
	partners, err := s.fetchPartners(ctx)
	if err != nil {
		log.Printf("[internal-supply-sync] warning: failed to fetch partners: %v", err)
	} else if len(partners) > 0 {
		contacts := make([]models.CompanyContact, 0, len(partners))
		defaultEmail := models.ResolveDefaultCompanyContactEmail()
		for _, p := range partners {
			contacts = append(contacts, models.CompanyContact{
				MaSoThue:  strings.TrimSpace(p.Code), // Use partner Code (e.g. NCC_DATVIET) as primary key ma_so_thue
				TenCongTy: strings.TrimSpace(p.Name),
				Gmail:     defaultEmail,
			})
		}
		if err := s.contactRepo.ReplaceAll(contacts); err != nil {
			log.Printf("[internal-supply-sync] warning: failed to update company contacts: %v", err)
		} else {
			log.Printf("[internal-supply-sync] synced %d company contacts successfully", len(contacts))
		}
	}

	return len(inputs), nil
}

func (s *InternalSupplySyncService) runScheduler(ctx context.Context) {
	if s.config.InternalSupplySyncRunOnStartup {
		if _, err := s.RunOnce(ctx); err != nil {
			log.Printf("[internal-supply-sync] startup sync failed: %v", err)
		}
	}

	for {
		waitDuration := s.durationUntilNextRun(time.Now())
		timer := time.NewTimer(waitDuration)

		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if _, err := s.RunOnce(ctx); err != nil {
				log.Printf("[internal-supply-sync] scheduled sync failed: %v", err)
			}
		}
	}
}

func (s *InternalSupplySyncService) durationUntilNextRun(now time.Time) time.Duration {
	current := now.In(s.location)
	nextRun := time.Date(
		current.Year(),
		current.Month(),
		current.Day(),
		s.config.InternalSupplySyncHour,
		s.config.InternalSupplySyncMinute,
		0,
		0,
		s.location,
	)

	if !nextRun.After(current) {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	return nextRun.Sub(current)
}

func (s *InternalSupplySyncService) fetchRows(ctx context.Context, path string) ([]internalSupplyAPIRow, error) {
	apiURL := s.config.InternalSupplyAPIURL + path
	bodyString := strings.TrimSpace(s.config.InternalSupplyAPIBody)
	if bodyString == "" {
		bodyString = "{}"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(bodyString))
	if err != nil {
		return nil, fmt.Errorf("error creating internal supply API request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	if token := strings.TrimSpace(s.config.InternalSupplyAPIToken); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if cookie := strings.TrimSpace(s.config.InternalSupplyAPICookie); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error calling internal supply API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("internal supply API returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading internal supply API response: %w", err)
	}

	var objectPayload internalSupplyAPIResponse
	if err := json.Unmarshal(body, &objectPayload); err != nil {
		var listPayload []internalSupplyAPIRow
		if err2 := json.Unmarshal(body, &listPayload); err2 == nil {
			return listPayload, nil
		}
		return nil, fmt.Errorf("error decoding internal supply API response: %w", err)
	}

	switch {
	case len(objectPayload.Data) > 0:
		return objectPayload.Data, nil
	case len(objectPayload.Rows) > 0:
		return objectPayload.Rows, nil
	case len(objectPayload.Items) > 0:
		return objectPayload.Items, nil
	default:
		return nil, nil
	}
}

type hospitalPartnerRow struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type hospitalPartnerAPIResponse struct {
	Data []hospitalPartnerRow `json:"data"`
}

func (s *InternalSupplySyncService) fetchPartners(ctx context.Context) ([]hospitalPartnerRow, error) {
	apiURL := s.config.InternalSupplyAPIURL + "/partner_select_for_po?method=select"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader("{}"))
	if err != nil {
		return nil, fmt.Errorf("error creating partner API request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	if token := strings.TrimSpace(s.config.InternalSupplyAPIToken); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if cookie := strings.TrimSpace(s.config.InternalSupplyAPICookie); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error calling partner API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("partner API returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading partner API response: %w", err)
	}

	var payload hospitalPartnerAPIResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		var listPayload []hospitalPartnerRow
		if err2 := json.Unmarshal(body, &listPayload); err2 == nil {
			return listPayload, nil
		}
		return nil, fmt.Errorf("error decoding partner API response: %w", err)
	}

	return payload.Data, nil
}

func mapInternalSupplyRow(row internalSupplyAPIRow, index int) models.SupplyUpsertInput {
	productID := firstNonZero(row.ProductID, row.NewID)
	id := firstNonEmpty(row.ID, row.MaVtyt, row.NewCode, row.MaVtytLower)
	name := firstNonEmpty(row.Name, row.TenVatTuBV, row.NewName, row.TenVtytBvLower)
	unit := firstNonEmpty(row.Unit, row.DonViTinh, row.UomName, row.PurchaseUomName, row.DonViTinhLower)
	quyCachDongGoi := firstNonEmpty(row.QuyCachDongGoi, row.QuyCach, row.QuyCachLower)
	thongTinThau := firstNonEmpty(row.ThongTinThau, row.QuyetDinh, row.QuyetDinhLower)
	nhaCungCap := firstNonEmpty(row.NhaCungCap, row.NhaThau, row.NhaThauLower)

	price := row.Price
	if price == 0 {
		price = row.DonGia
	}
	if price == 0 {
		price = row.OriginalPrice
	}
	if price == 0 {
		price = row.DonGiaLower
	}

	hangSX := firstNonEmpty(row.HangSXAlt, row.HangSX, row.HangSXLower)
	nuocSX := firstNonEmpty(row.NuocSXAlt, row.NuocSX, row.NuocSXLower)
	if hangSX == "" && row.ManufactureName != "" {
		parts := strings.Split(row.ManufactureName, "/")
		hangSX = strings.TrimSpace(parts[0])
		if len(parts) > 1 {
			nuocSX = strings.TrimSpace(parts[1])
		}
	}

	tonDauKy := firstNonZero(row.TonDauKy, row.SoLuongTonKho, row.SlTon, row.SoLuongTonDauKyLower)
	nhapTrongKy := firstNonZero(row.NhapTrongKy, row.SlNhap, row.SlNhapLower)
	xuatTrongKy := firstNonZero(row.XuatTrongKy, row.SlXuat, row.SlXuatLower)

	return models.SupplyUpsertInput{
		IDX1:            firstNonZero(row.Idx1, index+1),
		ProductID:       productID,
		GroupName:       strings.TrimSpace(row.GroupName),
		ID:              strings.TrimSpace(id),
		IDX2:            strings.TrimSpace(row.Idx2),
		MaHieu:          strings.TrimSpace(firstNonEmpty(row.MaHieu, row.MaHieuLower)),
		TypeName:        strings.TrimSpace(row.TypeName),
		Name:            strings.TrimSpace(name),
		Unit:            strings.TrimSpace(unit),
		QuyCachDongGoi:  strings.TrimSpace(quyCachDongGoi),
		QuyCachGiaoHang: strings.TrimSpace(row.QuyCachGiaoHang),
		QuyCachToiThieu: strings.TrimSpace(row.QuyCachToiThieu),
		ThongTinThau:    strings.TrimSpace(thongTinThau),
		TongThau:        strings.TrimSpace(row.TongThau),
		HangSX:          strings.TrimSpace(hangSX),
		NuocSX:          strings.TrimSpace(nuocSX),
		NhaCungCap:      strings.TrimSpace(nhaCungCap),
		Price:           price,
		TonDauKy:        tonDauKy,
		NhapTrongKy:     nhapTrongKy,
		XuatTrongKy:     xuatTrongKy,
		TongNhap:        firstNonZero(row.TongNhap, nhapTrongKy),
		TonKhoMin:       row.TonKhoMin,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
