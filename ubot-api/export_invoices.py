"""
Script lấy hóa đơn theo đúng format yêu cầu
"""

from ubot_client import UBotAPIClient
import json
import re
from datetime import datetime, timedelta


def is_valid_item(item):
    """
    Kiểm tra xem có phải hàng hóa thật hay chỉ là dòng ghi chú
    Lọc bỏ các dòng:
    - Dòng ghi chú thuần túy không có giá trị (SL=0 VÀ giá=0)
    - Dòng CHỈ có chữ "Theo hợp đồng" mà không có mã hàng/tên hàng thật
    """
    item_name = item.get("itemName", "")
    item_qty = item.get("itemQuantity") or 0
    item_price = item.get("itemPrice") or 0
    
    # Lọc bỏ dòng có cả số lượng và đơn giá = 0 (dòng ghi chú không có giá trị)
    if item_qty == 0 and item_price == 0:
        return False
    
    # Nếu có giá trị (SL > 0 HOẶC giá > 0), chấp nhận dù có chữ "theo hợp đồng"
    # Vì nhiều hàng hóa thật có ghi chú hợp đồng ở cuối tên
    if item_qty > 0 or item_price > 0:
        return True
    
    return True


def extract_item_code(item_name):
    """
    Tách mã hàng hóa từ tên hàng hóa
    Mã hàng thường nằm trong ngoặc vuông [] hoặc ngoặc đơn () ở đầu tên
    Ví dụ: 
    - "[A33201] Bóng nong..." -> "A33201"
    - "(C02141) Dao cắt..." -> "C02141"
    """
    if not item_name:
        return ""
    
    # Tìm mã trong ngoặc vuông [..] ở đầu
    match = re.match(r'\[([^\]]+)\]', item_name)
    if match:
        code = match.group(1).strip()
        # Chỉ lấy nếu không phải là "Theo hợp đồng..."
        if "theo" not in code.lower():
            return code
    
    # Tìm mã trong ngoặc đơn (..) ở đầu
    match = re.match(r'\(([^)]+)\)', item_name)
    if match:
        code = match.group(1).strip()
        # Chỉ lấy nếu không phải là "Theo hợp đồng..."
        if "theo" not in code.lower():
            return code
    
    return ""


def export_invoices_to_format(
    username,
    password,
    is_production=False,
    page=0,
    size=100,
    invoice_types=["INPUT_ELECTRONIC_INVOICE"],
    invoice_status="VALID",
    released_date_from=None,
    released_date_to=None
):
    """
    Lấy hóa đơn và xuất theo format yêu cầu
    
    Returns:
        List của các dòng dữ liệu, mỗi dòng là 1 hàng hóa
    """
    # Khởi tạo client
    client = UBotAPIClient(username, password, is_production)
    
    # Đăng nhập
    print("[UBOT] Logging in...")
    client.login(remember_me=False)
    
    # Lấy hóa đơn
    print(f"[UBOT] Fetching invoices (page {page}, size {size})...")
    result = client.get_invoices(
        page=page,
        size=size,
        invoice_types=invoice_types,
        invoice_status=invoice_status,
        released_date_from=released_date_from,
        released_date_to=released_date_to,
        get_matching_data=False,
        get_attachments=True,  # Bật để lấy link PDF
        get_taxes=False
    )
    
    invoices = result.get("data", [])
    total = result.get("metadata", {}).get("total", 0)
    
    print(f"[UBOT] Fetched {len(invoices)}/{total} invoices")
    
    # Chuyển đổi sang format yêu cầu
    output_data = []
    
    for invoice in invoices:
        # Lấy invoiceId để tạo link UBot
        invoice_id = invoice.get("invoiceId", "")
        ubot_link = f"https://portal.ubot.vn/api/invoices/{invoice_id}/pdf/blob" if invoice_id else ""
        
        # Thông tin chung của hóa đơn
        invoice_info = {
            "Trạng thái hóa đơn": invoice.get("status"),
            "Loại hóa đơn": invoice.get("releaseStatus"),
            "Số hóa đơn": invoice.get("invoiceNo"),
            "Ngày hóa đơn": invoice.get("invoiceReleaseDate"),
            "Mã số thuế người bán": invoice.get("sellerTaxNo"),
            "Công ty": invoice.get("sellerName"),
            "Địa chỉ": invoice.get("sellerAddress"),
            "Link tra cứu hóa đơn": ubot_link,
            "Id của hóa đơn": invoice_id,
        }
        
        # Lấy danh sách hàng hóa
        items = invoice.get("invoiceItems", [])
        
        if items:
            # Lọc chỉ lấy hàng hóa thật, bỏ qua dòng ghi chú
            valid_items = [item for item in items if is_valid_item(item)]
            
            if valid_items:
                # Mỗi hàng hóa tạo 1 dòng riêng
                for item in valid_items:
                    item_name = item.get("itemName", "")
                    item_code = extract_item_code(item_name)
                    
                    row = invoice_info.copy()
                    row.update({
                        "STT dòng hàng": item.get("itemOrderNo"),
                        "Tên hàng hóa": item_name,
                        "Mã hàng hóa": item_code,
                        "Đơn vị tính": item.get("itemUnit") or "",
                        "Số lượng": item.get("itemQuantity") or 0,
                        "Đơn giá chưa thuế": item.get("itemPrice") or 0,
                        "Thuế suất GTGT": item.get("itemTax") or 0,
                    })
                    
                    output_data.append(row)
            else:
                # Nếu không có hàng hóa hợp lệ (chỉ toàn ghi chú), bỏ qua hóa đơn này
                pass
        else:
            # Nếu không có hàng hóa gì cả, bỏ qua hóa đơn này
            pass
    
    return output_data, total


def export_all_invoices(
    username,
    password,
    is_production=False,
    max_invoices=None,
    invoice_types=["INPUT_ELECTRONIC_INVOICE"],  # Khôi phục filter
    invoice_status="VALID"  # Khôi phục filter
):
    """
    Lấy tất cả hóa đơn với phân trang (CHỈ LẤY 3 NGÀY GẦN NHẤT)
    
    Args:
        max_invoices: Số lượng hóa đơn tối đa cần lấy (None = lấy hết)
    """
    # Tính toán ngày: từ 3 ngày trước đến hôm nay
    today = datetime.now()
    three_days_ago = today - timedelta(days=3)
    
    # Format theo định dạng DD/MM/YYYY
    date_to = today.strftime("%d/%m/%Y")
    date_from = three_days_ago.strftime("%d/%m/%Y")
    
    print(f"[UBOT] Fetching invoices from {date_from} to {date_to} (last 3 days)")
    
    all_data = []
    page = 0
    page_size = 100  # Max
    
    while True:
        data, total = export_invoices_to_format(
            username=username,
            password=password,
            is_production=is_production,
            page=page,
            size=page_size,
            invoice_types=invoice_types,
            invoice_status=invoice_status,
            released_date_from=date_from,
            released_date_to=date_to
        )
        
        all_data.extend(data)
        
        print(f"[UBOT] Retrieved {len(all_data)} rows")
        
        # Kiểm tra xem đã lấy đủ chưa
        if max_invoices and len(all_data) >= max_invoices:
            all_data = all_data[:max_invoices]
            break
        
        # Kiểm tra xem còn trang nữa không
        if (page + 1) * page_size >= total:
            break
        
        page += 1
    
    return all_data


def save_to_json(data, filename="invoices_export.json"):
    """Lưu dữ liệu ra file JSON"""
    with open(filename, "w", encoding="utf-8") as f:
        json.dump(data, f, ensure_ascii=False, indent=2)
    print(f"[UBOT] Saved {len(data)} rows to: {filename}")


def save_to_csv(data, filename="invoices_export.csv"):
    """Lưu dữ liệu ra file CSV"""
    import csv
    
    if not data:
        print("[UBOT] No data to export")
        return
    
    # Lấy headers từ dòng đầu tiên
    headers = list(data[0].keys())
    
    # Clean data: Thay xuống dòng bằng dấu chấm phẩy
    cleaned_data = []
    for row in data:
        cleaned_row = {}
        for key, value in row.items():
            if isinstance(value, str):
                # Thay thế xuống dòng và tab bằng dấu chấm phẩy + khoảng trắng
                cleaned_value = value.replace('\n', '; ').replace('\r', '').replace('\t', ' ')
                # Loại bỏ nhiều khoảng trắng liên tiếp
                cleaned_value = ' '.join(cleaned_value.split())
                cleaned_row[key] = cleaned_value
            else:
                cleaned_row[key] = value
        cleaned_data.append(cleaned_row)
    
    with open(filename, "w", encoding="utf-8-sig", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=headers)
        writer.writeheader()
        writer.writerows(cleaned_data)
    
    print(f"[UBOT] Saved {len(data)} rows to: {filename}")


# ============================================================================
# MAIN - Chạy script
# ============================================================================

if __name__ == "__main__":
    print("""
    ╔════════════════════════════════════════════════════════════════╗
    ║     Export Hóa Đơn UBot - 3 Ngày Gần Nhất (Database Ready)    ║
    ╚════════════════════════════════════════════════════════════════╝
    """)
    
    # Cấu hình
    USERNAME = "trangbibh@benhvien108.vn"
    PASSWORD = "Bv108@123"
    IS_PRODUCTION = False  # Fasle là môi trường dev, true là môi trường thật
    
    print("Chọn chế độ export (chỉ lấy 3 ngày gần nhất):")
    print("1. Lấy 100 dòng dữ liệu đầu tiên (test nhanh)")
    print("2. Lấy 1000 dòng dữ liệu")
    print("3. Lấy 5000 dòng dữ liệu")
    print("4. Lấy TẤT CẢ dữ liệu của 3 ngày gần nhất")
    
    choice = input("\nNhập số (1-4): ").strip()
    
    max_invoices = None
    if choice == "1":
        max_invoices = 100
    elif choice == "2":
        max_invoices = 1000
    elif choice == "3":
        max_invoices = 5000
    elif choice == "4":
        max_invoices = None  # Lấy hết
    else:
        print("Lựa chọn không hợp lệ!")
        exit(1)
    
    try:
        # Lấy dữ liệu
        print("\n" + "="*70)
        if max_invoices:
            print(f"Bắt đầu lấy {max_invoices} dòng dữ liệu (3 ngày gần nhất)...")
        else:
            print("Bắt đầu lấy TẤT CẢ dữ liệu của 3 ngày gần nhất...")
        print("="*70 + "\n")
        
        data = export_all_invoices(
            username=USERNAME,
            password=PASSWORD,
            is_production=IS_PRODUCTION,
            max_invoices=max_invoices,
            invoice_types=["INPUT_ELECTRONIC_INVOICE"],
            invoice_status="VALID"
        )
        
        print("\n" + "="*70)
        print(f"✓ HOÀN THÀNH! Đã lấy tổng cộng {len(data)} dòng dữ liệu")
        print("="*70 + "\n")
        
        # Hiển thị preview 3 dòng đầu
        print("📋 Preview 3 dòng đầu tiên:")
        print("-"*70)
        for i, row in enumerate(data[:3], 1):
            print(f"\n[{i}] {row.get('Số hóa đơn')} - {row.get('Công ty')}")
            print(f"    Hàng hóa: {row.get('Tên hàng hóa')}")
            print(f"    Mã HH: {row.get('Mã hàng hóa')}")
            print(f"    Số lượng: {row.get('Số lượng')} {row.get('Đơn vị tính')}")
            print(f"    Đơn giá: {row.get('Đơn giá chưa thuế'):,.0f}")
        
        # Lưu file
        print("\n" + "="*70)
        print("Đang lưu file...")
        print("="*70)
        
        save_to_json(data, "invoices_export.json")
        save_to_csv(data, "invoices_export.csv")
        
        print("\n" + "="*70)
        print("✓ XONG! Bạn có thể import các file sau vào database:")
        print("  - invoices_export.json")
        print("  - invoices_export.csv")
        print("="*70)
        
    except Exception as e:
        print(f"\n❌ Lỗi: {e}")
        import traceback
        traceback.print_exc()
