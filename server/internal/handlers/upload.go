package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	db *sql.DB
}

func NewUploadHandler(db *sql.DB) *UploadHandler {
	return &UploadHandler{db: db}
}

var supportedExtensions = map[string]bool{
	".mp4": true, ".mkv": true, ".avi": true, ".mov": true, ".wmv": true,
	".flv": true, ".webm": true, ".m4v": true, ".mpg": true, ".mpeg": true, ".ts": true,
	".mp3": true, ".flac": true, ".aac": true, ".ogg": true, ".wav": true,
	".m4a": true, ".wma": true, ".opus": true,
}

const maxUploadSize = 10 * 1024 * 1024 * 1024 // 10GB

func (h *UploadHandler) Upload(c *gin.Context) {
	// Get target directory
	targetDir := c.PostForm("target_dir")
	if targetDir == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请指定目标目录"})
		return
	}

	// Validate target directory is within scan folders
	rows, err := h.db.Query("SELECT path FROM scan_folders")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询扫描目录失败"})
		return
	}

	var validDir bool
	for rows.Next() {
		var scanPath string
		rows.Scan(&scanPath)
		if strings.HasPrefix(targetDir, scanPath) || strings.HasPrefix(targetDir, strings.ReplaceAll(scanPath, "/", "\\")) {
			validDir = true
			break
		}
	}
	rows.Close()

	if !validDir {
		c.JSON(http.StatusBadRequest, gin.H{"error": "目标目录不在扫描范围内"})
		return
	}

	// Ensure directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建目标目录失败"})
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择文件"})
		return
	}
	defer file.Close()

	// Validate extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !supportedExtensions[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的格式: " + ext})
		return
	}

	// Validate size
	if header.Size > maxUploadSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("文件过大，最大支持 10GB（当前: %.1fGB）", float64(header.Size)/(1024*1024*1024))})
		return
	}

	// Save file
	destPath := filepath.Join(targetDir, header.Filename)
	out, err := os.Create(destPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建文件失败"})
		return
	}
	defer out.Close()

	// Copy with progress tracking
	written, err := io.Copy(out, file)
	if err != nil {
		os.Remove(destPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入文件失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "上传成功",
		"file_path":  destPath,
		"file_size":  written,
		"file_name":  header.Filename,
	})
}
