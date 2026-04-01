package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"bv108-consumables-management-backend/internal/realtime"

	"github.com/gin-gonic/gin"
)

type RefreshHandler struct {
	hoaDonRepo interface{ GetCount() (int, error) }
	hub        *realtime.Hub
}

func NewRefreshHandler(repo interface{ GetCount() (int, error) }, hub *realtime.Hub) *RefreshHandler {
	return &RefreshHandler{
		hoaDonRepo: repo,
		hub:        hub,
	}
}

func parseCommandSpec(spec string) (string, []string) {
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" {
		return "", nil
	}

	unquoted := strings.Trim(trimmed, "\"")
	if _, err := os.Stat(unquoted); err == nil {
		return unquoted, nil
	}

	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return "", nil
	}
	parts[0] = strings.Trim(parts[0], "\"")
	return parts[0], parts[1:]
}

func commandExists(command string) bool {
	if command == "" {
		return false
	}

	if strings.ContainsAny(command, `/\`) {
		_, err := os.Stat(command)
		return err == nil
	}

	_, err := exec.LookPath(command)
	return err == nil
}

func pythonRuntimeUsable(command string, args []string) bool {
	probeArgs := append(append([]string{}, args...), "--version")
	probeCmd := exec.Command(command, probeArgs...)

	var outBuf, errBuf bytes.Buffer
	probeCmd.Stdout = &outBuf
	probeCmd.Stderr = &errBuf

	if err := probeCmd.Run(); err != nil {
		return false
	}

	output := strings.ToLower(outBuf.String() + "\n" + errBuf.String())
	return strings.Contains(output, "python")
}

func resolvePythonRuntime(ubotDir string) (string, []string, []string) {
	type candidate struct {
		label string
		spec  string
	}

	candidates := make([]candidate, 0, 8)

	if envSpec := strings.TrimSpace(os.Getenv("PYTHON_PATH")); envSpec != "" {
		candidates = append(candidates, candidate{
			label: "PYTHON_PATH",
			spec:  envSpec,
		})
	}

	candidates = append(candidates,
		candidate{label: "venv-windows", spec: filepath.Join(ubotDir, ".venv", "Scripts", "python.exe")},
		candidate{label: "venv-linux", spec: filepath.Join(ubotDir, ".venv", "bin", "python")},
	)

	if runtime.GOOS == "windows" {
		candidates = append(candidates,
			candidate{label: "py -3", spec: "py -3"},
			candidate{label: "py", spec: "py"},
			candidate{label: "python3", spec: "python3"},
			candidate{label: "python", spec: "python"},
		)
	} else {
		candidates = append(candidates,
			candidate{label: "python3", spec: "python3"},
			candidate{label: "python", spec: "python"},
			candidate{label: "py -3", spec: "py -3"},
			candidate{label: "py", spec: "py"},
		)
	}

	tried := make([]string, 0, len(candidates))
	for _, item := range candidates {
		cmd, args := parseCommandSpec(item.spec)
		if cmd == "" {
			continue
		}

		tried = append(tried, fmt.Sprintf("%s => %s", item.label, item.spec))
		if commandExists(cmd) && pythonRuntimeUsable(cmd, args) {
			return cmd, args, tried
		}
	}

	return "", nil, tried
}

func (h *RefreshHandler) RefreshInvoices(c *gin.Context) {
	// Lấy thư mục gốc của project từ working directory
	projectRoot, err := os.Getwd()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get working directory",
			"details": err.Error(),
		})
		return
	}

	ubotDir := filepath.Join(projectRoot, "ubot-api")
	if _, err := os.Stat(ubotDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "UBot script directory not found",
			"details": fmt.Sprintf("expected path: %s", ubotDir),
		})
		return
	}

	pythonCmd, pythonBaseArgs, tried := resolvePythonRuntime(ubotDir)
	if pythonCmd == "" {
		hint := "Install Python 3 and set PYTHON_PATH (e.g. PYTHON_PATH=python3)."
		if runtime.GOOS == "windows" {
			hint = "Install Python 3 and set PYTHON_PATH=py -3 or PYTHON_PATH=<path-to-python.exe>."
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Python runtime not found",
			"details": hint,
			"tried":   tried,
		})
		return
	}

	// Step 1: Export - Dùng auto_export.py không cần input
	fmt.Println("🚀 Crawling invoices...")
	exportArgs := append(append([]string{}, pythonBaseArgs...), "auto_export.py")
	exportCmd := exec.Command(pythonCmd, exportArgs...)
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
	importArgs := append(append([]string{}, pythonBaseArgs...), "import_csv_to_db.py", "invoices_export.csv", "true")
	importCmd := exec.Command(pythonCmd, importArgs...)
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

	fmt.Println("Refresh completed successfully")
	if h.hub != nil {
		h.hub.Broadcast("invoices.data_refreshed", gin.H{
			"total":       total,
			"refreshedAt": time.Now().UTC().Format(time.RFC3339),
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Invoices refreshed successfully",
		"total":   total,
	})
}
