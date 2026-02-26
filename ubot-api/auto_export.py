"""
Auto-export script - Tự động lấy TẤT CẢ hóa đơn 3 ngày gần nhất
Dùng để gọi từ backend API
"""
from export_invoices import export_all_invoices, save_to_csv
import sys

USERNAME = "trangbibh@benhvien108.vn"
PASSWORD = "Bv108@123"
IS_PRODUCTION = False

try:
    print("[AUTO_EXPORT] Starting invoice crawl for last 3 days...")
    
    # Get ALL invoices (option 4)
    data = export_all_invoices(
        username=USERNAME,
        password=PASSWORD,
        is_production=IS_PRODUCTION,
        max_invoices=None,  # None = get all
        invoice_types=["INPUT_ELECTRONIC_INVOICE"],
        invoice_status="VALID"
    )
    
    print(f"[AUTO_EXPORT] Retrieved {len(data)} invoice rows")
    
    # Save to CSV
    save_to_csv(data, "invoices_export.csv")
    print("[AUTO_EXPORT] Saved to invoices_export.csv")
    
    sys.exit(0)
    
except Exception as e:
    print(f"[AUTO_EXPORT] ERROR: {e}")
    import traceback
    traceback.print_exc()
    sys.exit(1)
