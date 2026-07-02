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
			expected: "9528/QD-BV",
		},
		{
			name:     "qd short form",
			input:    "(D06041) ... ( HD so 2063 ngay 31/12/2025 va QD so 9534 )",
			expected: "9534",
		},
		{
			name:     "missing tender",
			input:    "Vat tu khong co thong tin goi thau",
			expected: defaultVinmesKyHieu,
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
		ReconciliationID: 7,
		OrderHistoryID:   12,
		InvoiceNumber:    "00002116",
		InvoiceDate:      &invoiceDate,
		InvoiceItemCode:  "D06041",
		InvoiceItemName:  "(D06041) ... QD so 9534",
		InvoiceQty:       19,
		InvoiceTaxRate:   &tax,
		SupplierName:     "MERINCO",
		MatchedAt:        matchedAt,
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
	if item.Thue != "5%" {
		t.Fatalf("unexpected thue: %s", item.Thue)
	}
}
