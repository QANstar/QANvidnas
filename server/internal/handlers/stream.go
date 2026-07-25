package handlers

import (
	"database/sql"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type StreamHandler struct {
	db *sql.DB
}

func NewStreamHandler(db *sql.DB) *StreamHandler {
	return &StreamHandler{db: db}
}

func (h *StreamHandler) StreamVideo(c *gin.Context) {
	h.serveStream(c, "video")
}

func (h *StreamHandler) StreamAudio(c *gin.Context) {
	h.serveStream(c, "audio")
}

func (h *StreamHandler) serveStream(c *gin.Context, expectedType string) {
	id := c.Param("id")

	var filePath, mediaType string
	err := h.db.QueryRow(
		"SELECT path, type FROM media WHERE id = ? AND deleted = 0", id,
	).Scan(&filePath, &mediaType)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "媒体不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	if expectedType != "" && mediaType != expectedType {
		c.JSON(http.StatusBadRequest, gin.H{"error": "媒体类型不匹配"})
		return
	}

	// Security: prevent path traversal
	if strings.Contains(filePath, "..") {
		c.JSON(http.StatusForbidden, gin.H{"error": "非法路径"})
		return
	}

	// Check file exists
	stat, err := os.Stat(filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	fileSize := stat.Size()

	// Determine MIME type
	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Set common headers
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Type", contentType)

	// Handle Range request
	rangeHeader := c.GetHeader("Range")
	if rangeHeader == "" {
		c.Header("Content-Length", strconv.FormatInt(fileSize, 10))
		c.File(filePath)
		return
	}

	// Parse Range header: "bytes=start-end"
	rangeStr := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.SplitN(rangeStr, "-", 2)

	var start, end int64
	start, _ = strconv.ParseInt(parts[0], 10, 64)

	if parts[1] != "" {
		end, _ = strconv.ParseInt(parts[1], 10, 64)
	} else {
		end = fileSize - 1
	}

	if end >= fileSize {
		end = fileSize - 1
	}

	if start > end || start >= fileSize {
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	// Open file and seek
	file, err := os.Open(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "打开文件失败"})
		return
	}
	defer file.Close()

	file.Seek(start, 0)
	chunkSize := end - start + 1

	contentRange := fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize)
	c.Header("Content-Range", contentRange)
	c.Header("Content-Length", strconv.FormatInt(chunkSize, 10))
	c.Status(http.StatusPartialContent)

	// Stream the chunk
	buf := make([]byte, 32*1024) // 32KB buffer
	remaining := chunkSize
	for remaining > 0 {
		readSize := int64(len(buf))
		if remaining < readSize {
			readSize = remaining
		}
		n, err := file.Read(buf[:readSize])
		if err != nil {
			break
		}
		c.Writer.Write(buf[:n])
		remaining -= int64(n)
	}
}

func (h *StreamHandler) ServeCover(c *gin.Context) {
	id := c.Param("id")
	coverType := c.DefaultQuery("type", "cover") // "cover" or "sprite"

	var coverPath, spritePath string
	var mediaType string
	err := h.db.QueryRow(
		"SELECT cover_path, sprite_path, type FROM media WHERE id = ? AND deleted = 0", id,
	).Scan(&coverPath, &spritePath, &mediaType)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "媒体不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	var filePath string
	if coverType == "sprite" && spritePath != "" {
		filePath = spritePath
	} else if coverPath != "" {
		filePath = coverPath
	}

	if filePath == "" || !strings.Contains(filePath, "/") {
		// Return SVG placeholder
		c.Header("Content-Type", "image/svg+xml")
		if mediaType == "audio" {
			c.String(http.StatusOK, audioPlaceholderSVG)
		} else {
			c.String(http.StatusOK, videoPlaceholderSVG)
		}
		return
	}

	// Security check
	if strings.Contains(filePath, "..") {
		c.JSON(http.StatusForbidden, gin.H{"error": "非法路径"})
		return
	}

	// Set content type
	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "image/jpeg"
	}
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=86400")

	c.File(filePath)
}

// SVG placeholders
const videoPlaceholderSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="480" height="270" viewBox="0 0 480 270">
  <rect fill="#1A1A2E" width="480" height="270"/>
  <polygon points="200,90 200,180 310,135" fill="#FF6B9D" opacity="0.8"/>
  <text x="240" y="230" text-anchor="middle" fill="#666" font-size="14" font-family="sans-serif">暂无封面</text>
</svg>`

const audioPlaceholderSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="480" height="270" viewBox="0 0 480 270">
  <rect fill="#1A1A2E" width="480" height="270"/>
  <circle cx="240" cy="120" r="40" fill="#6C5CE7" opacity="0.8"/>
  <text x="240" y="190" text-anchor="middle" fill="#FF6B9D" font-size="14" font-family="sans-serif">🎵</text>
  <text x="240" y="230" text-anchor="middle" fill="#666" font-size="14" font-family="sans-serif">音频文件</text>
</svg>`
