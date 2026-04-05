package services

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRenderPlacedOrderEmailBody(t *testing.T) {
	body, err := renderPlacedOrderEmailBody()
	if err != nil {
		t.Fatalf("renderPlacedOrderEmailBody() error = %v", err)
	}

	expected := "Khoa trang bị bênh viên quân đội TW 108 đặt hàng quý công ty theo nội dung đính kèm sau:\npdf đính kèm"
	if body != expected {
		t.Fatalf("unexpected email body: %q", body)
	}
}

func TestRenderPlacedOrderAttachmentPDF(t *testing.T) {
	if _, _, err := resolveOrderPDFFontPaths(); err != nil {
		t.Skipf("PDF fonts unavailable in test environment: %v", err)
	}

	pdfBytes, err := renderPlacedOrderAttachmentPDF(placedOrderDocumentData{
		CompanyName:  "Công ty ABC",
		CurrentMonth: "04/2026",
		CurrentDate:  "05/04/2026",
		ContactName:  "Nguyễn Thành Trung",
		ContactDept:  "Khoa Trang bị",
		Items: []OrderEmailItem{
			{
				Index:     1,
				TenVatTu:  "Bơm kim tiêm",
				MaVatTu:   "VT001",
				DonViTinh: "Cái",
				SoLuong:   25,
			},
			{
				Index:     2,
				TenVatTu:  "Dây truyền dịch",
				MaVatTu:   "VT002",
				DonViTinh: "Bộ",
				SoLuong:   10,
			},
		},
	})
	if err != nil {
		t.Fatalf("renderPlacedOrderAttachmentPDF() error = %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Fatal("expected non-empty PDF bytes")
	}
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF")) {
		t.Fatalf("expected PDF header, got %q", string(pdfBytes[:min(len(pdfBytes), 8)]))
	}
	if len(pdfBytes) < 1024 {
		t.Fatalf("expected PDF to contain rendered content, got only %d bytes", len(pdfBytes))
	}
}

func TestResolveOrderPDFFontPathsUsesEnvOverride(t *testing.T) {
	tempDir := t.TempDir()
	regularPath := filepath.Join(tempDir, "regular.ttf")
	boldPath := filepath.Join(tempDir, "bold.ttf")

	if err := os.WriteFile(regularPath, []byte("regular"), 0o644); err != nil {
		t.Fatalf("WriteFile regular font: %v", err)
	}
	if err := os.WriteFile(boldPath, []byte("bold"), 0o644); err != nil {
		t.Fatalf("WriteFile bold font: %v", err)
	}

	t.Setenv(orderPDFFontRegularEnv, regularPath)
	t.Setenv(orderPDFFontBoldEnv, boldPath)

	gotRegular, gotBold, err := resolveOrderPDFFontPaths()
	if err != nil {
		t.Fatalf("resolveOrderPDFFontPaths() error = %v", err)
	}
	if gotRegular != regularPath {
		t.Fatalf("expected regular path %q, got %q", regularPath, gotRegular)
	}
	if gotBold != boldPath {
		t.Fatalf("expected bold path %q, got %q", boldPath, gotBold)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
