package scanner

import (
	"encoding/json"
	"log"
	"os/exec"
	"strconv"
)

type StreamInfo struct {
	CodecType string `json:"codec_type"`
	CodecName string `json:"codec_name"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	BitRate   string `json:"bit_rate"`
}

type FormatInfo struct {
	Duration string `json:"duration"`
	BitRate  string `json:"bit_rate"`
	Size     string `json:"size"`
}

type ffprobeOutput struct {
	Streams []StreamInfo `json:"streams"`
	Format  FormatInfo   `json:"format"`
}

type MediaMeta struct {
	Duration   float64
	Resolution string
	Codec      string
	Bitrate    int64
}

func extractMetadata(path, mediaType string) *MediaMeta {
	meta := &MediaMeta{}

	// Try ffprobe
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	)

	output, err := cmd.Output()
	if err != nil {
		log.Printf("ffprobe failed for %s: %v", path, err)
		// Return basic metadata from file info
		return meta
	}

	var probe ffprobeOutput
	if err := json.Unmarshal(output, &probe); err != nil {
		log.Printf("ffprobe parse failed for %s: %v", path, err)
		return meta
	}

	// Duration
	if probe.Format.Duration != "" {
		if d, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil {
			meta.Duration = d
		}
	}

	// Bitrate
	if probe.Format.BitRate != "" {
		if b, err := strconv.ParseInt(probe.Format.BitRate, 10, 64); err == nil {
			meta.Bitrate = b
		}
	}

	// Find video/audio stream info
	for _, stream := range probe.Streams {
		if mediaType == "video" && stream.CodecType == "video" {
			meta.Codec = stream.CodecName
			if stream.Width > 0 && stream.Height > 0 {
				meta.Resolution = strconv.Itoa(stream.Width) + "x" + strconv.Itoa(stream.Height)
			}
			break
		}
		if mediaType == "audio" && stream.CodecType == "audio" {
			meta.Codec = stream.CodecName
			break
		}
	}

	return meta
}
