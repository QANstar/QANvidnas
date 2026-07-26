package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"qanvidnas/internal/config"
	"qanvidnas/internal/middleware"
	"qanvidnas/internal/models"
	"qanvidnas/internal/scanner"

	"github.com/gin-gonic/gin"
)

type MediaHandler struct {
	db      *sql.DB
	scanner *scanner.Scanner
	cfg     *config.Config
	dataDir string
}

func NewMediaHandler(db *sql.DB, scanner *scanner.Scanner, cfg *config.Config) *MediaHandler {
	return &MediaHandler{
		db:      db,
		scanner: scanner,
		cfg:     cfg,
		dataDir: cfg.DataDir,
	}
}

func (h *MediaHandler) List(c *gin.Context) {
	page, pageSize := middleware.GetPageParams(c)
	typeFilter := c.Query("type")
	sortBy := c.DefaultQuery("sort", "created_at")
	order := c.DefaultQuery("order", "desc")

	// Validate sort
	validSorts := map[string]string{
		"title": "title", "created_at": "created_at", "duration": "duration", "file_size": "file_size",
	}
	sortCol, ok := validSorts[sortBy]
	if !ok {
		sortCol = "created_at"
	}
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	// Build query
	where := "WHERE m.deleted = 0"
	args := []interface{}{}
	if typeFilter == "video" || typeFilter == "audio" {
		where += " AND m.type = ?"
		args = append(args, typeFilter)
	}

	// Count total
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM media m %s", where)
	h.db.QueryRow(countQuery, args...).Scan(&total)

	offset := (page - 1) * pageSize
	query := fmt.Sprintf(`
		SELECT m.id, m.title, m.description, m.type, m.path, m.duration,
			   m.resolution, m.cover_path, m.sprite_path, m.sprite_meta,
			   m.file_size, m.codec, m.bitrate, m.created_at, m.updated_at
		FROM media m %s
		ORDER BY m.%s %s
		LIMIT ? OFFSET ?
	`, where, sortCol, order)

	args = append(args, pageSize, offset)
	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	var items []models.Media
	for rows.Next() {
		var m models.Media
		rows.Scan(&m.ID, &m.Title, &m.Description, &m.Type, &m.Path, &m.Duration,
			&m.Resolution, &m.CoverPath, &m.SpritePath, &m.SpriteMeta,
			&m.FileSize, &m.Codec, &m.Bitrate, &m.CreatedAt, &m.UpdatedAt)
		m.Tags = h.loadTags(m.ID)
		items = append(items, m)
	}

	if items == nil {
		items = []models.Media{}
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	c.JSON(http.StatusOK, models.MediaListResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

func (h *MediaHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var m models.Media
	err := h.db.QueryRow(`
		SELECT id, title, description, type, path, duration,
			   resolution, cover_path, sprite_path, sprite_meta,
			   file_size, codec, bitrate, created_at, updated_at
		FROM media WHERE id = ? AND deleted = 0
	`, id).Scan(&m.ID, &m.Title, &m.Description, &m.Type, &m.Path, &m.Duration,
		&m.Resolution, &m.CoverPath, &m.SpritePath, &m.SpriteMeta,
		&m.FileSize, &m.Codec, &m.Bitrate, &m.CreatedAt, &m.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "媒体不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	m.Tags = h.loadTags(m.ID)
	c.JSON(http.StatusOK, m)
}

func (h *MediaHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Tags        []int64 `json:"tag_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败"})
		return
	}
	defer tx.Rollback()

	// Update title and description
	if req.Title != "" {
		_, err := tx.Exec("UPDATE media SET title = ?, description = ?, updated_at = ? WHERE id = ?",
			req.Title, req.Description, time.Now(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
			return
		}
	}

	// Update tags if provided
	if req.Tags != nil {
		tx.Exec("DELETE FROM media_tags WHERE media_id = ?", id)
		for _, tagID := range req.Tags {
			tx.Exec("INSERT OR IGNORE INTO media_tags (media_id, tag_id) VALUES (?, ?)", id, tagID)
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	// Update FTS index
	mediaID, _ := strconv.ParseInt(id, 10, 64)
	h.updateFTS(mediaID)

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

func (h *MediaHandler) UploadCover(c *gin.Context) {
	id := c.Param("id")

	// Check media exists
	var mediaType string
	err := h.db.QueryRow("SELECT type FROM media WHERE id = ? AND deleted = 0", id).Scan(&mediaType)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "媒体不存在"})
		return
	}

	file, err := c.FormFile("cover")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择封面图片"})
		return
	}

	// Validate image type
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 JPG 和 PNG 格式"})
		return
	}

	// Create cover directory
	coverDir := filepath.Join(h.dataDir, "covers")
	os.MkdirAll(coverDir, 0755)

	coverPath := filepath.Join(coverDir, fmt.Sprintf("%s_custom%s", id, ext))
	if err := c.SaveUploadedFile(file, coverPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存封面失败"})
		return
	}

	// Update database
	relativePath := filepath.Join("covers", fmt.Sprintf("%s_custom%s", id, ext))
	_, err = h.db.Exec("UPDATE media SET cover_path = ?, updated_at = ? WHERE id = ?",
		relativePath, time.Now(), id)
	if err != nil {
		os.Remove(coverPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新数据库失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "封面上传成功", "cover_path": relativePath})
}

func (h *MediaHandler) DeleteCover(c *gin.Context) {
	id := c.Param("id")

	var coverPath string
	h.db.QueryRow("SELECT cover_path FROM media WHERE id = ?", id).Scan(&coverPath)

	// Remove custom cover file
	if coverPath != "" {
		os.Remove(filepath.Join(h.dataDir, coverPath))
	}

	// Clear cover path to revert to auto-generated
	_, err := h.db.Exec("UPDATE media SET cover_path = '' WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已恢复自动封面"})
}

func (h *MediaHandler) GetSprite(c *gin.Context) {
	id := c.Param("id")

	var spritePath, spriteMeta string
	err := h.db.QueryRow(
		"SELECT sprite_path, sprite_meta FROM media WHERE id = ? AND deleted = 0", id,
	).Scan(&spritePath, &spriteMeta)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "媒体不存在"})
		return
	}

	if spriteMeta == "" {
		c.JSON(http.StatusOK, gin.H{"sprite_url": "", "meta": nil})
		return
	}

	var meta map[string]interface{}
	json.Unmarshal([]byte(spriteMeta), &meta)

	c.JSON(http.StatusOK, gin.H{
		"sprite_url": "/api/stream/cover/" + id + "?type=sprite",
		"meta":       meta,
	})
}

// Scan triggers a manual full scan.
func (h *MediaHandler) Scan(c *gin.Context) {
	go h.scanner.FullScan()

	c.JSON(http.StatusOK, gin.H{
		"message": "扫描已启动",
		"status":  "scanning",
	})
}

func (h *MediaHandler) GetScanProgress(c *gin.Context) {
	progress := h.scanner.GetProgress()
	c.JSON(http.StatusOK, progress)
}

// ScanFolder management

func (h *MediaHandler) GetScanFolders(c *gin.Context) {
	rows, err := h.db.Query("SELECT id, path, last_scan_at, status FROM scan_folders ORDER BY id")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	var folders []models.ScanFolder
	for rows.Next() {
		var f models.ScanFolder
		var lastScan sql.NullTime
		rows.Scan(&f.ID, &f.Path, &lastScan, &f.Status)
		if lastScan.Valid {
			f.LastScanAt = lastScan.Time
		}
		folders = append(folders, f)
	}

	if folders == nil {
		folders = []models.ScanFolder{}
	}

	c.JSON(http.StatusOK, gin.H{"items": folders})
}

func (h *MediaHandler) AddScanFolder(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入文件夹路径"})
		return
	}

	// Verify path exists
	info, err := os.Stat(req.Path)
	if err != nil || !info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "路径不存在或无法访问"})
		return
	}

	_, err = h.db.Exec("INSERT OR IGNORE INTO scan_folders (path) VALUES (?)", req.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加失败"})
		return
	}

	// Trigger scan
	go h.scanner.ScanPath(req.Path)

	c.JSON(http.StatusOK, gin.H{"message": "添加成功，扫描已启动"})
}

func (h *MediaHandler) DeleteScanFolder(c *gin.Context) {
	id := c.Param("id")

	_, err := h.db.Exec("DELETE FROM scan_folders WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	// Mark media from this folder as deleted
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// Helper methods

func (h *MediaHandler) loadTags(mediaID int64) []models.Tag {
	rows, err := h.db.Query(`
		SELECT t.id, t.name, t.color FROM tags t
		JOIN media_tags mt ON t.id = mt.tag_id
		WHERE mt.media_id = ?
	`, mediaID)
	if err != nil {
		return []models.Tag{}
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var t models.Tag
		rows.Scan(&t.ID, &t.Name, &t.Color)
		tags = append(tags, t)
	}
	if tags == nil {
		tags = []models.Tag{}
	}
	return tags
}

func (h *MediaHandler) updateFTS(mediaID int64) {
	var title, desc string
	var tagNames []string

	h.db.QueryRow("SELECT title, description FROM media WHERE id = ?", mediaID).Scan(&title, &desc)

	rows, _ := h.db.Query(`
		SELECT t.name FROM tags t
		JOIN media_tags mt ON t.id = mt.tag_id
		WHERE mt.media_id = ?
	`, mediaID)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var name string
			rows.Scan(&name)
			tagNames = append(tagNames, name)
		}
	}

	tags := strings.Join(tagNames, " ")

	h.db.Exec("DELETE FROM media_fts WHERE rowid = ?", mediaID)
	h.db.Exec("INSERT INTO media_fts (rowid, title, description, tags) VALUES (?, ?, ?, ?)",
		mediaID, title, desc, tags)
}

func (h *MediaHandler) GetAllScanPaths() []string {
	rows, err := h.db.Query("SELECT path FROM scan_folders")
	if err != nil {
		return nil
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var p string
		rows.Scan(&p)
		paths = append(paths, p)
	}
	return paths
}
