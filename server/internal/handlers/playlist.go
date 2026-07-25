package handlers

import (
	"database/sql"
	"math/rand"
	"net/http"
	"sort"
	"strings"

	"qanvidnas/internal/middleware"
	"qanvidnas/internal/models"

	"github.com/gin-gonic/gin"
)

type PlaylistHandler struct {
	db *sql.DB
}

func NewPlaylistHandler(db *sql.DB) *PlaylistHandler {
	return &PlaylistHandler{db: db}
}

func (h *PlaylistHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	rows, err := h.db.Query(
		"SELECT id, name, folder_path, play_mode, created_at FROM playlists WHERE user_id = ? ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	var playlists []models.Playlist
	for rows.Next() {
		var p models.Playlist
		rows.Scan(&p.ID, &p.Name, &p.FolderPath, &p.PlayMode, &p.CreatedAt)
		p.UserID = userID

		// Count items
		var count int
		h.db.QueryRow("SELECT COUNT(*) FROM playlist_items WHERE playlist_id = ?", p.ID).Scan(&count)

		playlists = append(playlists, p)
		p.Items = nil // Don't load items for list view
		_ = count
	}

	if playlists == nil {
		playlists = []models.Playlist{}
	}

	c.JSON(http.StatusOK, gin.H{"items": playlists})
}

func (h *PlaylistHandler) Get(c *gin.Context) {
	id := c.Param("id")
	userID := middleware.GetUserID(c)

	var p models.Playlist
	err := h.db.QueryRow(
		"SELECT id, name, folder_path, user_id, play_mode, created_at FROM playlists WHERE id = ? AND user_id = ?",
		id, userID,
	).Scan(&p.ID, &p.Name, &p.FolderPath, &p.UserID, &p.PlayMode, &p.CreatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "播放列表不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	// Load items with media
	rows, err := h.db.Query(`
		SELECT pi.media_id, pi.position,
			   m.id, m.title, m.description, m.type, m.duration,
			   m.resolution, m.cover_path, m.file_size
		FROM playlist_items pi
		JOIN media m ON m.id = pi.media_id AND m.deleted = 0
		WHERE pi.playlist_id = ?
		ORDER BY pi.position
	`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	var items []models.PlaylistItem
	for rows.Next() {
		var item models.PlaylistItem
		var media models.Media
		rows.Scan(&item.MediaID, &item.Position,
			&media.ID, &media.Title, &media.Description, &media.Type, &media.Duration,
			&media.Resolution, &media.CoverPath, &media.FileSize)
		item.PlaylistID = p.ID
		item.Media = &media
		items = append(items, item)
	}

	if items == nil {
		items = []models.PlaylistItem{}
	}

	// Apply shuffle if mode is random
	if p.PlayMode == "random" {
		items = shuffleItems(items)
	}

	p.Items = items
	c.JSON(http.StatusOK, p)
}

func (h *PlaylistHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		Name       string `json:"name" binding:"required"`
		FolderPath string `json:"folder_path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写播放列表名称和文件夹路径"})
		return
	}

	// Create playlist
	result, err := h.db.Exec(
		"INSERT INTO playlists (name, folder_path, user_id, play_mode) VALUES (?, ?, ?, 'sequential')",
		req.Name, req.FolderPath, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建播放列表失败"})
		return
	}

	playlistID, _ := result.LastInsertId()

	// Recursively scan folder and add media
	go h.populatePlaylist(playlistID, req.FolderPath)

	c.JSON(http.StatusOK, gin.H{
		"id":          playlistID,
		"name":        req.Name,
		"folder_path": req.FolderPath,
		"message":     "播放列表创建中，媒体文件正在添加",
	})
}

func (h *PlaylistHandler) Update(c *gin.Context) {
	id := c.Param("id")
	userID := middleware.GetUserID(c)

	var req struct {
		PlayMode string `json:"play_mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	validModes := map[string]bool{"sequential": true, "loop": true, "random": true, "single-loop": true}
	if !validModes[req.PlayMode] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的播放模式"})
		return
	}

	_, err := h.db.Exec(
		"UPDATE playlists SET play_mode = ? WHERE id = ? AND user_id = ?",
		req.PlayMode, id, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

func (h *PlaylistHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	userID := middleware.GetUserID(c)

	_, err := h.db.Exec("DELETE FROM playlists WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

func (h *PlaylistHandler) PlayNow(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		FolderPath string `json:"folder_path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请指定文件夹路径"})
		return
	}

	// Create a temporary playlist
	folderName := req.FolderPath
	if idx := strings.LastIndex(folderName, "/"); idx >= 0 {
		folderName = folderName[idx+1:]
	}
	if idx := strings.LastIndex(folderName, "\\"); idx >= 0 {
		folderName = folderName[idx+1:]
	}

	result, err := h.db.Exec(
		"INSERT INTO playlists (name, folder_path, user_id, play_mode) VALUES (?, ?, ?, 'sequential')",
		folderName, req.FolderPath, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建播放列表失败"})
		return
	}

	playlistID, _ := result.LastInsertId()

	// Populate synchronously for immediate playback
	h.populatePlaylist(playlistID, req.FolderPath)

	// Get first item
	var firstMediaID int64
	err = h.db.QueryRow(
		"SELECT media_id FROM playlist_items WHERE playlist_id = ? ORDER BY position LIMIT 1",
		playlistID,
	).Scan(&firstMediaID)

	if err == sql.ErrNoRows {
		h.db.Exec("DELETE FROM playlists WHERE id = ?", playlistID)
		c.JSON(http.StatusNotFound, gin.H{"error": "文件夹中没有可播放的媒体"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"playlist_id": playlistID,
		"first_media": gin.H{"id": firstMediaID},
	})
}

// populatePlaylist recursively scans a folder and adds its media files to the playlist.
func (h *PlaylistHandler) populatePlaylist(playlistID int64, folderPath string) {
	var mediaList []int64

	// Get all media whose path starts with this folder
	rows, err := h.db.Query(`
		SELECT id FROM media WHERE deleted = 0 AND (path LIKE ? OR path LIKE ?)
		ORDER BY path
	`, folderPath+"%", folderPath+"\\%")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		rows.Scan(&id)
		mediaList = append(mediaList, id)
	}

	// Insert into playlist
	for i, mediaID := range mediaList {
		h.db.Exec(
			"INSERT OR IGNORE INTO playlist_items (playlist_id, media_id, position) VALUES (?, ?, ?)",
			playlistID, mediaID, i,
		)
	}
}

// shuffleItems performs Fisher-Yates shuffle on a slice of playlist items.
func shuffleItems(items []models.PlaylistItem) []models.PlaylistItem {
	shuffled := make([]models.PlaylistItem, len(items))
	copy(shuffled, items)

	for i := len(shuffled) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	// Update positions
	for i := range shuffled {
		shuffled[i].Position = i
	}

	// Sort by position
	sort.Slice(shuffled, func(i, j int) bool {
		return shuffled[i].Position < shuffled[j].Position
	})

	return shuffled
}
