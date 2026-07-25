package thumbnail

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

type Thumbnailer struct {
	db        *sql.DB
	dataDir   string
	queue     chan thumbnailJob
	workerWg  sync.WaitGroup
}

type thumbnailJob struct {
	Type     string // "cover" or "sprite"
	MediaID  int64
	FilePath string
	Duration float64
}

func NewThumbnailer(db *sql.DB, dataDir string) *Thumbnailer {
	t := &Thumbnailer{
		db:      db,
		dataDir: dataDir,
		queue:   make(chan thumbnailJob, 100),
	}

	// Start worker pool (2 workers to avoid CPU overload)
	for i := 0; i < 2; i++ {
		t.workerWg.Add(1)
		go t.worker()
	}

	return t
}

func (t *Thumbnailer) GenerateCover(mediaID int64, filePath string, duration float64) {
	t.queue <- thumbnailJob{Type: "cover", MediaID: mediaID, FilePath: filePath, Duration: duration}
}

func (t *Thumbnailer) GenerateSprite(mediaID int64, filePath string, duration float64) {
	t.queue <- thumbnailJob{Type: "sprite", MediaID: mediaID, FilePath: filePath, Duration: duration}
}

func (t *Thumbnailer) worker() {
	defer t.workerWg.Done()
	for job := range t.queue {
		switch job.Type {
		case "cover":
			t.processCover(job)
		case "sprite":
			t.processSprite(job)
		}
	}
}

func (t *Thumbnailer) processCover(job thumbnailJob) {
	coverDir := filepath.Join(t.dataDir, "covers")
	os.MkdirAll(coverDir, 0755)

	outputPath := filepath.Join(coverDir, fmt.Sprintf("%d_cover.jpg", job.MediaID))

	// Determine timestamp position
	position := job.Duration * 0.1 // 10% position
	if job.Duration < 10 {
		position = job.Duration * 0.5 // Middle for short videos
	}

	// Extract frame
	if err := t.extractFrame(job.FilePath, outputPath, position, 480); err != nil {
		log.Printf("Cover generation failed for media %d: %v", job.MediaID, err)
		// Try fallback position
		if job.Duration >= 10 {
			position = job.Duration * 0.3
			if err := t.extractFrame(job.FilePath, outputPath, position, 480); err != nil {
				return
			}
		} else {
			return
		}
	}

	// Check for black frame
	if t.isBlackFrame(outputPath) {
		position = job.Duration * 0.3
		if job.Duration < 10 {
			position = job.Duration * 0.5
		}
		if err := t.extractFrame(job.FilePath, outputPath, position, 480); err != nil {
			return
		}

		// If still black, try 50%
		if t.isBlackFrame(outputPath) {
			position = job.Duration * 0.5
			t.extractFrame(job.FilePath, outputPath, position, 480)
		}
	}

	// Update DB
	relativePath := filepath.Join("covers", fmt.Sprintf("%d_cover.jpg", job.MediaID))
	t.db.Exec("UPDATE media SET cover_path = ? WHERE id = ?", relativePath, job.MediaID)
}

func (t *Thumbnailer) processSprite(job thumbnailJob) {
	spriteDir := filepath.Join(t.dataDir, "sprites")
	os.MkdirAll(spriteDir, 0755)

	tempDir := filepath.Join(spriteDir, fmt.Sprintf("temp_%d", job.MediaID))
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	interval := 10.0 // one frame every 10 seconds
	totalFrames := int(math.Ceil(job.Duration / interval))
	if totalFrames > 720 {
		totalFrames = 720 // Cap at 2 hours worth
	}

	cols := 10
	rows := int(math.Ceil(float64(totalFrames) / float64(cols)))

	// Extract each frame
	successfulFrames := 0
	for i := 0; i < totalFrames; i++ {
		timestamp := float64(i) * interval
		framePath := filepath.Join(tempDir, fmt.Sprintf("frame_%04d.jpg", i))

		if err := t.extractFrame(job.FilePath, framePath, timestamp, 160); err != nil {
			continue
		}
		successfulFrames++
	}

	if successfulFrames == 0 {
		return
	}

	// Stitch frames into sprite grid using ffmpeg
	outputPath := filepath.Join(spriteDir, fmt.Sprintf("%d_sprite.jpg", job.MediaID))

	// Use ffmpeg tile filter to create sprite
	cmd := exec.Command("ffmpeg",
		"-y",
		"-framerate", fmt.Sprintf("1/%f", interval),
		"-start_number", "0",
		"-i", filepath.Join(tempDir, "frame_%04d.jpg"),
		"-vf", fmt.Sprintf("tile=%dx%d", cols, rows),
		"-q:v", "5",
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		log.Printf("Sprite generation failed for media %d: %v", job.MediaID, err)
		return
	}

	// Get actual frame dimensions of the sprite
	// We'll just record the metadata
	spriteMeta := map[string]interface{}{
		"frames":   successfulFrames,
		"cols":     cols,
		"rows":     rows,
		"fw":       160,
		"fh":       90,
		"interval": interval,
	}
	metaJSON, _ := json.Marshal(spriteMeta)

	relativePath := filepath.Join("sprites", fmt.Sprintf("%d_sprite.jpg", job.MediaID))
	t.db.Exec("UPDATE media SET sprite_path = ?, sprite_meta = ? WHERE id = ?",
		relativePath, string(metaJSON), job.MediaID)
}

func (t *Thumbnailer) extractFrame(input string, output string, timestamp float64, width int) error {
	vf := fmt.Sprintf("scale=%d:-1", width)
	cmd := exec.Command("ffmpeg",
		"-y",
		"-ss", fmt.Sprintf("%.2f", timestamp),
		"-i", input,
		"-vframes", "1",
		"-vf", vf,
		"-q:v", "3",
		output,
	)
	return cmd.Run()
}

func (t *Thumbnailer) isBlackFrame(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return true // Assume black if can't read
	}
	defer file.Close()

	img, err := jpeg.Decode(file)
	if err != nil {
		return true
	}

	bounds := img.Bounds()
	totalPixels := bounds.Dx() * bounds.Dy()
	if totalPixels == 0 {
		return true
	}

	var totalBrightness float64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// Convert 16-bit to 8-bit
			brightness := (float64(r>>8) + float64(g>>8) + float64(b>>8)) / 3.0
			totalBrightness += brightness
		}
	}

	avgBrightness := totalBrightness / float64(totalPixels)
	return avgBrightness < 30.0 // Threshold: 30/255
}

func (t *Thumbnailer) Shutdown() {
	close(t.queue)
	t.workerWg.Wait()
}
