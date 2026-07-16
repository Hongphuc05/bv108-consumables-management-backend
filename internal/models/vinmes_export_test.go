package models

import (
	"testing"
	"time"
)

func TestExtractTenderReference(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "quyet dinh full suffix",
			input:    "[A00191] Theo hop dong so 1833 ngay 31/12/2025 va Quyet dinh so 9528/QD-BV ngay 25/12/2025",
			expected: "9528/QĐ-BV",
		},
		{
			name:     "qd short form",
			input:    "(D06041) ... ( HD so 2063 ngay 31/12/2025 va QD so 9534 )",
			expected: "9534/QĐ-BV",
		},
		{
			name:     "missing tender",
			input:    "Vat tu khong co thong tin goi thau",
			expected: defaultVinmesGoiThau,
		},
		{
			name:     "ignore unsupported code",
			input:    "Theo quyet dinh so 9529/QD-BV",
			expected: defaultVinmesGoiThau,
		},
		{
			name:     "extract from invoice context line",
			input:    "[B001142.1] Nẹp 2.0mm, thẳng, dày 1mm Hãng sản xuất: AGOMED | Nước sản xuất: Đức | (Theo HĐ số 1864 Ngày 31/12/2025 và QĐ số 9530)",
			expected: "9530/QĐ-BV",
		},
		{
			name:     "Vinmes contract package 4418",
			input:    "Theo HĐ số DMEC-DEMO và QĐ số 4418",
			expected: "4418/QĐ-BV",
		},
		{
			name:     "Vinmes contract package 2233",
			input:    "Theo HĐ số DMEC-DEMO và QĐ số 2233",
			expected: "2233/QĐ-BV",
		},
		{
			name:     "Vinmes contract package 7313",
			input:    "Theo HĐ số DMEC-DEMO và QĐ số 7313/QĐ-BV;G1;N1;2023",
			expected: "7313/QĐ-BV",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := ExtractTenderReference(tc.input)
			if actual != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, actual)
			}
		})
	}
}

func TestBuildVinmesExportItem(t *testing.T) {
	t.Parallel()

	invoiceDate := time.Date(2026, 6, 29, 7, 0, 0, 0, time.UTC)
	matchedAt := time.Date(2026, 7, 1, 9, 30, 0, 0, time.UTC)
	tax := 5.0

	item := BuildVinmesExportItem(VinmesExportSource{
		ReconciliationID:    7,
		OrderHistoryID:      12,
		InvoiceNumber:       "00002116",
		InvoiceDate:         &invoiceDate,
		InvoiceItemCode:     "D06041",
		InvoiceItemName:     "(D06041) ... QD so 9534",
		InvoiceContext:      "Hãng sản xuất: Test | (Theo HĐ số 2063 ngày 31/12/2025 và QĐ số 9534)",
		InvoiceQty:          19,
		InvoiceTaxRate:      &tax,
		SupplierName:        "MERINCO",
		SupplierTaxCode:     "0101234567",
		SupplierBankAccount: "0031 10133 7009",
		MatchedAt:           matchedAt,
	})

	if item.SoPhieu != "PN20260701000007" {
		t.Fatalf("unexpected soPhieu: %s", item.SoPhieu)
	}
	if item.NgayYeuCau != "01/07/2026" {
		t.Fatalf("unexpected ngayYeuCau: %s", item.NgayYeuCau)
	}
	if item.NgayHoaDon != "29/06/2026" {
		t.Fatalf("unexpected ngayHoaDon: %s", item.NgayHoaDon)
	}
	if item.GoiThau != "9534/QĐ-BV" {
		t.Fatalf("unexpected goiThau: %s", item.GoiThau)
	}
	if item.Thue != "5%" {
		t.Fatalf("unexpected thue: %s", item.Thue)
	}
	if item.MaSoThueNhaCungCap != "0101234567" {
		t.Fatalf("unexpected maSoThueNhaCungCap: %s", item.MaSoThueNhaCungCap)
	}
	if item.SoTKNganHangNhaCungCap != "0031 10133 7009" {
		t.Fatalf("unexpected soTkNganHangNhaCungCap: %s", item.SoTKNganHangNhaCungCap)
	}
}

func TestBuildVinmesExportMastersGroupsInvoiceLines(t *testing.T) {
	t.Parallel()

	invoiceDate := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)
	matchedAt := time.Date(2026, 7, 7, 9, 30, 0, 0, time.UTC)
	tax := 0.0
	sources := []VinmesExportSource{
		{
			ReconciliationID:    88,
			OrderHistoryID:      333,
			InvoiceIDHoaDon:     "invoice-a",
			InvoiceNumber:       "00000315",
			InvoiceKyHieu:       "1C26TKK",
			InvoiceDate:         &invoiceDate,
			InvoiceItemCode:     "B001142.1",
			InvoiceItemName:     "Nẹp 2.0mm",
			InvoiceQty:          15,
			InvoiceTaxRate:      &tax,
			SupplierName:        "KINDAKARE",
			SupplierTaxCode:     "0101234567",
			SupplierBankAccount: "0031 10133 7009",
			MatchedAt:           matchedAt,
		},
		{
			ReconciliationID:    87,
			OrderHistoryID:      332,
			InvoiceIDHoaDon:     "invoice-a",
			InvoiceNumber:       "00000315",
			InvoiceKyHieu:       "1C26TKK",
			InvoiceDate:         &invoiceDate,
			InvoiceItemCode:     "B001141.1",
			InvoiceItemName:     "Nẹp chữ T",
			InvoiceContext:      "Theo HĐ số 1864 và QĐ số 9530",
			InvoiceQty:          2,
			InvoiceTaxRate:      &tax,
			SupplierName:        "KINDAKARE",
			SupplierTaxCode:     "0101234567",
			SupplierBankAccount: "0031 10133 7009",
			MatchedAt:           matchedAt,
		},
		{
			ReconciliationID: 90,
			OrderHistoryID:   335,
			InvoiceIDHoaDon:  "invoice-b",
			InvoiceNumber:    "00000316",
			InvoiceDate:      &invoiceDate,
			InvoiceItemCode:  "D06041",
			InvoiceQty:       5,
			MatchedAt:        matchedAt,
		},
	}

	masters := BuildVinmesExportMasters(sources)
	if len(masters) != 2 {
		t.Fatalf("expected 2 masters, got %d", len(masters))
	}
	if masters[0].UserID != "trangbi" {
		t.Fatalf("unexpected userId: %s", masters[0].UserID)
	}
	if masters[0].MaSoThueNhaCungCap != "0101234567" {
		t.Fatalf("unexpected supplier tax code: %s", masters[0].MaSoThueNhaCungCap)
	}
	if masters[0].SoTKNganHangNhaCungCap != "0031 10133 7009" {
		t.Fatalf("unexpected supplier bank account: %s", masters[0].SoTKNganHangNhaCungCap)
	}
	if masters[0].GoiThau != "9530/QĐ-BV" {
		t.Fatalf("unexpected goiThau: %s", masters[0].GoiThau)
	}
	if len(masters[0].Details) != 2 {
		t.Fatalf("expected 2 details in first master, got %d", len(masters[0].Details))
	}
	if len(masters[1].Details) != 1 {
		t.Fatalf("expected 1 detail in second master, got %d", len(masters[1].Details))
	}
}
