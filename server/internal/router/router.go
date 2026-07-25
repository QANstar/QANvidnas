package router

import (
	"database/sql"
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"qanvidnas/internal/auth"
	"qanvidnas/internal/config"
	"qanvidnas/internal/handlers"
	"qanvidnas/internal/middleware"
	"qanvidnas/internal/scanner"
	"qanvidnas/internal/thumbnail"

	"github.com/gin-gonic/gin"
)

//go:embed web/dist
var embeddedWeb embed.FS

func Setup(db *sql.DB, cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(middleware.CORSMiddleware())

	// Initialize services
	authSvc := auth.NewService(db, cfg.Server.Host+cfg.Server.Port) // Simplified: use timestamp for JWT secret
	thumb := thumbnail.NewThumbnailer(db, cfg.DataDir)
	scan := scanner.NewScanner(db, thumb)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, authSvc)
	mediaHandler := handlers.NewMediaHandler(db, scan, cfg)
	searchHandler := handlers.NewSearchHandler(db)
	playlistHandler := handlers.NewPlaylistHandler(db)
	streamHandler := handlers.NewStreamHandler(db)
	uploadHandler := handlers.NewUploadHandler(db)

	// Start file watcher
	go scan.StartWatch()

	// Public routes (no auth required)
	api := r.Group("/api")
	{
		api.GET("/auth/check-setup", authHandler.CheckSetup)
		api.POST("/auth/setup", authHandler.Setup)
		api.POST("/auth/login", authHandler.Login)
		api.POST("/auth/register", authHandler.Register)
		api.GET("/auth/device-code", authHandler.GenerateDeviceCode)
		api.GET("/auth/device-code/poll", authHandler.PollDeviceCode)
	}

	// Protected routes (auth required)
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(authSvc))
	{
		// Auth
		protected.GET("/auth/me", authHandler.Me)
		protected.POST("/auth/device-code/authorize", authHandler.AuthorizeDeviceCode)

		// Media
		protected.GET("/media", mediaHandler.List)
		protected.GET("/media/:id", mediaHandler.Get)
		protected.PUT("/media/:id", mediaHandler.Update)
		protected.POST("/media/:id/cover", mediaHandler.UploadCover)
		protected.DELETE("/media/:id/cover", mediaHandler.DeleteCover)
		protected.GET("/media/:id/sprite", mediaHandler.GetSprite)
		protected.POST("/media/scan", mediaHandler.Scan)
		protected.GET("/media/scan/progress", mediaHandler.GetScanProgress)

		// Stream
		protected.GET("/stream/video/:id", streamHandler.StreamVideo)
		protected.GET("/stream/audio/:id", streamHandler.StreamAudio)
		protected.GET("/stream/cover/:id", streamHandler.ServeCover)

		// Search
		protected.GET("/search", searchHandler.Search)
		protected.GET("/tags", searchHandler.GetTags)
		protected.POST("/tags", searchHandler.CreateTag)
		protected.DELETE("/tags/:id", searchHandler.DeleteTag)

		// Playlist
		protected.GET("/playlists", playlistHandler.List)
		protected.GET("/playlists/:id", playlistHandler.Get)
		protected.POST("/playlists", playlistHandler.Create)
		protected.PUT("/playlists/:id", playlistHandler.Update)
		protected.DELETE("/playlists/:id", playlistHandler.Delete)
		protected.POST("/playlists/play-now", playlistHandler.PlayNow)

		// Upload
		protected.POST("/upload", uploadHandler.Upload)

		// Password
		protected.POST("/auth/change-password", authHandler.ChangePassword)

		// Admin routes
		admin := protected.Group("")
		admin.Use(middleware.AdminMiddleware())
		{
			// Invite codes
			admin.GET("/admin/invite-codes", authHandler.GetInviteCodes)
			admin.POST("/admin/invite-codes", authHandler.CreateInviteCode)
			admin.DELETE("/admin/invite-codes/:code", authHandler.DeleteInviteCode)

			// Scan folders
			admin.GET("/admin/scan-folders", mediaHandler.GetScanFolders)
			admin.POST("/admin/scan-folders", mediaHandler.AddScanFolder)
			admin.DELETE("/admin/scan-folders/:id", mediaHandler.DeleteScanFolder)

			// Config
			admin.GET("/admin/config", func(c *gin.Context) {
				c.JSON(http.StatusOK, cfg)
			})
		}
	}

	// Serve embedded React SPA
	webFS, err := fs.Sub(embeddedWeb, "web/dist")
	if err != nil {
		// If embedded web not available, return 404 for non-API routes
		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.String(http.StatusOK, "QANvidnas API Server - Frontend not embedded")
		})
		return r
	}

	fileServer := http.FileServer(http.FS(webFS))
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		// Try to serve the file, fallback to index.html for SPA routing
		path := c.Request.URL.Path
		f, err := webFS.Open(strings.TrimPrefix(path, "/"))
		if err != nil {
			// SPA fallback
			c.Request.URL.Path = "/"
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		f.Close()
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	return r
}
