package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"qanvidnas/internal/middleware"

	"github.com/gin-gonic/gin"
)

type SearchHandler struct {
	db *sql.DB
}

func NewSearchHandler(db *sql.DB) *SearchHandler {
	return &SearchHandler{db: db}
}

func (h *SearchHandler) Search(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	tagFilter := c.Query("tags")     // comma-separated tag names
	typeFilter := c.Query("type")    // video, audio, or empty
	page, pageSize := middleware.GetPageParams(c)

	// Build SQL
	var conditions []string
	var args []interface{}

	if query != "" {
		conditions = append(conditions, `m.id IN (
			SELECT rowid FROM media_fts WHERE media_fts MATCH ?
		)`)
		// Escape FTS5 special chars and add prefix matching
		ftsQuery := escapeFTS5(query)
		args = append(args, ftsQuery)
	}

	if tagFilter != "" {
		tags := strings.Split(tagFilter, ",")
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				conditions = append(conditions, `m.id IN (
					SELECT mt.media_id FROM media_tags mt
					JOIN tags t ON t.id = mt.tag_id
					WHERE t.name = ?
				)`)
				args = append(args, tag)
			}
		}
	}

	if typeFilter == "video" || typeFilter == "audio" {
		conditions = append(conditions, "m.type = ?")
		args = append(args, typeFilter)
	}

	conditions = append(conditions, "m.deleted = 0")

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Count
	var total int64
	countQuery := "SELECT COUNT(*) FROM media m " + whereClause
	h.db.QueryRow(countQuery, args...).Scan(&total)

	// Query
	offset := (page - 1) * pageSize
	queryStr := `
		SELECT m.id, m.title, m.description, m.type, m.path, m.duration,
			   m.resolution, m.cover_path, m.file_size, m.codec, m.bitrate, m.created_at
		FROM media m ` + whereClause + `
		ORDER BY m.created_at DESC
		LIMIT ? OFFSET ?
	`
	fullArgs := append(args, pageSize, offset)

	rows, err := h.db.Query(queryStr, fullArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "搜索失败"})
		return
	}
	defer rows.Close()

	type SearchItem struct {
		ID          int64   `json:"id"`
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Type        string  `json:"type"`
		Path        string  `json:"path"`
		Duration    float64 `json:"duration"`
		Resolution  string  `json:"resolution"`
		CoverPath   string  `json:"cover_path"`
		FileSize    int64   `json:"file_size"`
		Codec       string  `json:"codec"`
		Bitrate     int64   `json:"bitrate"`
		CreatedAt   string  `json:"created_at"`
	}

	var items []SearchItem
	for rows.Next() {
		var item SearchItem
		rows.Scan(&item.ID, &item.Title, &item.Description, &item.Type, &item.Path,
			&item.Duration, &item.Resolution, &item.CoverPath, &item.FileSize,
			&item.Codec, &item.Bitrate, &item.CreatedAt)
		items = append(items, item)
	}

	if items == nil {
		items = []SearchItem{}
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  items,
		"total":  total,
		"page":   page,
		"query":  query,
	})
}

func (h *SearchHandler) GetTags(c *gin.Context) {
	rows, err := h.db.Query("SELECT id, name, color FROM tags ORDER BY name")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	var tags []gin.H
	for rows.Next() {
		var id int64
		var name, color string
		rows.Scan(&id, &name, &color)
		tags = append(tags, gin.H{"id": id, "name": name, "color": color})
	}

	if tags == nil {
		tags = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{"items": tags})
}

func (h *SearchHandler) CreateTag(c *gin.Context) {
	var req struct {
		Name  string `json:"name" binding:"required"`
		Color string `json:"color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "标签名不能为空"})
		return
	}

	if req.Color == "" {
		req.Color = "#6C5CE7" // Default blue
	}

	result, err := h.db.Exec("INSERT INTO tags (name, color) VALUES (?, ?)", req.Name, req.Color)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			c.JSON(http.StatusConflict, gin.H{"error": "标签名已存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	id, _ := result.LastInsertId()
	c.JSON(http.StatusOK, gin.H{"id": id, "name": req.Name, "color": req.Color})
}

func (h *SearchHandler) DeleteTag(c *gin.Context) {
	id := c.Param("id")

	_, err := h.db.Exec("DELETE FROM tags WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// escapeFTS5 escapes special characters in FTS5 query and adds prefix matching.
func escapeFTS5(query string) string {
	// Remove FTS5 special chars: " * - ( ) :
	replacer := strings.NewReplacer(
		"\"", "", "*", "", "-", "", "(", "", ")", "", ":", "",
	)
	cleaned := replacer.Replace(query)
	words := strings.Fields(cleaned)

	// Add prefix matching to each word for partial matching
	var escapedWords []string
	for _, w := range words {
		if len(w) > 0 {
			escapedWords = append(escapedWords, w+"*")
		}
	}

	return strings.Join(escapedWords, " ")
}
