package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

type RefreshHandler struct {
	hoaDonRepo interface{ GetCount() (int, error) }
}

func NewRefreshHandler(repo interface{ GetCount() (int, error) }) *RefreshHandler {
	return &RefreshHandler{hoaDonRepo: repo}
}

func (h *RefreshHandler) RefreshInvoices(c *gin.Context) {
	pythonPath := "C:/Users/User/anaconda3/envs/EnvironmentHP/python.exe"
	projectRoot := "D:/Projects/bv-108/bv108-consumables-management-backend"
	ubotDir := filepath.Join(projectRoot, "ubot-api")

	// Step 1: Export - Dùng auto_export.py không cần input
	fmt.Println("🚀 Crawling invoices...")
	exportCmd := exec.Command(pythonPath, "auto_export.py")
	exportCmd.Dir = ubotDir

	var outBuf, errBuf bytes.Buffer
	exportCmd.Stdout = &outBuf
	exportCmd.Stderr = &errBuf

	if err := exportCmd.Run(); err != nil {
		output := outBuf.String() + "\n" + errBuf.String()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Export script failed",
			"details": err.Error(),
			"output":  output,
		})
		return
	}

	// Step 2: Import
	fmt.Println("📥 Importing to database...")
	importCmd := exec.Command(pythonPath, "import_csv_to_db.py", "invoices_export.csv", "true")
	importCmd.Dir = ubotDir

	var importOut, importErr bytes.Buffer
	importCmd.Stdout = &importOut
	importCmd.Stderr = &importErr

	if err := importCmd.Run(); err != nil {
		output := importOut.String() + "\n" + importErr.String()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Import failed",
			"details": err.Error(),
			"output":  output,
		})
		return
	}

	// Get total count
	total, err := h.hoaDonRepo.GetCount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get count",
		})
		return
	}

	fmt.Println("✅ Refresh completed successfully")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Invoices refreshed successfully",
		"total":   total,
	})
}
