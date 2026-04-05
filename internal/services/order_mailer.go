package services

import (
	"bytes"
	"fmt"
	"io/fs"
	stdmail "net/mail"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/phpdave11/gofpdf"
	gomail "github.com/wneessen/go-mail"
)

type OrderEmailSender interface {
	SendPlacedOrderEmail(recipientEmail, supplierName string, items []OrderEmailItem) error
}

type OrderEmailItem struct {
	Index         int
	TenVatTu      string
	MaVatTu       string
	DonViTinh     string
	SoLuong       int
	DotGiaoHang   string
}

type SMTPOrderMailer struct {
	host        string
	port        string
	username    string
	appPassword string
	from        string
}

func NewSMTPOrderMailer(host, port, username, appPassword, from string) *SMTPOrderMailer {
	return &SMTPOrderMailer{
		host:        strings.TrimSpace(host),
		port:        strings.TrimSpace(port),
		username:    strings.TrimSpace(username),
		appPassword: strings.ReplaceAll(strings.TrimSpace(appPassword), " ", ""),
		from:        strings.TrimSpace(from),
	}
}

const (
	placedOrderEmailSubject    = "ĐƠN ĐẶT HÀNG VẬT TƯ TẠI BỆNH VIỆN QUÂN ĐỘI TW 108"
	placedOrderEmailBody       = "Khoa trang bị bênh viên quân đội TW 108 đặt hàng quý công ty theo nội dung đính kèm sau:\npdf đính kèm"
	placedOrderAttachmentName  = "don-dat-hang-vat-tu-bv108.pdf"
	placedOrderPDFContentType  = gomail.ContentType("application/pdf")
	orderPDFFontFamily         = "orderpdf"
	orderPDFFontRegularEnv     = "ORDER_PDF_FONT_REGULAR"
	orderPDFFontBoldEnv        = "ORDER_PDF_FONT_BOLD"
	orderPDFBodyFontSize       = 13.0
	orderPDFTableFontSize      = 11.0
	orderPDFBodyLineHeight     = 6.0
	orderPDFParagraphSpacing   = 4.0
	orderPDFSectionSpacing     = 6.0
)

func (m *SMTPOrderMailer) SendPlacedOrderEmail(recipientEmail, supplierName string, items []OrderEmailItem) error {
	supplierName = strings.TrimSpace(supplierName)
	recipientEmail = strings.TrimSpace(recipientEmail)

	if recipientEmail == "" {
		if supplierName == "" {
			return fmt.Errorf("missing company email")
		}
		return fmt.Errorf("missing company email for %s", supplierName)
	}

	if _, err := stdmail.ParseAddress(recipientEmail); err != nil {
		if supplierName == "" {
			return fmt.Errorf("invalid company email: %s", recipientEmail)
		}
		return fmt.Errorf("invalid company email for %s: %s", supplierName, recipientEmail)
	}

	if m.host == "" || m.port == "" || m.username == "" || m.appPassword == "" || m.from == "" {
		return fmt.Errorf("smtp is not configured. Set SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_APP_PASSWORD, and SMTP_FROM")
	}

	if _, err := stdmail.ParseAddress(m.from); err != nil {
		return fmt.Errorf("invalid SMTP_FROM address: %w", err)
	}

	if len(items) == 0 {
		if supplierName == "" {
			return fmt.Errorf("missing order items for email")
		}
		return fmt.Errorf("missing order items for %s", supplierName)
	}

	port, err := strconv.Atoi(m.port)
	if err != nil || port <= 0 {
		return fmt.Errorf("invalid SMTP_PORT: %s", m.port)
	}

	client, err := gomail.NewClient(
		m.host,
		gomail.WithPort(port),
		gomail.WithTLSPolicy(gomail.TLSMandatory),
		gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
		gomail.WithUsername(m.username),
		gomail.WithPassword(m.appPassword),
	)
	if err != nil {
		return fmt.Errorf("error creating go-mail client: %w", err)
	}

	message := gomail.NewMsg()
	if err := message.From(m.from); err != nil {
		return fmt.Errorf("error setting FROM address: %w", err)
	}
	if err := message.To(recipientEmail); err != nil {
		return fmt.Errorf("error setting TO address: %w", err)
	}
	message.Subject(placedOrderEmailSubject)

	document := placedOrderDocumentData{
		CompanyName:   supplierName,
		CurrentMonth:  currentOrderMonth(),
		CurrentDate:   currentOrderDate(),
		ContactName:   "Nguyễn Thành Trung",
		ContactDept:   "Khoa Trang bị",
		Items:         items,
	}

	pdfBytes, err := renderPlacedOrderAttachmentPDF(document)
	if err != nil {
		return fmt.Errorf("error rendering order PDF attachment: %w", err)
	}

	body, err := renderPlacedOrderEmailBody()
	if err != nil {
		return fmt.Errorf("error rendering email body: %w", err)
	}
	message.SetBodyString(gomail.TypeTextPlain, body)
	if err := message.AttachReader(
		placedOrderAttachmentName,
		bytes.NewReader(pdfBytes),
		gomail.WithFileContentType(placedOrderPDFContentType),
		gomail.WithFileName(placedOrderAttachmentName),
	); err != nil {
		return fmt.Errorf("error attaching order PDF: %w", err)
	}

	if err := client.DialAndSend(message); err != nil {
		if supplierName == "" {
			return fmt.Errorf("error sending email to %s: %w", recipientEmail, err)
		}
		return fmt.Errorf("error sending email to %s (%s): %w", supplierName, recipientEmail, err)
	}

	return nil
}

type placedOrderDocumentData struct {
	CompanyName  string
	CurrentMonth string
	CurrentDate  string
	ContactName  string
	ContactDept  string
	Items        []OrderEmailItem
}

func renderPlacedOrderEmailBody() (string, error) {
	return placedOrderEmailBody, nil
}

func renderPlacedOrderAttachmentPDF(data placedOrderDocumentData) ([]byte, error) {
	regularFontPath, boldFontPath, err := resolveOrderPDFFontPaths()
	if err != nil {
		return nil, err
	}
	regularFontBytes, err := os.ReadFile(regularFontPath)
	if err != nil {
		return nil, fmt.Errorf("error reading regular PDF font: %w", err)
	}
	boldFontBytes, err := os.ReadFile(boldFontPath)
	if err != nil {
		return nil, fmt.Errorf("error reading bold PDF font: %w", err)
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)
	pdf.SetTitle(placedOrderEmailSubject, true)
	pdf.SetAuthor("Khoa Trang bị Bệnh viện Quân đội TW 108", true)
	pdf.SetCreator("BV108 Consumables Management Backend", true)
	pdf.SetSubject("Đơn đặt hàng vật tư", true)
	pdf.SetKeywords("đơn đặt hàng vật tư BV108", true)
	pdf.SetCreationDate(time.Now())

	pdf.AddUTF8FontFromBytes(orderPDFFontFamily, "", regularFontBytes)
	pdf.AddUTF8FontFromBytes(orderPDFFontFamily, "B", boldFontBytes)
	pdf.AddPage()

	renderPlacedOrderPDFHeader(pdf, data)
	renderPlacedOrderPDFTable(pdf, data.Items)
	renderPlacedOrderPDFFooter(pdf, data)

	var buffer bytes.Buffer
	if err := pdf.Output(&buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func renderPlacedOrderPDFHeader(pdf *gofpdf.Fpdf, data placedOrderDocumentData) {
	pageWidth, _ := pdf.GetPageSize()
	leftMargin, _, rightMargin, _ := pdf.GetMargins()
	contentWidth := pageWidth - leftMargin - rightMargin
	leftWidth := contentWidth * 0.45
	rightWidth := contentWidth - leftWidth

	pdf.SetFont(orderPDFFontFamily, "B", orderPDFBodyFontSize)
	pdf.CellFormat(leftWidth, 6, "BỆNH VIỆN TWQĐ 108", "", 0, "C", false, 0, "")
	pdf.CellFormat(rightWidth, 6, "CỘNG HÒA XÃ HỘI CHỦ NGHĨA VIỆT NAM", "", 1, "C", false, 0, "")
	pdf.CellFormat(leftWidth, 6, "KHOA TRANG BỊ", "", 0, "C", false, 0, "")
	pdf.SetFont(orderPDFFontFamily, "", orderPDFBodyFontSize)
	pdf.CellFormat(rightWidth, 6, "Độc lập - Tự do - Hạnh phúc", "", 1, "C", false, 0, "")
	pdf.Ln(orderPDFSectionSpacing + 2)

	pdf.SetFont(orderPDFFontFamily, "B", orderPDFBodyFontSize)
	pdf.CellFormat(0, 8, placedOrderEmailSubject, "", 1, "C", false, 0, "")
	pdf.Ln(orderPDFParagraphSpacing)

	pdf.SetFont(orderPDFFontFamily, "", orderPDFBodyFontSize)

	if data.CompanyName == "" {
		pdf.MultiCell(0, orderPDFBodyLineHeight, "Kính gửi Quý công ty,", "", "L", false)
	} else {
		pdf.Write(orderPDFBodyLineHeight, "Kính gửi Quý công ty ")
		pdf.SetFont(orderPDFFontFamily, "B", orderPDFBodyFontSize)
		pdf.Write(orderPDFBodyLineHeight, data.CompanyName)
		pdf.SetFont(orderPDFFontFamily, "", orderPDFBodyFontSize)
		pdf.Write(orderPDFBodyLineHeight, ",")
		pdf.Ln(orderPDFBodyLineHeight)
	}
	pdf.Ln(orderPDFParagraphSpacing)
	pdf.MultiCell(0, orderPDFBodyLineHeight, fmt.Sprintf("Khoa Trang bị- BV Quân đội TW 108 xin gửi đến Quý công ty đơn đặt hàng vật tư tháng %s như sau:", data.CurrentMonth), "", "L", false)
	pdf.Ln(orderPDFParagraphSpacing)
}

func renderPlacedOrderPDFTable(pdf *gofpdf.Fpdf, items []OrderEmailItem) {
	leftMargin, _, _, _ := pdf.GetMargins()
	_, pageHeight := pdf.GetPageSize()
	bottomMargin := 15.0
	tableWidths := []float64{12, 62, 28, 24, 20, 34}
	lineHeight := 5.5
	cellPadding := 1.2

	pdf.SetFont(orderPDFFontFamily, "B", orderPDFBodyFontSize)
	pdf.MultiCell(0, orderPDFBodyLineHeight, "1. Danh sách vật tư đặt hàng:", "", "L", false)
	pdf.Ln(orderPDFParagraphSpacing - 1)

	drawTableHeader := func() {
		drawValues := []string{"STT", "Tên vật tư", "Mã vật tư", "Đơn vị tính", "Số lượng", "Đợt giao hàng"}
		pdf.SetFont(orderPDFFontFamily, "B", orderPDFTableFontSize)
		x := leftMargin
		y := pdf.GetY()
		for index, value := range drawValues {
			width := tableWidths[index]
			pdf.Rect(x, y, width, 8, "")
			pdf.SetXY(x+cellPadding, y+cellPadding)
			pdf.MultiCell(width-(cellPadding*2), 5.5, value, "", "C", false)
			x += width
			pdf.SetXY(x, y)
		}
		pdf.SetXY(leftMargin, y+8)
	}

	drawRow := func(values []string, bold bool) {
		if bold {
			pdf.SetFont(orderPDFFontFamily, "B", orderPDFTableFontSize)
		} else {
			pdf.SetFont(orderPDFFontFamily, "", orderPDFTableFontSize)
		}

		maxLineCount := 1
		for index, value := range values {
			lines := wrapOrderPDFTableText(pdf, value, tableWidths[index]-(cellPadding*2))
			if len(lines) > maxLineCount {
				maxLineCount = len(lines)
			}
		}

		rowHeight := float64(maxLineCount)*lineHeight + (cellPadding * 2)
		if pdf.GetY()+rowHeight > pageHeight-bottomMargin {
			pdf.AddPage()
			drawTableHeader()
			if bold {
				pdf.SetFont(orderPDFFontFamily, "B", orderPDFTableFontSize)
			} else {
				pdf.SetFont(orderPDFFontFamily, "", orderPDFTableFontSize)
			}
		}

		x := leftMargin
		y := pdf.GetY()
		alignments := []string{"C", "C", "C", "C", "C", "C"}

		for index, value := range values {
			width := tableWidths[index]
			pdf.Rect(x, y, width, rowHeight, "")
			pdf.SetXY(x+cellPadding, y+cellPadding)
			pdf.MultiCell(width-(cellPadding*2), lineHeight, value, "", alignments[index], false)
			x += width
			pdf.SetXY(x, y)
		}

		pdf.SetXY(leftMargin, y+rowHeight)
	}

	drawTableHeader()

	for _, item := range items {
		drawRow([]string{
			strconv.Itoa(item.Index),
			nonEmptyPDFText(item.TenVatTu),
			nonEmptyPDFText(item.MaVatTu),
			nonEmptyPDFText(item.DonViTinh),
			strconv.Itoa(item.SoLuong),
			nonEmptyPDFText(item.DotGiaoHang),
		}, false)
	}

	if len(items) == 0 {
		drawRow([]string{"1", "...", "...", "...", "...", "..."}, false)
	}

	pdf.Ln(orderPDFSectionSpacing - 1)
}

func renderPlacedOrderPDFFooter(pdf *gofpdf.Fpdf, data placedOrderDocumentData) {
	pdf.SetFont(orderPDFFontFamily, "B", orderPDFBodyFontSize)
	pdf.MultiCell(0, orderPDFBodyLineHeight, fmt.Sprintf("2. Thông tin liên hệ: %s – %s", data.ContactName, data.ContactDept), "", "L", false)
	pdf.Ln(orderPDFParagraphSpacing)

	pdf.SetFont(orderPDFFontFamily, "", orderPDFBodyFontSize)
	pdf.MultiCell(0, orderPDFBodyLineHeight, "Kính mong Quý công ty xác nhận lại đơn hàng, và tiến hành giao hàng theo đúng đợt giao hàng đã nêu. Trân trọng cảm ơn sự hợp tác của Quý công ty!", "", "L", false)
	pdf.Ln(orderPDFSectionSpacing)
	pdf.CellFormat(0, orderPDFBodyLineHeight, "Trân trọng,", "", 1, "L", false, 0, "")
	pdf.Ln(orderPDFParagraphSpacing - 1)
	pdf.SetFont(orderPDFFontFamily, "B", orderPDFBodyFontSize)
	pdf.CellFormat(0, orderPDFBodyLineHeight, fmt.Sprintf("%s – %s.", data.ContactName, data.ContactDept), "", 1, "R", false, 0, "")
}

func resolveOrderPDFFontPaths() (string, string, error) {
	regularOverride := strings.TrimSpace(os.Getenv(orderPDFFontRegularEnv))
	boldOverride := strings.TrimSpace(os.Getenv(orderPDFFontBoldEnv))
	if regularOverride != "" || boldOverride != "" {
		switch {
		case regularOverride == "":
			return "", "", fmt.Errorf("%s is required when %s is set", orderPDFFontRegularEnv, orderPDFFontBoldEnv)
		case boldOverride == "":
			return "", "", fmt.Errorf("%s is required when %s is set", orderPDFFontBoldEnv, orderPDFFontRegularEnv)
		case !fileExists(regularOverride):
			return "", "", fmt.Errorf("configured regular PDF font does not exist: %s", regularOverride)
		case !fileExists(boldOverride):
			return "", "", fmt.Errorf("configured bold PDF font does not exist: %s", boldOverride)
		default:
			return regularOverride, boldOverride, nil
		}
	}

	fontIndex := indexOrderPDFFonts([]string{
		"/usr/share/fonts",
		"/usr/local/share/fonts",
		"/System/Library/Fonts",
		"/System/Library/Fonts/Supplemental",
		"/Library/Fonts",
		"C:\\Windows\\Fonts",
		"/mnt/c/Windows/Fonts",
	})

	candidatePairs := []struct {
		regular []string
		bold    []string
	}{
		{
			regular: []string{"times.ttf", "Times.ttf", "times new roman.ttf", "Times New Roman.ttf"},
			bold:    []string{"timesbd.ttf", "Timesbd.ttf", "times new roman bold.ttf", "Times New Roman Bold.ttf"},
		},
		{
			regular: []string{"DejaVuSans.ttf"},
			bold:    []string{"DejaVuSans-Bold.ttf"},
		},
		{
			regular: []string{"Arial.ttf", "arial.ttf"},
			bold:    []string{"Arialbd.ttf", "arialbd.ttf"},
		},
		{
			regular: []string{"LiberationSans-Regular.ttf"},
			bold:    []string{"LiberationSans-Bold.ttf"},
		},
		{
			regular: []string{"NotoSans-Regular.ttf"},
			bold:    []string{"NotoSans-Bold.ttf"},
		},
		{
			regular: []string{"FreeSans.ttf"},
			bold:    []string{"FreeSansBold.ttf", "FreeSans-Bold.ttf"},
		},
	}

	for _, pair := range candidatePairs {
		regular := lookupOrderPDFFont(fontIndex, pair.regular)
		bold := lookupOrderPDFFont(fontIndex, pair.bold)
		if regular != "" && bold != "" {
			return regular, bold, nil
		}
	}

	return "", "", fmt.Errorf(
		"unable to find a supported Unicode font pair for PDF generation. Times New Roman is preferred. Set %s and %s to valid .ttf files if needed",
		orderPDFFontRegularEnv,
		orderPDFFontBoldEnv,
	)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func indexOrderPDFFonts(searchDirs []string) map[string]string {
	fontIndex := make(map[string]string)
	for _, dir := range searchDirs {
		dir = strings.TrimSpace(dir)
		if dir == "" || !fileExists(dir) && !dirExists(dir) {
			continue
		}

		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d == nil || d.IsDir() {
				return nil
			}
			if !strings.EqualFold(filepath.Ext(d.Name()), ".ttf") {
				return nil
			}

			key := strings.ToLower(d.Name())
			if _, exists := fontIndex[key]; !exists {
				fontIndex[key] = path
			}
			return nil
		})
	}
	return fontIndex
}

func lookupOrderPDFFont(fontIndex map[string]string, candidates []string) string {
	for _, candidate := range candidates {
		if path, ok := fontIndex[strings.ToLower(strings.TrimSpace(candidate))]; ok {
			return path
		}
	}
	return ""
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func nonEmptyPDFText(value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return "-"
	}
	return text
}

func wrapOrderPDFTableText(pdf *gofpdf.Fpdf, value string, width float64) []string {
	text := nonEmptyPDFText(value)
	lines := pdf.SplitText(text, width)
	if len(lines) == 0 {
		return []string{text}
	}
	return lines
}

func currentOrderMonth() string {
	location, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		now := time.Now()
		return fmt.Sprintf("%02d/%d", int(now.Month()), now.Year())
	}

	now := time.Now().In(location)
	return fmt.Sprintf("%02d/%d", int(now.Month()), now.Year())
}

func currentOrderDate() string {
	location, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		now := time.Now()
		return fmt.Sprintf("%02d/%02d/%d", now.Day(), int(now.Month()), now.Year())
	}

	now := time.Now().In(location)
	return fmt.Sprintf("%02d/%02d/%d", now.Day(), int(now.Month()), now.Year())
}
