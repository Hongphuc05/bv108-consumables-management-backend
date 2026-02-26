# UBot API Client - Hướng dẫn sử dụng

Client Python để kết nối với UBot API và lấy thông tin hóa đơn điện tử.

## 📋 Tài liệu tham khảo
- [Tài liệu API chính thức](https://help.ubot.vn/tai-lieu-ket-noi-api/)

## 🚀 Cài đặt

### 1. Cài đặt thư viện cần thiết
```bash
pip install -r requirements.txt
```

### 2. Cấu hình thông tin đăng nhập
Liên hệ với UBot để nhận:
- Username (email đăng nhập)
- Password
- Company ID

## 💡 Sử dụng

### Ví dụ cơ bản

```python
from ubot_client import UBotAPIClient

# Khởi tạo client
client = UBotAPIClient(
    username="your-email@example.com",
    password="your-password",
    is_production=False  # False = môi trường test, True = production
)

# Đăng nhập
client.login(remember_me=False)

# Lấy danh sách hóa đơn
invoices = client.get_invoices(
    page=0,
    size=10,
    invoice_types=["INPUT_ELECTRONIC_INVOICE"],
    get_attachments=True
)

# Xử lý kết quả
for invoice in invoices.get("invoices", []):
    print(f"Số HĐ: {invoice['invoiceNo']}")
    print(f"Bên bán: {invoice['sellerName']}")
    print(f"Tổng tiền: {invoice['grandTotal']}")
```

### Các loại hóa đơn (Invoice Types)
- `INPUT_PAPER_INVOICE`: Hóa đơn đầu vào (giấy)
- `INPUT_ELECTRONIC_INVOICE`: Hóa đơn đầu vào (điện tử)
- `OUTPUT_ELECTRONIC_INVOICE`: Hóa đơn đầu ra (điện tử)
- `INPUT_PAPER_INVOICE_OCR`: Hóa đơn đầu vào (giấy) đọc OCR
- `INPUT_ELECTRONIC_INVOICE_OCR`: Hóa đơn đầu vào (điện tử) đọc OCR
- `OUTPUT_ELECTRONIC_INVOICE_OCR`: Hóa đơn đầu ra (điện tử) đọc OCR

### Trạng thái hóa đơn (Invoice Status)
- `VALID`: Hợp lệ
- `INVALID`: Không hợp lệ
- `IS_WAITING`: Chờ xác thực
- `IS_RECHECKING`: Chờ xác thực lại

## 🔍 Các tính năng chính

### 1. Lấy danh sách hóa đơn
```python
invoices = client.get_invoices(
    page=0,
    size=10,
    sort="receivedDate",  # hoặc "invoiceReleaseDate"
    invoice_types=["INPUT_ELECTRONIC_INVOICE"],
    invoice_status="VALID",
    get_matching_data=True,  # Lấy dữ liệu danh mục
    get_attachments=True,    # Lấy link download file
    get_taxes=True           # Lấy chi tiết thuế
)
```

### 2. Tìm hóa đơn theo số
```python
invoice = client.get_invoices(
    invoice_no="0001234",
    size=1
)
```

### 3. Tìm hóa đơn theo MST bên bán
```python
invoices = client.get_invoices(
    seller_tax_no="0123456789",
    size=50
)
```

### 4. Lấy hóa đơn theo khoảng thời gian
```python
invoices = client.get_invoices(
    received_date_from="01/01/2024",
    received_date_to="31/12/2024",
    size=100
)
```

### 5. Gửi hóa đơn lên UBot
```python
result = client.send_invoice(
    transaction_id="TXN001",
    company_id="your-company-id",
    files=["file1.pdf", "file2.xml"],
    sender="sender@example.com",
    title="Hóa đơn tháng 1"
)
```

### 6. Kiểm tra trạng thái giao dịch
```python
status = client.get_transaction_status(
    transaction_ids=["TXN001", "TXN002"]
)
```

## 📊 Cấu trúc dữ liệu hóa đơn

Mỗi hóa đơn trả về bao gồm:

```python
{
    "invoiceId": "uuid",
    "invoiceNo": "0001234",
    "modelNo": "01GTKT",
    "serial": "AA/21T",
    "sellerName": "Công ty ABC",
    "sellerTaxNo": "0123456789",
    "buyerName": "Công ty XYZ",
    "buyerTaxNo": "9876543210",
    "subTotal": 1000000,      # Tiền trước thuế
    "taxAmount": 100000,      # Tiền thuế
    "grandTotal": 1100000,    # Tổng tiền
    "currency": "VND",
    "invoiceReleaseDate": "2024-01-15",
    "receivedDate": "2024-01-16",
    "status": "VALID",
    "invoiceItems": [         # Chi tiết hàng hóa
        {
            "itemName": "Sản phẩm A",
            "itemQuantity": 10,
            "itemUnit": "Cái",
            "itemPrice": 100000,
            "itemSubTotal": 1000000
        }
    ]
}
```

## 🌐 Môi trường API

### Test/Development
```
https://portal-dev.ubot.vn/api
```

### Production
```
https://portal.ubot.vn/api
```

## 🔑 Token Authentication

- Token có thời gian hết hạn:
  - `rememberMe=False`: 30 phút
  - `rememberMe=True`: 30 ngày
- Khi token hết hạn, cần gọi lại `login()` để lấy token mới

## ⚠️ Giới hạn

- Số lượng hóa đơn tối đa mỗi request: **100**
- Số lượng file tối đa khi gửi hóa đơn: **15 PDF & XML**
- Định dạng ngày tháng: **dd/mm/yyyy**

## 📝 Ví dụ Response

### Thành công
```json
{
  "statusResponse": {
    "statusCode": 200,
    "errorCode": null,
    "message": "Success"
  },
  "metadata": {
    "page": 0,
    "size": 10,
    "total": 45
  },
  "invoices": [...]
}
```

### Lỗi
```json
{
  "statusResponse": {
    "statusCode": 400,
    "errorCode": "INV0005",
    "message": "Maximum size is 100"
  }
}
```

## 🐛 Xử lý lỗi

### Mã lỗi thường gặp

| Status Code | Mô tả |
|-------------|-------|
| 200 | Thành công |
| 400 | Bad Request (thiếu trường bắt buộc, sai kiểu dữ liệu) |
| 401 | Chưa đăng nhập hoặc sai token |
| 403 | Không có quyền truy cập |
| 500 | Lỗi server |

| Error Code | Message |
|------------|---------|
| INV0005 | Maximum size is 100 |
| INV0006 | The company does not belong to this account |

## 📞 Hỗ trợ

Liên hệ UBot để được hỗ trợ:
- Website: https://ubot.vn
- Tài liệu: https://help.ubot.vn

## 📄 License

Tài liệu này tuân theo điều khoản sử dụng của UBot.
