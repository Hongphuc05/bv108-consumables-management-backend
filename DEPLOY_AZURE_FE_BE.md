# Deploy FE + BE lên Azure Container Apps

Tài liệu này là hướng dẫn deploy thủ công để bạn có thể tự làm lại mà không cần dùng script.

Phạm vi:
- Backend Go: `bv108-consumables-management-backend`
- Frontend Vite: `bv108-consumables-management`
- Hạ tầng Azure đang dùng:
  - Resource Group: `bv108-rg`
  - Container Apps Environment: `bv108-aca-env`
  - Backend app: `bv108-backend`
  - Frontend app: `bv108-frontend`
  - ACR: `bv108acr509662`

## 1. Khi nào nên dùng cách này

Nên dùng deploy thủ công nếu:
- Bạn chỉ cần update code mới lên app đã tồn tại.
- `az acr build` bị lỗi `TasksOperationsNotAllowed`.
- Docker trong WSL không chạy ổn.
- Script `scripts/deploy_aca_fe_be.sh` lỗi do line ending `CRLF` hoặc path WSL/Windows.

Trên máy hiện tại, cách ổn định nhất là:
- Chạy bằng `PowerShell` trên Windows.
- Build/push bằng Docker Desktop Windows.
- Update app bằng Azure CLI Windows.

## 2. Chuẩn bị

Yêu cầu:
- Đã cài Azure CLI.
- Đã đăng nhập Azure: `az login`
- Đã cài Docker Desktop và Docker engine đang chạy.
- Có quyền với resource group, ACR, Container Apps.

Khuyến nghị trước khi deploy:
- Ở cả FE và BE, chạy `git status` để biết mình đang deploy code nào.
- Nếu muốn deploy đúng bản đã commit, commit trước rồi mới build.
- Nếu build từ local working tree thì Azure sẽ chạy đúng code local hiện tại, kể cả phần chưa commit.

## 3. Biến cần dùng

Không copy secret thật vào README. Điền giá trị thật trong terminal lúc deploy.

Giá trị backend lấy từ `bv108-consumables-management-backend/.env`:

```env
DB_HOST=...
DB_PORT=3306
DB_USER=...
DB_PASSWORD=...
DB_NAME=...
DB_TLS=true
JWT_SECRET=...
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=...
SMTP_APP_PASSWORD=...
SMTP_FROM=...

RG=bv108-rg
LOCATION=southeastasia
ACA_ENV_NAME=bv108-aca-env
BACKEND_APP_NAME=bv108-backend
FRONTEND_APP_NAME=bv108-frontend
ACR_NAME=bv108acr509662
```

Giá trị frontend lấy từ `bv108-consumables-management/.env`:

```env
VITE_GEMINI_API_KEY=
VITE_GEMINI_MODEL=gemini-2.5-flash-lite
VITE_GEMINI_WEB_SEARCH=true
```

## 4. Deploy thủ công bằng PowerShell

Mở `PowerShell` và đứng ở `D:\Projects\BV108`.

### 4.1. Khai báo biến

```powershell
$STAMP = Get-Date -Format "yyyyMMddHHmmss"

$RG = "bv108-rg"
$LOCATION = "southeastasia"
$ACA_ENV_NAME = "bv108-aca-env"
$ACR_NAME = "bv108acr509662"
$BACKEND_APP_NAME = "bv108-backend"
$FRONTEND_APP_NAME = "bv108-frontend"

$BACKEND_DIR = "D:\Projects\BV108\bv108-consumables-management-backend"
$FRONTEND_DIR = "D:\Projects\BV108\bv108-consumables-management"

$DB_HOST = "<db-host>"
$DB_PORT = "3306"
$DB_USER = "<db-user>"
$DB_PASSWORD = "<db-password>"
$DB_NAME = "hospital_db"
$DB_TLS = "true"

$JWT_SECRET = "<jwt-secret>"
$SMTP_HOST = "smtp.gmail.com"
$SMTP_PORT = "587"
$SMTP_USERNAME = "<smtp-username>"
$SMTP_APP_PASSWORD = "<smtp-app-password>"
$SMTP_FROM = "<smtp-from>"

$VITE_GEMINI_API_KEY = "<optional>"
$VITE_GEMINI_MODEL = "gemini-2.5-flash-lite"
$VITE_GEMINI_WEB_SEARCH = "true"

$ACR_LOGIN_SERVER = "$ACR_NAME.azurecr.io"
$BACKEND_IMAGE = "$ACR_LOGIN_SERVER/$BACKEND_APP_NAME:$STAMP"
$FRONTEND_IMAGE = "$ACR_LOGIN_SERVER/$FRONTEND_APP_NAME:$STAMP"
```

### 4.2. Kiểm tra Azure và Docker

```powershell
az login
az extension add --name containerapp --upgrade --yes
az group create --name $RG --location $LOCATION
az acr show --name $ACR_NAME --resource-group $RG
az containerapp env show --name $ACA_ENV_NAME --resource-group $RG
docker version
```

Nếu `docker version` không ra thông tin server:
- Mở Docker Desktop.
- Chờ engine chạy xong rồi mới tiếp tục.

### 4.3. Login ACR

```powershell
$ACR_USERNAME = az acr credential show --name $ACR_NAME --query username -o tsv
$ACR_PASSWORD = az acr credential show --name $ACR_NAME --query passwords[0].value -o tsv

docker login $ACR_LOGIN_SERVER --username $ACR_USERNAME --password $ACR_PASSWORD
```

### 4.4. Build + push backend

```powershell
docker build -t $BACKEND_IMAGE $BACKEND_DIR
docker push $BACKEND_IMAGE
```

### 4.5. Update backend Container App

```powershell
az containerapp secret set `
  --name $BACKEND_APP_NAME `
  --resource-group $RG `
  --secrets `
    db-password=$DB_PASSWORD `
    smtp-app-password=$SMTP_APP_PASSWORD

az containerapp registry set `
  --name $BACKEND_APP_NAME `
  --resource-group $RG `
  --server $ACR_LOGIN_SERVER `
  --username $ACR_USERNAME `
  --password $ACR_PASSWORD

az containerapp update `
  --name $BACKEND_APP_NAME `
  --resource-group $RG `
  --image $BACKEND_IMAGE `
  --min-replicas 0 `
  --max-replicas 1 `
  --set-env-vars `
    DB_HOST=$DB_HOST `
    DB_PORT=$DB_PORT `
    DB_USER=$DB_USER `
    DB_PASSWORD=secretref:db-password `
    DB_NAME=$DB_NAME `
    DB_TLS=$DB_TLS `
    GIN_MODE=release `
    JWT_SECRET=$JWT_SECRET `
    FRONTEND_URL=http://localhost `
    SMTP_HOST=$SMTP_HOST `
    SMTP_PORT=$SMTP_PORT `
    SMTP_USERNAME=$SMTP_USERNAME `
    SMTP_APP_PASSWORD=secretref:smtp-app-password `
    SMTP_FROM=$SMTP_FROM

az containerapp ingress enable `
  --name $BACKEND_APP_NAME `
  --resource-group $RG `
  --type external `
  --target-port 8080
```

Lấy backend URL:

```powershell
$BACKEND_FQDN = az containerapp show --name $BACKEND_APP_NAME --resource-group $RG --query properties.configuration.ingress.fqdn -o tsv
$BACKEND_URL = "https://$BACKEND_FQDN"
$VITE_API_URL = "$BACKEND_URL/api"

$BACKEND_URL
$VITE_API_URL
```

### 4.6. Build + push frontend

```powershell
docker build `
  -t $FRONTEND_IMAGE `
  --build-arg VITE_API_URL=$VITE_API_URL `
  --build-arg VITE_GEMINI_API_KEY=$VITE_GEMINI_API_KEY `
  --build-arg VITE_GEMINI_MODEL=$VITE_GEMINI_MODEL `
  --build-arg VITE_GEMINI_WEB_SEARCH=$VITE_GEMINI_WEB_SEARCH `
  $FRONTEND_DIR

docker push $FRONTEND_IMAGE
```

Lưu ý:
- `VITE_*` là biến frontend, build xong sẽ nằm trong bundle client.
- Nếu dùng `VITE_GEMINI_API_KEY`, key này thực chất bị public cho browser.

### 4.7. Update frontend Container App

```powershell
az containerapp registry set `
  --name $FRONTEND_APP_NAME `
  --resource-group $RG `
  --server $ACR_LOGIN_SERVER `
  --username $ACR_USERNAME `
  --password $ACR_PASSWORD

az containerapp update `
  --name $FRONTEND_APP_NAME `
  --resource-group $RG `
  --image $FRONTEND_IMAGE `
  --min-replicas 0 `
  --max-replicas 1

az containerapp ingress enable `
  --name $FRONTEND_APP_NAME `
  --resource-group $RG `
  --type external `
  --target-port 8080
```

Lấy frontend URL:

```powershell
$FRONTEND_FQDN = az containerapp show --name $FRONTEND_APP_NAME --resource-group $RG --query properties.configuration.ingress.fqdn -o tsv
$FRONTEND_URL = "https://$FRONTEND_FQDN"

$FRONTEND_URL
```

### 4.8. Đồng bộ CORS backend với frontend URL

Sau khi frontend có domain thật, phải update ngược lại vào backend:

```powershell
az containerapp update `
  --name $BACKEND_APP_NAME `
  --resource-group $RG `
  --set-env-vars FRONTEND_URL=$FRONTEND_URL
```

## 5. Kiểm tra sau deploy

Health check backend:

```powershell
curl "$BACKEND_URL/health"
```

Kiểm tra frontend:

```powershell
curl -I $FRONTEND_URL
```

Kiểm tra một API có auth:

```powershell
curl "$BACKEND_URL/api/orders/unread-snapshot"
```

Kỳ vọng:
- Nếu chưa truyền token, route này thường trả `401`.
- `401` ở đây là bình thường, chứng tỏ backend đã serve đúng router.

Xem revision mới:

```powershell
az containerapp show --name $BACKEND_APP_NAME --resource-group $RG --query properties.latestRevisionName -o tsv
az containerapp show --name $FRONTEND_APP_NAME --resource-group $RG --query properties.latestRevisionName -o tsv
```

## 6. Lệnh ngắn để redeploy bản mới

Mỗi lần update code, bạn chỉ cần:
1. Tạo `STAMP` mới.
2. Build + push backend image.
3. `az containerapp update` backend.
4. Lấy lại `BACKEND_URL`.
5. Build + push frontend image với `VITE_API_URL` mới.
6. `az containerapp update` frontend.
7. Update lại `FRONTEND_URL` cho backend.

## 7. Nếu app chưa tồn tại

Tài liệu trên ưu tiên trường hợp app đã có sẵn.

Nếu `az containerapp show` báo không tồn tại, thay `update` bằng `create`.

Backend create:

```powershell
az containerapp create `
  --name $BACKEND_APP_NAME `
  --resource-group $RG `
  --environment $ACA_ENV_NAME `
  --image $BACKEND_IMAGE `
  --ingress external `
  --target-port 8080 `
  --min-replicas 0 `
  --max-replicas 1 `
  --registry-server $ACR_LOGIN_SERVER `
  --registry-username $ACR_USERNAME `
  --registry-password $ACR_PASSWORD `
  --secrets `
    db-password=$DB_PASSWORD `
    smtp-app-password=$SMTP_APP_PASSWORD `
  --env-vars `
    DB_HOST=$DB_HOST `
    DB_PORT=$DB_PORT `
    DB_USER=$DB_USER `
    DB_PASSWORD=secretref:db-password `
    DB_NAME=$DB_NAME `
    DB_TLS=$DB_TLS `
    GIN_MODE=release `
    JWT_SECRET=$JWT_SECRET `
    FRONTEND_URL=http://localhost `
    SMTP_HOST=$SMTP_HOST `
    SMTP_PORT=$SMTP_PORT `
    SMTP_USERNAME=$SMTP_USERNAME `
    SMTP_APP_PASSWORD=secretref:smtp-app-password `
    SMTP_FROM=$SMTP_FROM
```

Frontend create:

```powershell
az containerapp create `
  --name $FRONTEND_APP_NAME `
  --resource-group $RG `
  --environment $ACA_ENV_NAME `
  --image $FRONTEND_IMAGE `
  --ingress external `
  --target-port 8080 `
  --min-replicas 0 `
  --max-replicas 1 `
  --registry-server $ACR_LOGIN_SERVER `
  --registry-username $ACR_USERNAME `
  --registry-password $ACR_PASSWORD
```

## 8. Các lỗi thường gặp

### `TasksOperationsNotAllowed` khi dùng `az acr build`

Nguyên nhân:
- Azure subscription hoặc registry không cho dùng ACR Tasks.

Cách xử lý:
- Không dùng `az acr build`.
- Dùng `docker build` + `docker push` local như tài liệu này.

### WSL báo `docker: command not found`

Nguyên nhân:
- Docker Desktop chưa bật WSL integration hoặc không dùng Docker trong WSL.

Cách xử lý:
- Deploy bằng `PowerShell` Windows.
- Hoặc gọi trực tiếp `docker.exe` từ Windows.

### Script báo `/usr/bin/env: 'bash\r'`

Nguyên nhân:
- File shell đang dùng line ending `CRLF`.

Cách xử lý:
- Convert file sang `LF`.
- Hoặc bỏ script, deploy thủ công theo README này.

### Frontend gọi sai backend URL

Nguyên nhân:
- Build frontend với `VITE_API_URL` cũ.

Cách xử lý:
- Lấy lại `BACKEND_URL`.
- Build lại frontend image với `--build-arg VITE_API_URL=...`.
- Update lại frontend app.

### Backend lỗi CORS sau khi đổi frontend domain

Nguyên nhân:
- `FRONTEND_URL` của backend chưa cập nhật domain FE mới.

Cách xử lý:

```powershell
az containerapp update `
  --name $BACKEND_APP_NAME `
  --resource-group $RG `
  --set-env-vars FRONTEND_URL=$FRONTEND_URL
```

## 9. Refresh bảng `so_sanh_vat_tu` từ CSV mới

Lệnh dưới đây sẽ xóa và tạo lại bảng `hospital_db.so_sanh_vat_tu`, sau đó import dữ liệu từ CSV:

```bash
cd /path/to/bv108-consumables-management-backend
python3 scripts/import_so_sanh_vat_tu_csv.py "/path/to/export.csv" --env-file .env
```

Kết quả script sẽ in:
- Số dòng đã insert
- Số dòng trống bị bỏ qua
- Tổng bản ghi trong DB sau khi import

## 10. Ghi nhớ nhanh

- Deploy bằng local Docker sẽ lấy đúng code trên máy lúc build.
- Nếu muốn biết đang deploy nhánh nào, kiểm tra `git branch --show-current` trước.
- Nếu muốn deploy đúng một commit cụ thể, checkout đúng commit/branch trước khi build.
- Với frontend, mọi `VITE_*` đều là public phía client.
