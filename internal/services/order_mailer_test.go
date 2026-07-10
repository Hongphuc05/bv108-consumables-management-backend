package services

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	gomail "github.com/wneessen/go-mail"
)

func TestRenderPlacedOrderEmailBody(t *testing.T) {
	body, err := renderPlacedOrderEmailBody("Công ty ABC")
	if err != nil {
		t.Fatalf("renderPlacedOrderEmailBody() error = %v", err)
	}

	expected := "Kính gửi công ty Công ty ABC, Khoa Trang bị- BV TWQĐ 108 xin gửi đến Quý công ty đơn đặt hàng vật tư theo file PDF đính kèm"
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
		ContactTitle: "PCNK Trang bị",
		ContactPhone: "0988335388",
		Items: []OrderEmailItem{
			{
				Index:        1,
				TenVatTu:     "Bơm kim tiêm",
				MaXuatHoaDon: "VT001",
				MaHieu:       "MH001",
				HangNuocSX:   "Hãng A / Việt Nam",
				DonViTinh:    "Cái",
				SoLuong:      25,
			},
			{
				Index:        2,
				TenVatTu:     "Dây truyền dịch",
				MaXuatHoaDon: "VT002",
				MaHieu:       "MH002",
				HangNuocSX:   "Hãng B / Đức",
				DonViTinh:    "Bộ",
				SoLuong:      10,
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

func TestResolveSMTPTLSPolicy(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  gomail.TLSPolicy
	}{
		{name: "default mandatory", input: "", want: gomail.TLSMandatory},
		{name: "mandatory explicit", input: "mandatory", want: gomail.TLSMandatory},
		{name: "opportunistic", input: "opportunistic", want: gomail.TLSOpportunistic},
		{name: "no tls", input: "NoTLS", want: gomail.NoTLS},
		{name: "none alias", input: "none", want: gomail.NoTLS},
		{name: "unknown falls back", input: "weird", want: gomail.TLSMandatory},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := resolveSMTPTLSPolicy(tc.input)
			if got != tc.want {
				t.Fatalf("resolveSMTPTLSPolicy(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestNewSMTPOrderMailerNormalizesConfig(t *testing.T) {
	mailer := NewSMTPOrderMailer(SMTPOrderMailerConfig{
		Host:        " smtp.example.com ",
		Port:        " 587 ",
		Username:    " user@example.com ",
		AppPassword: " ab cd ef ",
		From:        " sender@example.com ",
		TLSPolicy:   " opportunistic ",
	})

	if mailer.host != "smtp.example.com" || mailer.port != "587" {
		t.Fatalf("unexpected endpoint %q:%q", mailer.host, mailer.port)
	}
	if mailer.username != "user@example.com" || mailer.appPassword != "abcdef" {
		t.Fatal("SMTP credentials were not normalized")
	}
	if mailer.from != "sender@example.com" {
		t.Fatalf("from = %q", mailer.from)
	}
	if mailer.tlsPolicy != gomail.TLSOpportunistic {
		t.Fatalf("tlsPolicy = %v", mailer.tlsPolicy)
	}
}
