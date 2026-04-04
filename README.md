# BV108 Consumables Management - Backend API

Backend API cho hệ thống quản lý vật tư bệnh viện BV108, được xây dựng bằng **Golang** và **Gin Framework**.

## 📋 Yêu Cầu Hệ Thống

- Go 1.21 hoặc cao hơn
- MySQL 8.0 hoặc cao hơn
- Database: `hospital_db`
- Table: `supplies`

## 🚀 Cài Đặt và Chạy

### Bước 1: Cài đặt Go dependencies

```bash
cd backend
go mod download
```

### Bước 2: Cấu hình Database

1. Copy file `.env.example` thành `.env`:
```bash
copy .env.example .env
```

2. Sửa thông tin kết nối MySQL trong file `.env`:
```env
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password_here
DB_NAME=hospital_db

SERVER_PORT=8080
GIN_MODE=debug

FRONTEND_URL=http://localhost:5173
```

### Bước 3: Chạy Server

```bash
go run cmd/server/main.go
```

Server sẽ chạy tại: `http://localhost:8080`

## 📡 API Endpoints

### Health Check
```
GET /health
```
Kiểm tra trạng thái server

### Supplies Endpoints

#### 1. Lấy tất cả vật tư (phân trang)
```
GET /api/supplies?page=1&pageSize=20
```
**Query Parameters:**
- `page` (optional): Số trang, mặc định = 1
- `pageSize` (optional): Số item mỗi trang, mặc định = 20

**Response:**
```json
{
  "data": [...],
  "page": 1,
  "pageSize": 20,
  "total": 150,
  "totalPages": 8
}
```

#### 2. Lấy chi tiết một vật tư
```
GET /api/supplies/:id
```
**Parameters:**
- `id`: IDX1 của vật tư

#### 3. Tìm kiếm vật tư
```
GET /api/supplies/search?keyword=miếng&page=1&pageSize=20
```
**Query Parameters:**
- `keyword` (required): Từ khóa tìm kiếm (tìm theo NAME hoặc ID)
- `page` (optional): Số trang
- `pageSize` (optional): Số item mỗi trang

#### 4. Lấy danh sách nhóm vật tư
```
GET /api/supplies/groups
```
**Response:**
```json
{
  "groups": ["Nhóm A", "Nhóm B", ...],
  "total": 10
}
```

#### 5. Lấy vật tư theo nhóm
```
GET /api/supplies/group?groupName=Nhóm A&page=1&pageSize=20
```
**Query Parameters:**
- `groupName` (required): Tên nhóm vật tư
- `page` (optional): Số trang
- `pageSize` (optional): Số item mỗi trang

#### 6. Lấy vật tư tồn kho thấp
```
GET /api/supplies/low-stock?threshold=20&page=1&pageSize=20
```
**Query Parameters:**
- `threshold` (optional): Ngưỡng tồn kho, mặc định = 20
- `page` (optional): Số trang
- `pageSize` (optional): Số item mỗi trang

## 🗂️ Cấu Trúc Dự Án

```
backend/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── database/
│   │   └── db.go                # Database connection
│   ├── handlers/
│   │   └── supply_handler.go   # HTTP handlers
│   └── models/
│       └── supply.go            # Data models & repository
├── config/
│   └── config.go                # Configuration management
├── .env.example                 # Environment variables template
├── .gitignore
├── go.mod
└── README.md
```

## 📊 Database Schema

Bảng `supplies` trong database `hospital_db`:

| Column         | Type          | Description                    |
|----------------|---------------|--------------------------------|
| IDX1           | int           | Primary key                    |
| PRODUCTID      | int           | Product ID                     |
| GROUPNAME      | varchar(255)  | Tên nhóm vật tư                |
| ID             | varchar(100)  | Mã vật tư                      |
| IDX2           | varchar(100)  | Index phụ                      |
| TYPENAME       | varchar(255)  | Tên loại                       |
| NAME           | varchar(255)  | Tên vật tư                     |
| UNIT           | varchar(50)   | Đơn vị tính                    |
| THONG_TIN_THAU | text          | Thông tin thầu                 |
| TONGTHAU       | varchar(255)  | Tổng thầu                      |
| HANGSX         | varchar(255)  | Hãng sản xuất                  |
| NUOC_SX        | varchar(255)  | Nước sản xuất                  |
| NHA_CUNG_CAP   | text          | Nhà cung cấp                   |
| PRICE          | decimal(18,2) | Đơn giá                        |
| TONDAUKY       | int           | Tồn đầu kỳ                     |
| NHAPTRONGKY    | int           | Nhập trong kỳ                  |
| XUATTRONGKY    | int           | Xuất trong kỳ                  |
| TONGNHAP       | int           | Tổng nhập                      |

**Calculated Field:**
- `TonCuoiKy = TONDAUKY + NHAPTRONGKY - XUATTRONGKY`

## 🔧 Development

### Build Production
```bash
go build -o server.exe cmd/server/main.go
```

### Run Production Build
```bash
./server.exe
```

## 🐛 Troubleshooting

### Lỗi kết nối database
- Kiểm tra MySQL đã chạy chưa
- Kiểm tra thông tin trong file `.env`
- Kiểm tra database `hospital_db` đã tồn tại chưa

### Lỗi CORS
- Kiểm tra `FRONTEND_URL` trong `.env`
- Thêm domain của frontend vào CORS config trong `main.go`

## ☁️ Deploy Azure

Hướng dẫn deploy FE + BE thủ công lên Azure Container Apps nằm tại:

`DEPLOY_AZURE_FE_BE.md`

Tài liệu này ưu tiên đường deploy đã chạy ổn trên máy hiện tại:
- `PowerShell` Windows
- `docker build` + `docker push`
- `az containerapp update`

## 📝 Notes

- Server mặc định chạy ở port 8080
- Tất cả responses đều có format JSON
- API hỗ trợ pagination cho danh sách lớn
- TonCuoiKy được tính tự động từ công thức: TonDauKy + NhapTrongKy - XuatTrongKy

## 🔒 Security Notes

- Không commit file `.env` lên Git
- Thay đổi password database trước khi deploy production
- Sử dụng HTTPS trong production
- Thêm authentication/authorization nếu cần

## 📞 Support

Nếu gặp vấn đề, vui lòng tạo issue hoặc liên hệ team.




đang test
