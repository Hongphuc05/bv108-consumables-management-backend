#!/usr/bin/env python3
"""Refresh hospital_db.so_sanh_vat_tu from CSV (drop/create/import)."""

from __future__ import annotations

import argparse
import csv
import os
import re
from decimal import Decimal, InvalidOperation
from pathlib import Path
from typing import Dict, List, Optional, Tuple

import mysql.connector
from dotenv import load_dotenv


def normalize_header(value: str) -> str:
    return re.sub(r"\s+", " ", (value or "").strip().lower())


HEADER_TO_COLUMN = {
    normalize_header("ten_cong_ty"): "ten_cong_ty",
    normalize_header("ma_thu_vien"): "ma_thu_vien",
    normalize_header("ma_thong_tu_04"): "ma_thong_tu_04",
    normalize_header("ten_vat_tu"): "ten_vat_tu",
    normalize_header("ten_thuong_mai"): "ten_thuong_mai",
    normalize_header("TSKT_2025"): "tskt_2025",
    normalize_header("TSKT_2026"): "tskt_2026",
    normalize_header("Chất liệu/ Vật liệu"): "chat_lieu_vat_lieu",
    normalize_header("Đặc tính/Cấu tạo"): "dac_tinh_cau_tao",
    normalize_header("Kích thước"): "kich_thuoc",
    normalize_header("Chiều dài"): "chieu_dai",
    normalize_header("Tính năng sử dụng"): "tinh_nang_su_dung",
    normalize_header("TSKT khác"): "tskt_khac",
    normalize_header("ĐVT"): "dvt",
    normalize_header("Số lượng sử dụng 12 tháng (từ 01/6/2024 đến 31/5/2025)"): "so_luong_su_dung_12_thang",
    normalize_header("Số lượng trúng thầu 2025 + bổ sung"): "so_luong_trung_thau_2025_bo_sung",
    normalize_header("Đơn giá trúng thầu năm 2025"): "don_gia_trung_thau_2025",
    normalize_header("Đơn giá đề xuất năm 2026"): "don_gia_de_xuat_2026",
    normalize_header("Kết quả trúng thầu THẤP NHẤT trong vòng 12 tháng đăng tải trên HTMĐTQG (từ tháng 10/2024 đến nay)"): "ket_qua_trung_thau_thap_nhat",
    normalize_header("Thời gian/Đơn vị đăng tải kết quả trúng thầu có giá THẤP NHẤT"): "thoi_gian_don_vi_dang_tai_thap_nhat",
    normalize_header("Kết quả trúng thầu CAO NHẤT trong vòng 12 tháng đăng tải trên HTMĐTQG (từ tháng 10/2024 đến nay)"): "ket_qua_trung_thau_cao_nhat",
    normalize_header("Thời gian/Đơn vị đăng tải kết quả trúng thầu có giá CAO NHẤT"): "thoi_gian_don_vi_dang_tai_cao_nhat",
    normalize_header("Mã số thuế"): "ma_so_thue",
    normalize_header("MA_HIEU"): "ma_hieu",
    normalize_header("HANGSX"): "hangsx",
    normalize_header("NUOC_SX"): "nuoc_sx",
    normalize_header("Nhóm nước (G7,OECD, Châu âu….)"): "nhom_nuoc",
    normalize_header("Chất lượng (FDA,ISO….)"): "chat_luong",
    normalize_header("Mã 5086"): "ma_5086",
}

TARGET_COLUMNS = [
    "ten_cong_ty",
    "ma_thu_vien",
    "ma_thong_tu_04",
    "ten_vat_tu",
    "ten_thuong_mai",
    "tskt_2025",
    "tskt_2026",
    "chat_lieu_vat_lieu",
    "dac_tinh_cau_tao",
    "kich_thuoc",
    "chieu_dai",
    "tinh_nang_su_dung",
    "tskt_khac",
    "dvt",
    "so_luong_su_dung_12_thang",
    "so_luong_trung_thau_2025_bo_sung",
    "don_gia_trung_thau_2025",
    "don_gia_de_xuat_2026",
    "ket_qua_trung_thau_thap_nhat",
    "thoi_gian_don_vi_dang_tai_thap_nhat",
    "ket_qua_trung_thau_cao_nhat",
    "thoi_gian_don_vi_dang_tai_cao_nhat",
    "ma_so_thue",
    "ma_hieu",
    "hangsx",
    "nuoc_sx",
    "nhom_nuoc",
    "chat_luong",
    "ma_5086",
]

NUMERIC_COLUMNS = {
    "so_luong_su_dung_12_thang",
    "so_luong_trung_thau_2025_bo_sung",
    "don_gia_trung_thau_2025",
    "don_gia_de_xuat_2026",
    "ket_qua_trung_thau_thap_nhat",
    "ket_qua_trung_thau_cao_nhat",
}

CREATE_TABLE_SQL = """
CREATE TABLE so_sanh_vat_tu (
    stt INT NOT NULL AUTO_INCREMENT,
    ten_cong_ty VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    ma_thu_vien VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
    ma_thong_tu_04 VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    ten_vat_tu VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    ten_thuong_mai VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    tskt_2025 LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    tskt_2026 LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    chat_lieu_vat_lieu LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    dac_tinh_cau_tao LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    kich_thuoc LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    chieu_dai LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    tinh_nang_su_dung LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    tskt_khac LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    dvt VARCHAR(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    so_luong_su_dung_12_thang DECIMAL(18,2) DEFAULT NULL,
    so_luong_trung_thau_2025_bo_sung DECIMAL(18,2) DEFAULT NULL,
    don_gia_trung_thau_2025 DECIMAL(18,2) DEFAULT NULL,
    don_gia_de_xuat_2026 DECIMAL(18,2) DEFAULT NULL,
    ket_qua_trung_thau_thap_nhat DECIMAL(18,2) DEFAULT NULL,
    thoi_gian_don_vi_dang_tai_thap_nhat LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    ket_qua_trung_thau_cao_nhat DECIMAL(18,2) DEFAULT NULL,
    thoi_gian_don_vi_dang_tai_cao_nhat LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    ma_so_thue VARCHAR(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    ma_hieu LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    hangsx VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    nuoc_sx VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    nhom_nuoc VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    chat_luong VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    ma_5086 LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (stt),
    KEY idx_ma_thu_vien (ma_thu_vien),
    KEY idx_ten_vat_tu (ten_vat_tu)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
"""


def normalize_text(value: Optional[str]) -> Optional[str]:
    if value is None:
        return None
    cleaned = value.strip()
    return cleaned if cleaned else None


def parse_decimal(value: Optional[str]) -> Optional[Decimal]:
    text = normalize_text(value)
    if text is None:
        return None

    lowered = text.lower()
    if lowered in {"-", "--", "na", "n/a", "null", "none"}:
        return None

    compact = text.replace(" ", "")
    if "," in compact and "." in compact:
        compact = compact.replace(",", "")
    elif "," in compact:
        left, right = compact.rsplit(",", 1)
        if right.isdigit() and len(right) <= 2:
            compact = f"{left.replace(',', '')}.{right}"
        else:
            compact = compact.replace(",", "")

    compact = re.sub(r"[^0-9.\-]", "", compact)
    if not compact or compact in {"-", ".", "-."}:
        return None

    try:
        return Decimal(compact)
    except InvalidOperation:
        return None


def is_empty_numeric(value: Optional[str]) -> bool:
    text = normalize_text(value)
    if text is None:
        return True
    return text.lower() in {"-", "--", "na", "n/a", "null", "none"}


def build_column_indexes(header: List[str]) -> Dict[str, int]:
    indexes: Dict[str, int] = {}
    for idx, raw in enumerate(header):
        mapped = HEADER_TO_COLUMN.get(normalize_header(raw))
        if mapped and mapped not in indexes:
            indexes[mapped] = idx
    missing = [col for col in TARGET_COLUMNS if col not in indexes]
    if missing:
        raise ValueError(f"CSV thiếu cột bắt buộc: {', '.join(missing)}")
    return indexes


def load_rows(csv_path: Path) -> Tuple[List[Tuple[object, ...]], int, int, int]:
    rows: List[Tuple[object, ...]] = []
    skipped_empty = 0
    skipped_no_code = 0
    numeric_unparsed = 0

    with csv_path.open("r", encoding="utf-8-sig", newline="") as f:
        reader = csv.reader(f)
        header = next(reader, None)
        if not header:
            raise ValueError("CSV không có dòng tiêu đề")

        indexes = build_column_indexes(header)

        for raw_row in reader:
            extracted: Dict[str, Optional[str]] = {}
            for col, idx in indexes.items():
                raw_value = raw_row[idx] if idx < len(raw_row) else ""
                extracted[col] = normalize_text(raw_value)

            if not any(extracted.values()):
                skipped_empty += 1
                continue

            if not extracted.get("ma_thu_vien"):
                skipped_no_code += 1
                continue

            values: List[object] = []
            for col in TARGET_COLUMNS:
                raw_value = extracted.get(col)
                if col in NUMERIC_COLUMNS:
                    parsed = parse_decimal(raw_value)
                    if parsed is None and not is_empty_numeric(raw_value):
                        numeric_unparsed += 1
                    values.append(parsed)
                else:
                    values.append(raw_value)

            rows.append(tuple(values))

    return rows, skipped_empty, skipped_no_code, numeric_unparsed


def get_db_config(env_file: Optional[Path]) -> Dict[str, object]:
    if env_file:
        load_dotenv(env_file)
    else:
        load_dotenv()

    host = os.getenv("DB_HOST", "localhost")
    tls_setting = (os.getenv("DB_TLS", "") or "").strip().lower()

    use_tls = tls_setting in {"1", "true", "yes", "required"}
    if tls_setting == "" and ".mysql.database.azure.com" in host:
        use_tls = True

    return {
        "host": host,
        "port": int(os.getenv("DB_PORT", "3306")),
        "user": os.getenv("DB_USER", "root"),
        "password": os.getenv("DB_PASSWORD", ""),
        "database": os.getenv("DB_NAME", "hospital_db"),
        "charset": "utf8mb4",
        "use_unicode": True,
        "ssl_disabled": not use_tls,
    }


def refresh_table(conn: mysql.connector.MySQLConnection, rows: List[Tuple[object, ...]]) -> None:
    cursor = conn.cursor()
    try:
        cursor.execute("DROP TABLE IF EXISTS so_sanh_vat_tu")
        cursor.execute(CREATE_TABLE_SQL)

        insert_sql = f"""
            INSERT INTO so_sanh_vat_tu ({', '.join(TARGET_COLUMNS)})
            VALUES ({', '.join(['%s'] * len(TARGET_COLUMNS))})
        """
        if rows:
            cursor.executemany(insert_sql, rows)

        conn.commit()
    finally:
        cursor.close()


def main() -> None:
    parser = argparse.ArgumentParser(description="Import CSV so_sanh_vat_tu into hospital_db")
    parser.add_argument("csv_file", help="Đường dẫn file CSV")
    parser.add_argument("--env-file", default=".env", help="Đường dẫn file .env chứa DB config")
    args = parser.parse_args()

    csv_path = Path(args.csv_file).expanduser().resolve()
    if not csv_path.exists():
        raise FileNotFoundError(f"Không tìm thấy file CSV: {csv_path}")

    env_file: Optional[Path] = None
    if args.env_file:
        possible_env = Path(args.env_file).expanduser().resolve()
        if possible_env.exists():
            env_file = possible_env

    rows, skipped_empty, skipped_no_code, numeric_unparsed = load_rows(csv_path)

    db_config = get_db_config(env_file)
    conn = mysql.connector.connect(**db_config)
    try:
        refresh_table(conn, rows)

        cursor = conn.cursor()
        cursor.execute("SELECT COUNT(*) FROM so_sanh_vat_tu")
        total = cursor.fetchone()[0]
        cursor.close()

        print("[OK] Đã refresh bảng hospital_db.so_sanh_vat_tu")
        print(f"     - File CSV: {csv_path}")
        print(f"     - Dòng insert: {len(rows)}")
        print(f"     - Dòng trống bỏ qua: {skipped_empty}")
        print(f"     - Dòng thiếu ma_thu_vien bỏ qua: {skipped_no_code}")
        print(f"     - Ô số không parse được -> NULL: {numeric_unparsed}")
        print(f"     - Tổng bản ghi trong DB: {total}")
    finally:
        conn.close()


if __name__ == "__main__":
    main()
