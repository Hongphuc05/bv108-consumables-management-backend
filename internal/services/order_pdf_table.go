package services

import (
	"math"
	"strconv"

	"github.com/phpdave11/gofpdf"
)

const (
	orderPDFTableLineHeight       = 5.5
	orderPDFTableHeaderLineHeight = 5.0
	orderPDFTableCellPadding      = 1.2
	orderPDFTableMinimumRowHeight = 10.0
	orderPDFTableHeaderMinHeight  = 12.0
)

type orderPDFTableColumn struct {
	header string
	width  float64
}

type orderPDFTableCellLayout struct {
	width float64
	lines []string
}

type orderPDFTableRowLayout struct {
	cells  []orderPDFTableCellLayout
	height float64
}

var orderPDFTableColumns = []orderPDFTableColumn{
	{header: "STT", width: 12},
	{header: "Tên vật tư", width: 38},
	{header: "Mã xuất\nhóa đơn", width: 23},
	{header: "Mã hiệu", width: 22},
	{header: "Hãng/ Nước\nsản xuất", width: 46},
	{header: "Đơn vị\ntính", width: 21},
	{header: "Số\nlượng", width: 18},
}

func renderPlacedOrderPDFTable(pdf *gofpdf.Fpdf, items []OrderEmailItem) {
	pdf.SetFont(orderPDFFontFamily, "B", orderPDFBodyFontSize)
	pdf.MultiCell(0, orderPDFBodyLineHeight, "1. Danh sách vật tư đặt hàng:", "", "L", false)
	pdf.Ln(orderPDFParagraphSpacing - 1)

	drawHeader := func() {
		drawPlacedOrderPDFTableHeader(pdf)
	}

	drawHeader()
	if len(items) == 0 {
		drawPlacedOrderPDFTableRow(pdf, []string{"1", "...", "...", "...", "...", "...", "..."}, drawHeader)
	} else {
		for _, item := range items {
			drawPlacedOrderPDFTableRow(pdf, orderPDFTableValues(item), drawHeader)
		}
	}

	pdf.Ln(orderPDFSectionSpacing - 1)
}

func drawPlacedOrderPDFTableHeader(pdf *gofpdf.Fpdf) {
	pdf.SetFont(orderPDFFontFamily, "B", orderPDFTableFontSize)
	headerValues := make([]string, len(orderPDFTableColumns))
	for index, column := range orderPDFTableColumns {
		headerValues[index] = column.header
	}
	header := layoutOrderPDFTableRow(pdf, headerValues, orderPDFTableHeaderLineHeight, orderPDFTableHeaderMinHeight)
	drawCenteredOrderPDFTableRow(pdf, header, orderPDFTableHeaderLineHeight)
}

func orderPDFTableValues(item OrderEmailItem) []string {
	return []string{
		strconv.Itoa(item.Index),
		nonEmptyPDFText(item.TenVatTu),
		nonEmptyPDFText(item.MaXuatHoaDon),
		nonEmptyPDFText(item.MaHieu),
		nonEmptyPDFText(item.HangNuocSX),
		nonEmptyPDFText(item.DonViTinh),
		strconv.Itoa(item.SoLuong),
	}
}

func drawPlacedOrderPDFTableRow(pdf *gofpdf.Fpdf, values []string, drawHeader func()) {
	pdf.SetFont(orderPDFFontFamily, "", orderPDFTableFontSize)
	row := layoutOrderPDFTableRow(pdf, values, orderPDFTableLineHeight, orderPDFTableMinimumRowHeight)

	freshPageCapacity := orderPDFTableFreshPageCapacity(pdf)
	if row.height <= freshPageCapacity {
		if !orderPDFHasVerticalSpace(pdf, row.height) {
			pdf.AddPage()
			drawHeader()
			pdf.SetFont(orderPDFFontFamily, "", orderPDFTableFontSize)
		}
		drawCenteredOrderPDFTableRow(pdf, row, orderPDFTableLineHeight)
		return
	}

	drawSplitOrderPDFTableRow(pdf, row, drawHeader)
}

func drawSplitOrderPDFTableRow(pdf *gofpdf.Fpdf, row orderPDFTableRowLayout, drawHeader func()) {
	remaining := cloneOrderPDFTableCells(row.cells)
	for maxOrderPDFTableLineCount(remaining) > 0 {
		availableLines := orderPDFAvailableTableLines(pdf, orderPDFTableLineHeight)
		if availableLines < 1 || !orderPDFHasVerticalSpace(pdf, orderPDFTableMinimumRowHeight) {
			pdf.AddPage()
			drawHeader()
			pdf.SetFont(orderPDFFontFamily, "", orderPDFTableFontSize)
			availableLines = orderPDFAvailableTableLines(pdf, orderPDFTableLineHeight)
		}

		fragment, next := takeOrderPDFTableLines(remaining, availableLines)
		fragment.height = orderPDFTableHeight(maxOrderPDFTableLineCount(fragment.cells), orderPDFTableLineHeight, orderPDFTableMinimumRowHeight)
		drawCenteredOrderPDFTableRow(pdf, fragment, orderPDFTableLineHeight)
		remaining = next

		if maxOrderPDFTableLineCount(remaining) > 0 {
			pdf.AddPage()
			drawHeader()
			pdf.SetFont(orderPDFFontFamily, "", orderPDFTableFontSize)
		}
	}
}

func layoutOrderPDFTableRow(pdf *gofpdf.Fpdf, values []string, lineHeight, minimumHeight float64) orderPDFTableRowLayout {
	cells := make([]orderPDFTableCellLayout, len(orderPDFTableColumns))
	maxLineCount := 1
	for index, column := range orderPDFTableColumns {
		value := "-"
		if index < len(values) {
			value = values[index]
		}
		lines := wrapOrderPDFTableText(pdf, value, column.width-(orderPDFTableCellPadding*2))
		cells[index] = orderPDFTableCellLayout{width: column.width, lines: lines}
		if len(lines) > maxLineCount {
			maxLineCount = len(lines)
		}
	}

	return orderPDFTableRowLayout{
		cells:  cells,
		height: orderPDFTableHeight(maxLineCount, lineHeight, minimumHeight),
	}
}

func orderPDFTableHeight(lineCount int, lineHeight, minimumHeight float64) float64 {
	height := float64(max(lineCount, 1))*lineHeight + (orderPDFTableCellPadding * 2)
	return math.Max(height, minimumHeight)
}

func drawCenteredOrderPDFTableRow(pdf *gofpdf.Fpdf, row orderPDFTableRowLayout, lineHeight float64) {
	x := orderPDFTableStartX(pdf)
	y := pdf.GetY()

	for _, cell := range row.cells {
		pdf.Rect(x, y, cell.width, row.height, "")
		textHeight := float64(len(cell.lines)) * lineHeight
		textY := y + ((row.height - textHeight) / 2)
		innerWidth := cell.width - (orderPDFTableCellPadding * 2)

		for lineIndex, line := range cell.lines {
			pdf.SetXY(x+orderPDFTableCellPadding, textY+(float64(lineIndex)*lineHeight))
			pdf.CellFormat(innerWidth, lineHeight, line, "", 0, "C", false, 0, "")
		}

		x += cell.width
	}

	pdf.SetXY(orderPDFTableStartX(pdf), y+row.height)
}

func orderPDFTableStartX(pdf *gofpdf.Fpdf) float64 {
	pageWidth, _ := pdf.GetPageSize()
	leftMargin, _, rightMargin, _ := pdf.GetMargins()
	contentWidth := pageWidth - leftMargin - rightMargin
	tableWidth := 0.0
	for _, column := range orderPDFTableColumns {
		tableWidth += column.width
	}
	return leftMargin + math.Max(0, (contentWidth-tableWidth)/2)
}

func orderPDFTableFreshPageCapacity(pdf *gofpdf.Fpdf) float64 {
	_, topMargin, _, _ := pdf.GetMargins()
	return orderPDFPageBottomY(pdf) - topMargin - orderPDFTableHeaderMinHeight
}

func orderPDFHasVerticalSpace(pdf *gofpdf.Fpdf, height float64) bool {
	return pdf.GetY()+height <= orderPDFPageBottomY(pdf)
}

func ensureOrderPDFVerticalSpace(pdf *gofpdf.Fpdf, height float64) {
	if !orderPDFHasVerticalSpace(pdf, height) {
		pdf.AddPage()
	}
}

func orderPDFPageBottomY(pdf *gofpdf.Fpdf) float64 {
	_, pageHeight := pdf.GetPageSize()
	_, bottomMargin := pdf.GetAutoPageBreak()
	return pageHeight - bottomMargin
}

func orderPDFAvailableTableLines(pdf *gofpdf.Fpdf, lineHeight float64) int {
	availableHeight := orderPDFPageBottomY(pdf) - pdf.GetY() - (orderPDFTableCellPadding * 2)
	return max(0, int(math.Floor(availableHeight/lineHeight)))
}

func cloneOrderPDFTableCells(cells []orderPDFTableCellLayout) []orderPDFTableCellLayout {
	cloned := make([]orderPDFTableCellLayout, len(cells))
	for index, cell := range cells {
		cloned[index] = orderPDFTableCellLayout{
			width: cell.width,
			lines: append([]string(nil), cell.lines...),
		}
	}
	return cloned
}

func maxOrderPDFTableLineCount(cells []orderPDFTableCellLayout) int {
	maxLines := 0
	for _, cell := range cells {
		if len(cell.lines) > maxLines {
			maxLines = len(cell.lines)
		}
	}
	return maxLines
}

func takeOrderPDFTableLines(cells []orderPDFTableCellLayout, count int) (orderPDFTableRowLayout, []orderPDFTableCellLayout) {
	fragment := make([]orderPDFTableCellLayout, len(cells))
	remaining := make([]orderPDFTableCellLayout, len(cells))
	for index, cell := range cells {
		takeCount := min(count, len(cell.lines))
		fragment[index] = orderPDFTableCellLayout{
			width: cell.width,
			lines: append([]string(nil), cell.lines[:takeCount]...),
		}
		remaining[index] = orderPDFTableCellLayout{
			width: cell.width,
			lines: append([]string(nil), cell.lines[takeCount:]...),
		}
	}
	return orderPDFTableRowLayout{cells: fragment}, remaining
}
