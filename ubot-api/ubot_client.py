"""
UBot API Client - Kết nối và lấy thông tin hóa đơn
Tài liệu: https://help.ubot.vn/tai-lieu-ket-noi-api/
"""

import requests
import json
from typing import Dict, List, Optional
from datetime import datetime


class UBotAPIClient:
    """Client để kết nối với UBot API"""
    
    def __init__(self, username: str, password: str, is_production: bool = False):
        """
        Khởi tạo UBot API Client
        
        Args:
            username: Email đăng nhập (UBot cung cấp)
            password: Mật khẩu đăng nhập (UBot cung cấp)
            is_production: True nếu dùng môi trường production, False để dùng test/dev
        """
        self.username = username
        self.password = password
        
        # Chọn môi trường
        if is_production:
            self.base_url = "https://portal.ubot.vn/api"
        else:
            self.base_url = "https://portal-dev.ubot.vn/api"
        
        self.token = None
        self.headers = {
            "Content-Type": "application/json"
        }
    
    def login(self, remember_me: bool = False) -> Dict:
        """
        Đăng nhập và lấy token
        
        Args:
            remember_me: True = token hết hạn sau 30 ngày, False = token hết hạn sau 30 phút
            
        Returns:
            Dict chứa thông tin response, bao gồm token
        """
        url = f"{self.base_url}/authenticate"
        
        payload = {
            "username": self.username,
            "password": self.password,
            "rememberMe": remember_me
        }
        
        try:
            response = requests.post(url, json=payload, headers=self.headers)
            response.raise_for_status()
            
            data = response.json()
            # API trả về "id_token" chứ không phải "token"
            self.token = data.get("id_token") or data.get("token")
            
            # Cập nhật header với token
            self.headers["Authorization"] = f"Bearer {self.token}"
            
            print(f"[UBOT] Login successful!")
            return data
            
        except requests.exceptions.RequestException as e:
            print(f"[UBOT] Login error: {e}")
            raise
    
    def get_invoices(
        self,
        page: int = 0,
        size: int = 10,
        sort: str = "id,desc",  # Sort theo ID giảm dần (mới nhất trước)
        invoice_types: List[str] = None,
        invoice_id: str = None,
        transaction_ids: List[str] = None,
        buyer_tax_no: str = None,
        buyer_name: str = None,
        invoice_no: str = None,
        seller_tax_no: str = None,
        seller_name: str = None,
        released_date_from: str = None,
        released_date_to: str = None,
        received_date_from: str = None,
        received_date_to: str = None,
        tag_names: List[str] = None,
        is_combined_tag: bool = False,
        invoice_status: str = None,
        release_status: str = None,
        get_matching_data: bool = False,
        get_attachments: bool = False,
        get_taxes: bool = False
    ) -> Dict:
        """
        Lấy thông tin hóa đơn từ UBot
        
        Args:
            page: Số trang (mặc định 0)
            size: Số lượng hóa đơn mỗi trang (max 100)
            sort: Sắp xếp (ví dụ: 'id,desc', 'createdDate,asc'). Mặc định 'id,desc'
            invoice_types: Loại hóa đơn (INPUT_ELECTRONIC_INVOICE, OUTPUT_ELECTRONIC_INVOICE, etc.)
            invoice_id: ID hóa đơn cụ thể
            transaction_ids: Danh sách transaction ID
            buyer_tax_no: Mã số thuế bên mua
            buyer_name: Tên công ty bên mua
            invoice_no: Số hóa đơn
            seller_tax_no: Mã số thuế bên bán
            seller_name: Tên công ty bên bán
            released_date_from: Ngày phát hành từ (dd/mm/yyyy)
            released_date_to: Ngày phát hành đến (dd/mm/yyyy)
            received_date_from: Ngày nhận từ (dd/mm/yyyy)
            received_date_to: Ngày nhận đến (dd/mm/yyyy)
            tag_names: Danh sách tên nhãn
            is_combined_tag: True=AND, False=OR
            invoice_status: VALID, INVALID, IS_WAITING, IS_RECHECKING
            release_status: VALID, INVALID, VALID_INVOICE_MODIFIED, etc.
            get_matching_data: Lấy dữ liệu danh mục
            get_attachments: Lấy link download file đính kèm
            get_taxes: Lấy chi tiết thuế suất
            
        Returns:
            Dict chứa:
            - statusResponse: {statusCode, errorCode, message}
            - metadata: {page, size, total, orderBy, sort}
            - data: list của các hóa đơn (không phải 'invoices')
        """
        if not self.token:
            raise Exception("Chưa đăng nhập. Vui lòng gọi login() trước.")
        
        url = f"{self.base_url}/third-party/invoices"
        
        # Tạo request body
        body = {}
        
        if invoice_id:
            body["invoiceId"] = invoice_id
        if invoice_types:
            body["invoiceTypes"] = invoice_types
        else:
            body["invoiceTypes"] = []  # Lấy tất cả
        if transaction_ids:
            body["transactionIds"] = transaction_ids
        if buyer_tax_no:
            body["buyerTaxNo"] = buyer_tax_no
        if buyer_name:
            body["buyerName"] = buyer_name
        if invoice_no:
            body["invoiceNo"] = invoice_no
        if seller_tax_no:
            body["sellerTaxNo"] = seller_tax_no
        if seller_name:
            body["sellerName"] = seller_name
        if released_date_from:
            body["releasedDateFrom"] = released_date_from
        if released_date_to:
            body["releasedDateTo"] = released_date_to
        if received_date_from:
            body["receivedDateFrom"] = received_date_from
        if received_date_to:
            body["receivedDateTo"] = received_date_to
        if tag_names:
            body["tagNames"] = tag_names
        if is_combined_tag:
            body["isCombinedTag"] = is_combined_tag
        if invoice_status:
            body["invoiceStatus"] = invoice_status
        if release_status:
            body["releaseStatus"] = release_status
        
        body["getMatchingData"] = get_matching_data
        body["getAttachments"] = get_attachments
        body["getTaxes"] = get_taxes
        
        # Tạo query parameters
        params = {
            "page": page,
            "size": size,
            "sort": sort  # Sort là bắt buộc
        }
        
        try:
            response = requests.post(url, json=body, params=params, headers=self.headers)
            
            # Không raise error ngay, kiểm tra response trước
            data = response.json()
            
            # Nếu có lỗi từ server, in ra chi tiết
            if response.status_code >= 400:
                print(f"[UBOT] HTTP Error {response.status_code}")
                print(f"Response: {json.dumps(data, indent=2, ensure_ascii=False)}")
            
            return data
            
        except requests.exceptions.RequestException as e:
            print(f"[UBOT] Error fetching invoices: {e}")
            if hasattr(e, 'response') and hasattr(e.response, 'text'):
                print(f"Details: {e.response.text}")
            raise
    
    def send_invoice(
        self,
        transaction_id: str,
        company_id: str,
        files: List[str],
        sender: str,
        title: str = None,
        download_link: str = None,
        download_code: str = None,
        invoice_no: str = None,
        seller_tax_no: str = None,
        transaction_metadata: str = None,
        general_model_no: str = None,
        general_notation: str = None
    ) -> Dict:
        """
        Gửi thông tin hóa đơn lên UBot
        
        Args:
            transaction_id: Mã giao dịch
            company_id: ID công ty (UBot cung cấp)
            files: Danh sách file PDF/XML (max 15 files)
            sender: Người gửi
            title: Tiêu đề email
            download_link: Link tra cứu
            download_code: Mã tra cứu
            invoice_no: Số hóa đơn
            seller_tax_no: MST bên bán
            transaction_metadata: Thông tin gửi kèm
            general_model_no: Mẫu số
            general_notation: Ký hiệu
            
        Returns:
            Dict chứa status và message
        """
        if not self.token:
            raise Exception("Chưa đăng nhập. Vui lòng gọi login() trước.")
        
        url = f"{self.base_url}/third-party/upload-email"
        
        payload = {
            "transactionId": transaction_id,
            "companyId": company_id,
            "files": files,
            "sender": sender
        }
        
        if title:
            payload["title"] = title
        if download_link:
            payload["downloadLink"] = download_link
        if download_code:
            payload["downloadCode"] = download_code
        if invoice_no:
            payload["invoiceNo"] = invoice_no
        if seller_tax_no:
            payload["sellerTaxNo"] = seller_tax_no
        if transaction_metadata:
            payload["transactionMetadata"] = transaction_metadata
        if general_model_no:
            payload["generalModelNo"] = general_model_no
        if general_notation:
            payload["generalNotation"] = general_notation
        
        try:
            response = requests.post(url, json=payload, headers=self.headers)
            response.raise_for_status()
            
            return response.json()
            
        except requests.exceptions.RequestException as e:
            print(f"✗ Lỗi khi gửi hóa đơn: {e}")
            raise
    
    def get_transaction_status(self, transaction_ids: List[str]) -> Dict:
        """
        Lấy trạng thái của TransactionId
        
        Args:
            transaction_ids: Danh sách transaction ID cần kiểm tra
            
        Returns:
            Dict chứa thông tin trạng thái
        """
        if not self.token:
            raise Exception("Chưa đăng nhập. Vui lòng gọi login() trước.")
        
        url = f"{self.base_url}/third-party/transaction-status"
        
        payload = {
            "transactionIds": transaction_ids
        }
        
        try:
            response = requests.post(url, json=payload, headers=self.headers)
            response.raise_for_status()
            
            return response.json()
            
        except requests.exceptions.RequestException as e:
            print(f"✗ Lỗi khi lấy trạng thái: {e}")
            raise


def print_invoice_summary(invoice_data: Dict):
    """In tóm tắt thông tin hóa đơn"""
    print("\n" + "="*100)
    print(" "*40 + "THÔNG TIN HÓA ĐƠN")
    print("="*100)
    
    status_response = invoice_data.get("statusResponse", {})
    print(f"Status Code: {status_response.get('statusCode')} | Message: {status_response.get('message')}")
    
    metadata = invoice_data.get("metadata", {})
    print(f"Tổng số hóa đơn: {metadata.get('total')} | Trang: {metadata.get('page')} | Số lượng/trang: {metadata.get('size')}")
    
    invoices = invoice_data.get("data", [])  # API trả về 'data' chứ không phải 'invoices'
    print(f"\n📋 Danh sách {len(invoices)} hóa đơn:")
    print("="*100)
    
    for i, invoice in enumerate(invoices, 1):
        print(f"\n{'─'*100}")
        print(f"[{i}] HÓA ĐƠN SỐ: {invoice.get('invoiceNo')} | MẪU: {invoice.get('modelNo')} | KÝ HIỆU: {invoice.get('serial')}")
        print(f"{'─'*100}")
        print(f"🏢 Bên bán : {invoice.get('sellerName')}")
        print(f"   MST     : {invoice.get('sellerTaxNo')}")
        print(f"🏥 Bên mua : {invoice.get('buyerName')}")
        print(f"   MST     : {invoice.get('buyerTaxNo')}")
        print(f"{'─'*100}")
        
        subtotal = invoice.get('subTotal') or 0
        tax = invoice.get('taxAmount') or 0
        total = invoice.get('grandTotal') or 0
        currency = invoice.get('currency', 'VND')
        
        print(f"💰 Tổng tiền trước thuế : {subtotal:>20,.0f} {currency}")
        print(f"💰 Tiền thuế            : {tax:>20,.0f} {currency}")
        print(f"💰 TỔNG TIỀN SAU THUẾ   : {total:>20,.0f} {currency}")
        print(f"{'─'*100}")
        print(f"📅 Ngày phát hành : {invoice.get('invoiceReleaseDate')}")
        print(f"📅 Ngày nhận      : {invoice.get('receivedDate')}")
        print(f"✅ Trạng thái     : {invoice.get('status')}")
        print(f"📝 Loại hóa đơn   : {invoice.get('invoiceType')}")
        
        # In thông tin chi tiết hàng hóa
        items = invoice.get("invoiceItems", [])
        if items:
            print(f"\n📦 CHI TIẾT {len(items)} HÀNG HÓA:")
            print(f"{'─'*100}")
            for idx, item in enumerate(items, 1):
                qty = item.get('itemQuantity') or 0
                unit = item.get('itemUnit') or ''
                price = item.get('itemPrice') or 0
                item_total = item.get('itemGrandTotal') or 0
                
                print(f"  [{idx}] {item.get('itemName')}")
                print(f"      ├─ Số lượng  : {qty:>10.1f} {unit}")
                print(f"      ├─ Đơn giá   : {price:>20,.0f} {currency}")
                print(f"      └─ Thành tiền: {item_total:>20,.0f} {currency}")
                if idx < len(items):
                    print()


# ============================================================================
# VÍ DỤ SỬ DỤNG
# ============================================================================

if __name__ == "__main__":
    # Thông tin đăng nhập (UBot sẽ cung cấp)
    USERNAME = "your-email@example.com"  # Thay bằng email của bạn
    PASSWORD = "your-password"  # Thay bằng mật khẩu của bạn
    
    # Khởi tạo client (is_production=False để dùng môi trường test)
    client = UBotAPIClient(
        username=USERNAME,
        password=PASSWORD,
        is_production=False  # Đổi thành True khi dùng production
    )
    
    try:
        # 1. Đăng nhập
        print("🔐 Đang đăng nhập...")
        login_result = client.login(remember_me=False)
        
        # 2. Lấy danh sách hóa đơn
        print("\n📥 Đang lấy danh sách hóa đơn...")
        
        # Ví dụ 1: Lấy tất cả hóa đơn điện tử đầu vào
        invoices = client.get_invoices(
            page=0,
            size=10,
            invoice_types=["INPUT_ELECTRONIC_INVOICE"],
            get_matching_data=True,  # Lấy cả dữ liệu danh mục
            get_attachments=True,    # Lấy link download file
            get_taxes=True           # Lấy chi tiết thuế
        )
        
        # In thông tin
        print_invoice_summary(invoices)
        
        # 3. Lưu kết quả ra file JSON
        output_file = "invoices_result.json"
        with open(output_file, "w", encoding="utf-8") as f:
            json.dump(invoices, f, ensure_ascii=False, indent=2)
        print(f"\n✓ Đã lưu kết quả vào file: {output_file}")
        
        # Ví dụ 2: Tìm hóa đơn theo số hóa đơn
        print("\n\n🔍 Tìm hóa đơn theo số...")
        specific_invoice = client.get_invoices(
            invoice_no="0001234",  # Thay bằng số hóa đơn cần tìm
            size=1
        )
        
        # Ví dụ 3: Lấy hóa đơn theo khoảng thời gian
        print("\n\n📅 Lấy hóa đơn theo thời gian...")
        invoices_by_date = client.get_invoices(
            received_date_from="01/01/2024",
            received_date_to="31/12/2024",
            size=20,
            invoice_status="VALID"  # Chỉ lấy hóa đơn hợp lệ
        )
        
        print(f"\nTìm thấy {invoices_by_date.get('metadata', {}).get('total')} hóa đơn")
        
    except Exception as e:
        print(f"\n❌ Lỗi: {e}")
