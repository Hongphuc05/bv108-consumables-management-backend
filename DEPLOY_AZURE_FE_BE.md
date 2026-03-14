# Deploy FE + BE lên Azure Container Apps

Tài liệu này deploy:
- Backend Go (`bv108-consumables-management-backend`) lên Azure Container Apps.
- Frontend Vite (`../bv108-consumables-management`) lên Azure Container Apps.

## 1. Chuẩn bị

- Azure CLI đã cài.
- Đã đăng nhập Azure: `az login`
- Có quyền tạo resource group, ACR, Container Apps.

Biến bắt buộc:

```bash
export DB_HOST="bv108.mysql.database.azure.com"
export DB_USER="bv108"
export DB_PASSWORD="<your-db-password>"
export DB_NAME="hospital_db"
```

Biến tùy chọn:

```bash
export RG="bv108-rg"
export LOCATION="southeastasia"
export ACA_ENV_NAME="bv108-aca-env"
export ACR_NAME="bv108acrxxxxxx"          # nếu bỏ trống script tự tạo
export BACKEND_APP_NAME="bv108-backend"
export FRONTEND_APP_NAME="bv108-frontend"
export BUILD_MODE="docker"                 # docker (khuyến nghị cho Azure for Students) hoặc acr_tasks
export DB_PORT="3306"
export DB_TLS="true"
export GIN_MODE="release"
export JWT_SECRET="replace-this-secret"

# Optional cho frontend build:
export VITE_GEMINI_API_KEY=""
export VITE_GEMINI_MODEL="gemini-2.5-flash-lite"
export VITE_GEMINI_WEB_SEARCH="true"
```

Nếu frontend không nằm ở path mặc định `../bv108-consumables-management`, set thêm:

```bash
export FRONTEND_DIR="/absolute/path/to/bv108-consumables-management"
```

## 2. Chạy script deploy

Từ thư mục backend:

```bash
cd /path/to/bv108-consumables-management-backend
./scripts/deploy_aca_fe_be.sh
```

Script sẽ tự load biến từ file `.env` (mặc định).  
Nếu muốn dùng file khác:

```bash
ENV_FILE=/path/to/your.env ./scripts/deploy_aca_fe_be.sh
```

Script cũng tự đọc `VITE_GEMINI_API_KEY`, `VITE_GEMINI_MODEL`, `VITE_GEMINI_WEB_SEARCH`
từ `../bv108-consumables-management/.env` (hoặc `FRONTEND_ENV_FILE`) nếu các biến này chưa có trong `ENV_FILE`.

Script sẽ làm:
- Tạo `Resource Group`, `ACR`, `Container Apps Environment`.
- Build/push backend image.
- `BUILD_MODE=docker`: build/push bằng Docker local (không dùng ACR Tasks).
- `BUILD_MODE=acr_tasks`: build/push bằng `az acr build`.
- Deploy backend app (public ingress, port 8080).
- Lấy backend URL và build frontend image với `VITE_API_URL=<backend>/api`.
- Deploy frontend app (public ingress, port 8080).
- Update `FRONTEND_URL` cho backend để CORS đúng domain FE.

## 3. Test sau deploy

- Health check backend:
```bash
curl https://<backend-fqdn>/health
```

- Mở frontend URL:
```text
https://<frontend-fqdn>
```

## 4. Lưu ý quan trọng

- `DB_USER` phải là `bv108` (không phải `bv108@bv108`).
- `DB_TLS` nên là `true` cho Azure MySQL.
- Backend đọc `PORT` tự động (đã hỗ trợ trong `config/config.go`).
- FE dùng `VITE_API_URL` nên không còn hardcode localhost.

## 5. Nếu app đã tồn tại

Script này dành cho deploy mới (initial deploy).  
Nếu app đã tồn tại, đổi tên app hoặc xóa app cũ trước khi chạy lại.
