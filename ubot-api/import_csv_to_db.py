"""
Script import dá»¯ liá»‡u tá»« CSV vÃ o báº£ng hoa_don trong MySQL database
"""

import csv
import difflib
import mysql.connector
from datetime import datetime
import os
import unicodedata
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
    """Chuyá»ƒn Ä‘á»•i chuá»—i datetime tá»« CSV sang format MySQL DATE"""
    try:
        # Format: 2026-01-30T07:00:00Z
        dt = datetime.strptime(date_str, '%Y-%m-%dT%H:%M:%SZ')
        return dt.strftime('%Y-%m-%d')  # Chá»‰ láº¥y ngÃ y, khÃ´ng láº¥y giá»
    except Exception as e:
        print(f"[IMPORT] Error parsing datetime: {date_str} - {e}")
        return None


def parse_float(value_str):
    """Chuyá»ƒn Ä‘á»•i chuá»—i sá»‘ sang float"""
    try:
        return float(value_str) if value_str else 0.0
    except:
        return 0.0


def parse_int(value_str):
    """Chuyá»ƒn Ä‘á»•i chuá»—i sá»‘ sang int"""
    try:
        return int(value_str) if value_str else 0
    except:
        return 0


def _normalize_header_name(value):
    """Normalize CSV header names to a comparable ascii token."""
    if not value:
        return ""
    normalized = unicodedata.normalize('NFKD', str(value))
    no_diacritics = "".join(ch for ch in normalized if not unicodedata.combining(ch))
    return "".join(ch.lower() for ch in no_diacritics if ch.isalnum())


def _build_header_index(fieldnames):
    index = {}
    if not fieldnames:
        return index
    for raw_name in fieldnames:
        key = _normalize_header_name(raw_name)
        if key and key not in index:
            index[key] = raw_name
    return index


def _get_csv_value(row, header_index, expected_keys):
    """Fetch CSV value by normalized key, with fuzzy fallback for malformed headers."""
    for expected in expected_keys:
        normalized_expected = _normalize_header_name(expected)
        if normalized_expected in header_index:
            return row.get(header_index[normalized_expected], "")

    available_keys = list(header_index.keys())
    for expected in expected_keys:
        normalized_expected = _normalize_header_name(expected)
        if not normalized_expected or not available_keys:
            continue
        matches = difflib.get_close_matches(normalized_expected, available_keys, n=1, cutoff=0.75)
        if matches:
            return row.get(header_index[matches[0]], "")

    raise KeyError(expected_keys[0])


def _load_text_column_lengths(cursor, schema_name):
    """Load max lengths for text columns in hoa_don table."""
    sql = """
        SELECT COLUMN_NAME, CHARACTER_MAXIMUM_LENGTH
        FROM INFORMATION_SCHEMA.COLUMNS
        WHERE TABLE_SCHEMA = %s
          AND TABLE_NAME = 'hoa_don'
          AND DATA_TYPE IN ('char', 'varchar', 'text', 'tinytext', 'mediumtext', 'longtext')
    """
    cursor.execute(sql, (schema_name,))
    lengths = {}
    for column_name, max_len in cursor.fetchall():
        lengths[column_name] = max_len
    return lengths


def _fit_text(value, column_name, text_lengths):
    """Trim text to column size to avoid insertion failures."""
    if value is None:
        return ""
    text = str(value)
    max_len = text_lengths.get(column_name)
    if max_len and len(text) > max_len:
        return text[:max_len]
    return text


def import_csv_to_database(csv_file='invoices_export.csv', clear_existing=True):
    """
    Import dá»¯ liá»‡u tá»« file CSV vÃ o báº£ng hoa_don
    
    Args:
        csv_file: ÄÆ°á»ng dáº«n Ä‘áº¿n file CSV
        clear_existing: Náº¿u True, xÃ³a dá»¯ liá»‡u cÅ© trÆ°á»›c khi import
    """
    try:
        # Káº¿t ná»‘i database
        print(f"[IMPORT] Connecting to database {DB_CONFIG['database']}...")
        conn = mysql.connector.connect(**DB_CONFIG)
        cursor = conn.cursor()
        text_lengths = _load_text_column_lengths(cursor, DB_CONFIG['database'])

        # Äá»c file CSV vÃ  parse trÆ°á»›c. KhÃ´ng xÃ³a dá»¯ liá»‡u cá»§ nÃªu CSV rá»—ng/lá»—i.
        print(f"[IMPORT] Reading file {csv_file}...")
        with open(csv_file, 'r', encoding='utf-8-sig') as f:
            reader = csv.DictReader(f)
            header_index = _build_header_index(reader.fieldnames)

            parsed_rows = []
            errors = 0
            for row in reader:
                try:
                    invoice_date = parse_datetime(_get_csv_value(row, header_index, ['Ngay hoa don', 'Ngày hóa đơn']))
                    if not invoice_date:
                        continue

                    values = (
                        _fit_text(_get_csv_value(row, header_index, ['Trang thai hoa don', 'Trạng thái hóa đơn']), 'trang_thai_hoa_don', text_lengths),
                        _fit_text(_get_csv_value(row, header_index, ['Loai hoa don', 'Loại hóa đơn']), 'loai_hoa_don', text_lengths),
                        _fit_text(_get_csv_value(row, header_index, ['So hoa don', 'Số hóa đơn']), 'so_hoa_don', text_lengths),
                        invoice_date,
                        _fit_text(_get_csv_value(row, header_index, ['Ma so thue nguoi ban', 'Mã số thuế người bán']), 'ma_so_thue_nguoi_ban', text_lengths),
                        _fit_text(_get_csv_value(row, header_index, ['Cong ty', 'Công ty']), 'cong_ty', text_lengths),
                        _fit_text(_get_csv_value(row, header_index, ['Dia chi', 'Địa chỉ']), 'dia_chi', text_lengths),
                        _fit_text(_get_csv_value(row, header_index, ['Link tra cuu hoa don', 'Link tra cứu hóa đơn']), 'link_tra_cuu_hoa_don', text_lengths),
                        _fit_text(_get_csv_value(row, header_index, ['Id cua hoa don', 'Id của hóa đơn']), 'id_hoa_don', text_lengths),
                        parse_int(_get_csv_value(row, header_index, ['STT dong hang', 'STT dòng hàng'])),
                        _fit_text(_get_csv_value(row, header_index, ['Ten hang hoa', 'Tên hàng hóa']), 'ten_hang_hoa', text_lengths),
                        _fit_text(_get_csv_value(row, header_index, ['Ma hang hoa', 'Mã hàng hóa']), 'ma_hang_hoa', text_lengths),
                        _fit_text(_get_csv_value(row, header_index, ['Don vi tinh', 'Đơn vị tính']), 'don_vi_tinh', text_lengths),
                        parse_float(_get_csv_value(row, header_index, ['So luong', 'Số lượng'])),
                        parse_float(_get_csv_value(row, header_index, ['Don gia chua thue', 'Đơn giá chưa thuế'])),
                        parse_float(_get_csv_value(row, header_index, ['Thue suat GTGT', 'Thuế suất GTGT']))
                    )

                    parsed_rows.append(values)
                except Exception as e:
                    errors += 1
                    print(f"[IMPORT] Error parsing row {len(parsed_rows) + errors}: {e}")
                    continue

            if clear_existing and len(parsed_rows) == 0:
                raise ValueError("CSV has no valid rows; aborting refresh to avoid deleting existing data")

            # SQL insert - sá»­ dá»¥ng tÃªn cá»™t tiáº¿ng Viá»‡t theo báº£ng thá»±c táº¿
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

            # Transaction an toÃ n: cÃ³ lá»—i sáº½ rollback, khÃ´ng máº¥t dá»¯ liá»‡u cÅ©

            if clear_existing:
                print("[IMPORT] Clearing old data...")
                cursor.execute("DELETE FROM hoa_don")

            if parsed_rows:
                cursor.executemany(insert_sql, parsed_rows)

            print(f"\n[IMPORT] Completed!")
            print(f"   - Successful rows: {len(parsed_rows)}")
            print(f"   - Failed rows: {errors}")

            # Thá»‘ng kÃª
            cursor.execute("SELECT COUNT(*) FROM hoa_don")
            total = cursor.fetchone()[0]
            print(f"   - Total records in database: {total}")

            cursor.execute("SELECT COUNT(DISTINCT id_hoa_don) FROM hoa_don")
            unique_invoices = cursor.fetchone()[0]
            print(f"   - Unique invoices: {unique_invoices}")

            # Auto-fill company_contact_id after each refresh import
            update_contact_sql = """
                UPDATE hoa_don hd
                JOIN company_contacts cc
                  ON hd.cong_ty = cc.company_name
                SET hd.company_contact_id = cc.id
            """
            cursor.execute(update_contact_sql)
            print(f"   - company_contact_id updated rows: {cursor.rowcount}")

            conn.commit()
            
    except Exception as e:
        if 'conn' in locals() and conn.is_connected():
            try:
                conn.rollback()
            except Exception:
                pass
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
    
    # Láº¥y tham sá»‘ tá»« command line
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

