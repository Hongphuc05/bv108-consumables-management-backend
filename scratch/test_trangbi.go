//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"bv108-consumables-management-backend/config"
)

func main() {
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("load config: %v", err)
	}

	apiURL := "http://108.108.108.251/api/v1/resource/api_trangbi_thongtinvattu?method=select"
	reqBody := `{}`

	req, err := http.NewRequest("POST", apiURL, bytes.NewBufferString(reqBody))
	if err != nil {
		log.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if token := strings.TrimSpace(config.AppConfig.InternalSupplyAPIToken); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if cookie := strings.TrimSpace(config.AppConfig.InternalSupplyAPICookie); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("HTTP Status: %s\n", resp.Status)
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("read body: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		fmt.Printf("Raw response: %s\n", string(respBody))
		log.Fatalf("parse JSON: %v", err)
	}

	dataVal, exists := parsed["data"]
	if !exists {
		fmt.Printf("Raw response JSON: %s\n", string(respBody))
		return
	}

	dataList, ok := dataVal.([]interface{})
	if !ok {
		fmt.Printf("Data field is not a list. Type: %T\n", dataVal)
		return
	}

	fmt.Printf("Total items returned: %d\n", len(dataList))
	if len(dataList) > 0 {
		fmt.Println("\nFirst item:")
		itemJSON, _ := json.MarshalIndent(dataList[0], "", "  ")
		fmt.Println(string(itemJSON))
	}
	if len(dataList) > 1 {
		fmt.Println("\nSecond item:")
		itemJSON, _ := json.MarshalIndent(dataList[1], "", "  ")
		fmt.Println(string(itemJSON))
	}
}

		fmt.Printf("Decision: %s -> Total items: %d\n", dec, len(dataList))
		if len(dataList) > 0 {
			fmt.Println("  First item keys:")
			firstItem, _ := dataList[0].(map[string]interface{})
			for k := range firstItem {
				fmt.Printf("    * %s: %v\n", k, firstItem[k])
			}
		}
	}
}
