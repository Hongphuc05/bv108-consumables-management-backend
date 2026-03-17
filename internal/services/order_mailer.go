package services

import (
	"bytes"
	"fmt"
	"html/template"
	stdmail "net/mail"
	"strconv"
	"strings"
	"time"

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
	message.Subject("ĐƠN ĐẶT HÀNG VẬT TƯ TẠI BỆNH VIỆN QUÂN ĐỘI TW 108")

	body, err := renderPlacedOrderEmailBody(orderEmailTemplateData{
		CompanyName:   supplierName,
		CurrentMonth:  currentOrderMonth(),
		ContactName:   "Nguyễn Thành Trung",
		ContactDept:   "Khoa Trang bị",
		Items:         items,
	})
	if err != nil {
		return fmt.Errorf("error rendering email body: %w", err)
	}
	message.SetBodyString(gomail.TypeTextHTML, body)

	if err := client.DialAndSend(message); err != nil {
		if supplierName == "" {
			return fmt.Errorf("error sending email to %s: %w", recipientEmail, err)
		}
		return fmt.Errorf("error sending email to %s (%s): %w", supplierName, recipientEmail, err)
	}

	return nil
}

type orderEmailTemplateData struct {
	CompanyName  string
	CurrentMonth string
	ContactName  string
	ContactDept  string
	Items        []OrderEmailItem
}

var placedOrderEmailTemplate = template.Must(template.New("placed_order_email").Parse(`
<!doctype html>
<html lang="vi">
<head>
  <meta charset="UTF-8" />
  <title>ĐƠN ĐẶT HÀNG VẬT TƯ TẠI BỆNH VIỆN QUÂN ĐỘI TW 108</title>
</head>
<body style="font-family: Arial, Helvetica, sans-serif; font-size: 14px; color: #111827; line-height: 1.6; margin: 0; padding: 24px; background-color: #ffffff;">
  <div style="max-width: 860px; margin: 0 auto;">
    <div style="text-align: center; margin-bottom: 20px;">
      <div style="font-weight: 700;">BỆNH VIỆN TWQĐ 108</div>
      <div style="font-weight: 700;">KHOA TRANG BỊ</div>
      <div style="font-weight: 700; margin-top: 12px;">CỘNG HÒA XÃ HỘI CHỦ NGHĨA VIỆT NAM</div>
      <div>Độc lập - Tự do - Hạnh phúc</div>
    </div>

    <h2 style="text-align: center; margin: 0 0 20px; font-size: 20px;">ĐƠN ĐẶT HÀNG VẬT TƯ TẠI BỆNH VIỆN QUÂN ĐỘI TW 108</h2>

    <p style="margin: 0 0 12px;">Kính gửi Quý công ty <strong>{{.CompanyName}}</strong>,</p>
    <p style="margin: 0 0 16px;">Khoa Trang bị - BV Quân đội TW 108 xin gửi đến Quý công ty đơn đặt hàng vật tư tháng <strong>{{.CurrentMonth}}</strong> như sau:</p>

    <p style="margin: 0 0 10px;"><strong>1. Danh sách vật tư đặt hàng:</strong></p>
    <table style="width: 100%; border-collapse: collapse; margin-bottom: 16px;">
      <thead>
        <tr>
          <th style="border: 1px solid #111827; padding: 8px; background: #f3f4f6;">STT</th>
          <th style="border: 1px solid #111827; padding: 8px; background: #f3f4f6;">Tên vật tư</th>
          <th style="border: 1px solid #111827; padding: 8px; background: #f3f4f6;">Mã vật tư</th>
          <th style="border: 1px solid #111827; padding: 8px; background: #f3f4f6;">Đơn vị tính</th>
          <th style="border: 1px solid #111827; padding: 8px; background: #f3f4f6;">Số lượng</th>
        </tr>
      </thead>
      <tbody>
        {{range .Items}}
        <tr>
          <td style="border: 1px solid #111827; padding: 8px; text-align: center;">{{.Index}}</td>
          <td style="border: 1px solid #111827; padding: 8px;">{{.TenVatTu}}</td>
          <td style="border: 1px solid #111827; padding: 8px;">{{.MaVatTu}}</td>
          <td style="border: 1px solid #111827; padding: 8px;">{{.DonViTinh}}</td>
          <td style="border: 1px solid #111827; padding: 8px; text-align: right;">{{.SoLuong}}</td>
        </tr>
        {{end}}
      </tbody>
    </table>

    <p style="margin: 0 0 8px;"><strong>2. Thông tin liên hệ:</strong> {{.ContactName}} – {{.ContactDept}}</p>
    <p style="margin: 0 0 12px;">Kính mong Quý công ty xác nhận lại đơn hàng và tiến hành giao hàng theo đúng số lượng đã nêu. Trân trọng cảm ơn sự hợp tác của Quý công ty!</p>
    <p style="margin: 0;">Trân trọng,</p>
    <p style="margin: 4px 0 0;"><strong>{{.ContactName}} – {{.ContactDept}}</strong></p>
  </div>
</body>
</html>
`))

func renderPlacedOrderEmailBody(data orderEmailTemplateData) (string, error) {
	var buffer bytes.Buffer
	if err := placedOrderEmailTemplate.Execute(&buffer, data); err != nil {
		return "", err
	}
	return buffer.String(), nil
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
