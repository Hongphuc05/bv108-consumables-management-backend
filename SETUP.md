# HƯỚNG DẪN CHẠY BACKEND API - BV108

## 📝 Các Bước Thực Hiện

### 1️⃣ Cài Đặt Go (nếu chưa có)

Download và cài đặt Go từ: https://go.dev/dl/

Kiểm tra Go đã được cài đặt:
```bash
go version
```

### 2️⃣ Cấu Hình Database

Tạo file `.env` từ `.env.example`:
```bash
cd backend
copy .env.example .env
```

Sửa thông tin trong file `.env`:
```env
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=YOUR_MYSQL_PASSWORD
DB_NAME=hospital_db

SERVER_PORT=8080
GIN_MODE=debug

FRONTEND_URL=http://localhost:5173
```

### 3️⃣ Cài Đặt Dependencies

```bash
cd backend
go mod download
```

Nếu gặp lỗi, chạy:
```bash
go mod tidy
```

### 4️⃣ Chạy Server

```bash
go run cmd/server/main.go
```

Hoặc build rồi chạy:
```bash
go build -o server.exe cmd/server/main.go
./server.exe
```

### 5️⃣ Test API

Mở trình duyệt và truy cập:
- Health check: http://localhost:8080/health
- Lấy danh sách vật tư: http://localhost:8080/api/supplies
- Lấy nhóm vật tư: http://localhost:8080/api/supplies/groups

## 🧪 Test với Postman hoặc curl

### Lấy tất cả vật tư (có phân trang)
```bash
curl http://localhost:8080/api/supplies?page=1&pageSize=20
```

### Tìm kiếm vật tư
```bash
curl "http://localhost:8080/api/supplies/search?keyword=miếng"
```

### Lấy vật tư theo nhóm
```bash
curl "http://localhost:8080/api/supplies/group?groupName=Nhóm A"
```

### Lấy vật tư tồn kho thấp
```bash
curl http://localhost:8080/api/supplies/low-stock?threshold=20
```

## ❗ Xử Lý Lỗi Thường Gặp

### Lỗi: "Failed to initialize database"
➡️ Kiểm tra:
- MySQL đã chạy chưa
- Thông tin trong `.env` đúng chưa
- Database `hospital_db` đã tồn tại chưa
- User có quyền truy cập database không

### Lỗi: "panic: runtime error"
➡️ Kiểm tra:
- Bảng `supplies` đã tồn tại trong database chưa
- Cấu trúc bảng đúng với schema không

### Lỗi: "bind: address already in use"
➡️ Port 8080 đã được sử dụng:
- Thay đổi `SERVER_PORT` trong `.env`
- Hoặc kill process đang chạy port 8080

## 🔗 Tích Hợp với Frontend

Sau khi backend chạy thành công, cập nhật frontend để gọi API:

1. Tạo file `src/services/api.ts` trong frontend
2. Thay thế mockData bằng API calls
3. Xử lý loading states và errors

## 📊 Kiểm Tra Dữ Liệu

Đảm bảo database có dữ liệu:
```sql
USE hospital_db;
SELECT COUNT(*) FROM supplies;
SELECT * FROM supplies LIMIT 5;
```

## 🚀 Production Deployment

Khi deploy lên production:
1. Đổi `GIN_MODE=release` trong `.env`
2. Build binary: `go build -o server.exe cmd/server/main.go`
3. Chạy binary thay vì `go run`
4. Sử dụng reverse proxy (nginx)
5. Setup HTTPS

---

**Chúc bạn thành công! 🎉**
