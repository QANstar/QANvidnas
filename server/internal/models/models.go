package models

import "time"

// User represents a registered user.
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	IsAdmin      bool      `json:"is_admin"`
	CreatedAt    time.Time `json:"created_at"`
}

// InviteCode represents a registration invite code.
type InviteCode struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	MaxUses     int    `json:"max_uses"`
	Used        int    `json:"used"`
	ExpiresAt   string `json:"expires_at"`
}

// Media represents a media file (video or audio).
type Media struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Type        string    `json:"type"` // "video" or "audio"
	Path        string    `json:"path"`
	Duration    float64   `json:"duration"`
	Resolution  string    `json:"resolution"`
	CoverPath   string    `json:"cover_path"`
	SpritePath  string    `json:"sprite_path"`
	SpriteMeta  string    `json:"sprite_meta"` // JSON: {frames, cols, fw, fh, interval}
	FileSize    int64     `json:"file_size"`
	Codec       string    `json:"codec"`
	Bitrate     int64     `json:"bitrate"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Deleted     bool      `json:"deleted"`
	Tags        []Tag     `json:"tags,omitempty"`
}

// Tag represents a label that can be attached to media.
type Tag struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// MediaTag is the join table between Media and Tag.
type MediaTag struct {
	MediaID int64 `json:"media_id"`
	TagID   int64 `json:"tag_id"`
}

// Playlist represents a user's playlist.
type Playlist struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	FolderPath string    `json:"folder_path"`
	UserID     int64     `json:"user_id"`
	PlayMode   string    `json:"play_mode"` // sequential, loop, random, single-loop
	CreatedAt  time.Time `json:"created_at"`
	Items      []PlaylistItem `json:"items,omitempty"`
}

// PlaylistItem is a media entry within a playlist.
type PlaylistItem struct {
	PlaylistID int64 `json:"playlist_id"`
	MediaID    int64 `json:"media_id"`
	Position   int   `json:"position"`
	Media      *Media `json:"media,omitempty"`
}

// ScanFolder represents a folder path to scan for media.
type ScanFolder struct {
	ID         int64     `json:"id"`
	Path       string    `json:"path"`
	LastScanAt time.Time `json:"last_scan_at"`
	Status     string    `json:"status"` // idle, scanning, error
}

// DeviceCode represents a TV login device code.
type DeviceCode struct {
	Code      string    `json:"code"`
	UserID    int64     `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
}

// MediaListResponse is the paginated media list response.
type MediaListResponse struct {
	Items      []Media `json:"items"`
	Total      int64   `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	TotalPages int     `json:"total_pages"`
}

// ScanProgress represents the current scan status.
type ScanProgress struct {
	Status    string `json:"status"` // idle, scanning, complete, error
	Total     int    `json:"total"`
	Processed int    `json:"processed"`
	Message   string `json:"message"`
}
