package main

import (
	"encoding/json"
	"fmt"
	"log"

	"bv108-consumables-management-backend/config"
	"bv108-consumables-management-backend/internal/database"
)

func main() {
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("load config: %v", err)
	}

	if err := database.InitDB(); err != nil {
		log.Fatalf("init db: %v", err)
	}
	defer database.CloseDB()

	// Query for D00071
	row := database.DB.QueryRow(`
		SELECT 
			IDX1, PRODUCTID, GROUPNAME, ID, IDX2, MA_HIEU, TYPENAME, NAME, UNIT,
			QUY_CACH_DONG_GOI, QUY_CACH_GIAO_HANG, THONG_TIN_THAU, TONGTHAU,
			HANGSX, NUOC_SX, NHA_CUNG_CAP, PRICE, TONDAUKY, NHAPTRONGKY,
			XUATTRONGKY, TONGNHAP, TON_KHO_MIN
		FROM supplies 
		WHERE ID = 'D00071'
	`)

	var (
		idx1                                                                                                   int
		productID, tonDauKy, nhapTrongKy, xuatTrongKy, tongNhap, tonKhoMin                                     interface{}
		groupName, id, idx2, maHieu, typeName, name, unit, quyCachDongGoi, quyCachGiaoHang, thongTinThau, tongThau interface{}
		hangSX, nuocSX, nhaCungCap                                                                             interface{}
		price                                                                                                  float64
	)

	err := row.Scan(
		&idx1, &productID, &groupName, &id, &idx2, &maHieu, &typeName, &name, &unit,
		&quyCachDongGoi, &quyCachGiaoHang, &thongTinThau, &tongThau,
		&hangSX, &nuocSX, &nhaCungCap, &price, &tonDauKy, &nhapTrongKy,
		&xuatTrongKy, &tongNhap, &tonKhoMin,
	)

	if err != nil {
		log.Fatalf("Error scanning supplies: %v", err)
	}

	res := map[string]interface{}{
		"IDX1":               idx1,
		"PRODUCTID":          productID,
		"GROUPNAME":          groupName,
		"ID":                 id,
		"IDX2":               idx2,
		"MA_HIEU":            maHieu,
		"TYPENAME":           typeName,
		"NAME":               name,
		"UNIT":               unit,
		"QUY_CACH_DONG_GOI":  quyCachDongGoi,
		"QUY_CACH_GIAO_HANG": quyCachGiaoHang,
		"THONG_TIN_THAU":    thongTinThau,
		"TONGTHAU":           tongThau,
		"HANGSX":             hangSX,
		"NUOC_SX":            nuocSX,
		"NHA_CUNG_CAP":       nhaCungCap,
		"PRICE":              price,
		"TONDAUKY":           tonDauKy,
		"NHAPTRONGKY":        nhapTrongKy,
		"XUATTRONGKY":        xuatTrongKy,
		"TONGNHAP":           tongNhap,
		"TON_KHO_MIN":        tonKhoMin,
	}

	out, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println("Synced record in database (supplies table) for ID='D00071':")
	fmt.Println(string(out))
}
