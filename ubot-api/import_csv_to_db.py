"""
Script import dữ liệu từ CSV vào bảng hoa_don trong MySQL database
"""

import csv
import mysql.connector
from datetime import datetime
import os
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

DB_CONFIG = {
    'host': os.getenv('DB_HOST', 'localhost'),
    'port': int(os.getenv('DB_PORT', '3306')),
    'user': os.getenv('DB_USER', 'root'),
    'password': os.getenv('DB_PASSWORD', ''),
    'database': os.getenv('DB_NAME', 'hospital_db'),
}


def parse_datetime(date_str):
    """Chuyển đổi chuỗi datetime từ CSV sang format MySQL DATE"""
    try:
        # Format: 2026-01-30T07:00:00Z
        dt = datetime.strptime(date_str, '%Y-%m-%dT%H:%M:%SZ')
        return dt.strftime('%Y-%m-%d')  # Chỉ lấy ngày, không lấy giờ
    except Exception as e:
        print(f"[IMPORT] Error parsing datetime: {date_str} - {e}")
        return None


def parse_float(value_str):
    """Chuyển đổi chuỗi số sang float"""
    try:
        return float(value_str) if value_str else 0.0
    except:
        return 0.0


def parse_int(value_str):
    """Chuyển đổi chuỗi số sang int"""
    try:
        return int(value_str) if value_str else 0
    except:
        return 0


def import_csv_to_database(csv_file='invoices_export.csv', clear_existing=True):
    """
    Import dữ liệu từ file CSV vào bảng hoa_don
    
    Args:
        csv_file: Đường dẫn đến file CSV
        clear_existing: Nếu True, xóa dữ liệu cũ trước khi import
    """
    try:
        # Kết nối database
        print(f"[IMPORT] Connecting to database {DB_CONFIG['database']}...")
        conn = mysql.connector.connect(**DB_CONFIG)
        cursor = conn.cursor()
        
        # Xóa dữ liệu cũ nếu cần
        if clear_existing:
            print("[IMPORT] Clearing old data...")
            cursor.execute("DELETE FROM hoa_don")
            conn.commit()
        
        # Đọc file CSV
        print(f"[IMPORT] Reading file {csv_file}...")
        with open(csv_file, 'r', encoding='utf-8-sig') as f:
            reader = csv.DictReader(f)
            
            # SQL insert - sử dụng tên cột tiếng Việt theo bảng thực tế
            insert_sql = """
                INSERT INTO hoa_don (
                    trang_thai_hoa_don, loai_hoa_don, so_hoa_don, ngay_hoa_don,
                    ma_so_thue_nguoi_ban, cong_ty, dia_chi, link_tra_cuu_hoa_don,
                    id_hoa_don, stt_dong_hang, ten_hang_hoa, ma_hang_hoa,
                    don_vi_tinh, so_luong, don_gia_chua_thue, thue_suat_gtgt
                ) VALUES (
                    %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s
                )
            """
            
            # Import từng dòng
            count = 0
            errors = 0
            
            for row in reader:
                try:
                    invoice_date = parse_datetime(row['Ngày hóa đơn'])
                    if not invoice_date:
                        continue
                    
                    values = (
                        row['Trạng thái hóa đơn'],
                        row['Loại hóa đơn'],
                        row['Số hóa đơn'],
                        invoice_date,
                        row['Mã số thuế người bán'],
                        row['Công ty'],
                        row['Địa chỉ'],
                        row['Link tra cứu hóa đơn'],
                        row['Id của hóa đơn'],
                        parse_int(row['STT dòng hàng']),
                        row['Tên hàng hóa'],
                        row['Mã hàng hóa'],
                        row['Đơn vị tính'],
                        parse_float(row['Số lượng']),
                        parse_float(row['Đơn giá chưa thuế']),
                        parse_float(row['Thuế suất GTGT'])
                    )
                    
                    cursor.execute(insert_sql, values)
                    count += 1
                    
                    if count % 100 == 0:
                        print(f"[IMPORT] Imported {count} rows...")
                        conn.commit()
                        
                except Exception as e:
                    errors += 1
                    print(f"[IMPORT] Error importing row {count + errors}: {e}")
                    continue
            
            # Commit cuối cùng
            conn.commit()
            
            print(f"\n[IMPORT] Completed!")
            print(f"   - Successful rows: {count}")
            print(f"   - Failed rows: {errors}")
            
            # Thống kê
            cursor.execute("SELECT COUNT(*) FROM hoa_don")
            total = cursor.fetchone()[0]
            print(f"   - Total records in database: {total}")
            
            cursor.execute("SELECT COUNT(DISTINCT id_hoa_don) FROM hoa_don")
            unique_invoices = cursor.fetchone()[0]
            print(f"   - Unique invoices: {unique_invoices}")
            
    except Exception as e:
        print(f"[IMPORT] Error: {e}")
        raise
    finally:
        if 'cursor' in locals():
            cursor.close()
        if 'conn' in locals():
            conn.close()
        print("[IMPORT] Database connection closed")


if __name__ == "__main__":
    import sys
    
    # Lấy tham số từ command line
    csv_file = sys.argv[1] if len(sys.argv) > 1 else 'invoices_export.csv'
    clear_existing = sys.argv[2].lower() != 'false' if len(sys.argv) > 2 else True
    
    print("="*60)
    print("IMPORT INVOICE DATA FROM CSV TO DATABASE")
    print("="*60)
    print(f"CSV File: {csv_file}")
    print(f"Clear existing data: {clear_existing}")
    print(f"Database: {DB_CONFIG['database']}")
    print("-"*60)
    
    import_csv_to_database(csv_file, clear_existing)
