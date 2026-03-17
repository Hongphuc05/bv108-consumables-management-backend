Backend
Mở terminal tại thư mục backend:
D:\Projects\bv-108\bv108-consumables-management-backend
Cài dependencies Go:
go mod download
Chuẩn bị file môi trường:
dùng file .env riêng của họ
phải trỏ đúng Azure DB nếu team đang dùng Azure
Chạy server:
go run cmd/server/main.go

Kiểm tra sống:
http://localhost:8080/health

Frontend

Mở terminal tại thư mục frontend:
D:\Projects\bv-108\bv108-consumables-management

Cài packages:
npm install

Chạy dev:
npm run dev

Frontend sẽ gọi backend qua VITE_API_URL trong .env frontend.

Nếu chạy production build

Backend build:
go build -o server.exe cmd/server/main.go

Chạy backend binary:
server.exe

Frontend build:
npm run build

Serve thư mục dist bằng nginx hoặc static host.

Lưu ý quan trọng sau pull

Không dùng local DB nếu team thống nhất Azure.

Không commit file .env thật lên git.

Nếu máy bị Defender chặn go run, thêm exclusion cho:

C:\Users\User\AppData\Local\go-build
C:\Users\User\AppData\Local\Temp\go-build