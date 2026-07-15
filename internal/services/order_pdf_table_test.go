package services

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/phpdave11/gofpdf"
)

var orderPDFPagePattern = regexp.MustCompile(`/Type\s*/Page(?:\s|>>)`)

func TestRenderPlacedOrderAttachmentPDFWithTwentyLongItems(t *testing.T) {
	if _, _, err := resolveOrderPDFFontPaths(); err != nil {
		t.Skipf("PDF fonts unavailable in test environment: %v", err)
	}

	items := make([]OrderEmailItem, 0, 20)
	for index := 1; index <= 20; index++ {
		items = append(items, OrderEmailItem{
			Index:        index,
			TenVatTu:     fmt.Sprintf("Vật tư phẫu thuật số %02d, quy cách đóng gói vô trùng dùng một lần tại bệnh viện", index),
			MaXuatHoaDon: fmt.Sprintf("HD-%04d-%02d", 2026, index),
			MaHieu:       fmt.Sprintf("MODEL-%02d-VERSION-LONG", index),
			HangNuocSX:   "Công ty sản xuất thiết bị y tế tiêu chuẩn quốc tế / Việt Nam",
			DonViTinh:    "Hộp vô trùng",
			SoLuong:      index * 125,
		})
	}

	pdfBytes, err := renderPlacedOrderAttachmentPDF(orderPDFStressDocument(items))
	if err != nil {
		t.Fatalf("renderPlacedOrderAttachmentPDF() error = %v", err)
	}

	assertOrderPDFStructure(t, pdfBytes, 2, 10)
	writeOrderPDFTestArtifact(t, pdfBytes)
}

func TestRenderPlacedOrderAttachmentPDFSplitsExtremelyLongRow(t *testing.T) {
	if _, _, err := resolveOrderPDFFontPaths(); err != nil {
		t.Skipf("PDF fonts unavailable in test environment: %v", err)
	}

	items := []OrderEmailItem{
		{
			Index:        1,
			TenVatTu:     strings.Repeat("Tên vật tư rất dài cần được giữ đầy đủ khi tự động ngắt trang. ", 100),
			MaXuatHoaDon: "HD-EXTREME-001",
			MaHieu:       "MODEL-EXTREME-001",
			HangNuocSX:   "Nhà sản xuất thử nghiệm / Việt Nam",
			DonViTinh:    "Cái",
			SoLuong:      1,
		},
	}

	pdfBytes, err := renderPlacedOrderAttachmentPDF(orderPDFStressDocument(items))
	if err != nil {
		t.Fatalf("renderPlacedOrderAttachmentPDF() error = %v", err)
	}

	assertOrderPDFStructure(t, pdfBytes, 2, 20)
}

func TestOrderPDFTableLayoutHelpers(t *testing.T) {
	width := 0.0
	for _, column := range orderPDFTableColumns {
		width += column.width
	}
	if width != 180 {
		t.Fatalf("table width = %.1fmm, want 180mm", width)
	}

	if got := orderPDFTableHeight(1, orderPDFTableLineHeight, orderPDFTableMinimumRowHeight); got != orderPDFTableMinimumRowHeight {
		t.Fatalf("single-line row height = %.1f", got)
	}
	if got := orderPDFTableHeight(3, orderPDFTableLineHeight, orderPDFTableMinimumRowHeight); got <= orderPDFTableMinimumRowHeight {
		t.Fatalf("wrapped row height = %.1f", got)
	}

	cells := []orderPDFTableCellLayout{
		{width: 10, lines: []string{"a", "b", "c"}},
		{width: 20, lines: []string{"x"}},
	}
	fragment, remaining := takeOrderPDFTableLines(cells, 2)
	if len(fragment.cells[0].lines) != 2 || len(remaining[0].lines) != 1 {
		t.Fatalf("unexpected split: fragment=%v remaining=%v", fragment.cells[0].lines, remaining[0].lines)
	}
	if len(fragment.cells[1].lines) != 1 || len(remaining[1].lines) != 0 {
		t.Fatalf("short cell split incorrectly: fragment=%v remaining=%v", fragment.cells[1].lines, remaining[1].lines)
	}
}

func TestSplitOrderPDFTableRowMovesBeforeMinimumHeightOverflow(t *testing.T) {
	pdf := newOrderPDFTestCanvas(t)
	pageBottom := orderPDFPageBottomY(pdf)
	pdf.SetY(pageBottom - (orderPDFTableMinimumRowHeight - 1))

	cells := make([]orderPDFTableCellLayout, len(orderPDFTableColumns))
	for index, column := range orderPDFTableColumns {
		cells[index] = orderPDFTableCellLayout{width: column.width, lines: []string{"value"}}
	}
	cells[1].lines = make([]string, 60)
	for index := range cells[1].lines {
		cells[1].lines[index] = fmt.Sprintf("line %d", index+1)
	}

	startPage := pdf.PageNo()
	drawSplitOrderPDFTableRow(pdf, orderPDFTableRowLayout{cells: cells}, func() {
		drawPlacedOrderPDFTableHeader(pdf)
	})
	if pdf.PageNo() <= startPage {
		t.Fatalf("page = %d, want a page break after page %d", pdf.PageNo(), startPage)
	}
	if pdf.Error() != nil {
		t.Fatalf("PDF error = %v", pdf.Error())
	}
}

func orderPDFStressDocument(items []OrderEmailItem) placedOrderDocumentData {
	return placedOrderDocumentData{
		CompanyName:  "CÔNG TY CỔ PHẦN THIẾT BỊ Y TẾ THỬ NGHIỆM",
		CurrentMonth: "07/2026",
		CurrentDate:  "10/07/2026",
		ContactName:  "Nguyễn Thành Trung",
		ContactDept:  "Khoa Trang bị",
		ContactTitle: "PCNK Trang bị",
		ContactPhone: "0988335388",
		Items:        items,
	}
}

func assertOrderPDFStructure(t *testing.T, pdfBytes []byte, minimumPages, maximumPages int) {
	t.Helper()

	if !bytes.HasPrefix(pdfBytes, []byte("%PDF")) {
		t.Fatal("missing PDF header")
	}
	if len(pdfBytes) < 5_000 {
		t.Fatalf("PDF is unexpectedly small: %d bytes", len(pdfBytes))
	}

	pageCount := len(orderPDFPagePattern.FindAll(pdfBytes, -1))
	if pageCount < minimumPages || pageCount > maximumPages {
		t.Fatalf("page count = %d, want between %d and %d", pageCount, minimumPages, maximumPages)
	}
}

func writeOrderPDFTestArtifact(t *testing.T, pdfBytes []byte) {
	t.Helper()

	outputPath := strings.TrimSpace(os.Getenv("ORDER_PDF_TEST_OUTPUT"))
	if outputPath == "" {
		return
	}
	if err := os.WriteFile(outputPath, pdfBytes, 0o644); err != nil {
		t.Fatalf("write PDF test artifact: %v", err)
	}
}

func newOrderPDFTestCanvas(t *testing.T) *gofpdf.Fpdf {
	t.Helper()

	regularPath, boldPath, err := resolveOrderPDFFontPaths()
	if err != nil {
		t.Skipf("PDF fonts unavailable in test environment: %v", err)
	}
	regularFont, err := os.ReadFile(regularPath)
	if err != nil {
		t.Fatalf("read regular font: %v", err)
	}
	boldFont, err := os.ReadFile(boldPath)
	if err != nil {
		t.Fatalf("read bold font: %v", err)
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddUTF8FontFromBytes(orderPDFFontFamily, "", regularFont)
	pdf.AddUTF8FontFromBytes(orderPDFFontFamily, "B", boldFont)
	pdf.AddPage()
	return pdf
}
