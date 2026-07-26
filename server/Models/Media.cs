using System.ComponentModel.DataAnnotations;
using System.Text.Json.Serialization;

namespace QANvidnasServer.Models;

public class Media
{
    [Key]
    public long Id { get; set; }

    [Required]
    public string Title { get; set; } = string.Empty;

    public string Description { get; set; } = string.Empty;

    [Required]
    public string Type { get; set; } = string.Empty; // "video" or "audio"

    [Required]
    public string Path { get; set; } = string.Empty;

    public double Duration { get; set; }

    public string Resolution { get; set; } = string.Empty;

    public string CoverPath { get; set; } = string.Empty;

    public string SpritePath { get; set; } = string.Empty;

    public string SpriteMeta { get; set; } = string.Empty; // JSON

    public long FileSize { get; set; }

    public string Codec { get; set; } = string.Empty;

    public long Bitrate { get; set; }

    public bool Deleted { get; set; }

    public DateTime CreatedAt { get; set; } = DateTime.UtcNow;

    public DateTime UpdatedAt { get; set; } = DateTime.UtcNow;

    // Navigation
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public List<MediaTag> MediaTags { get; set; } = new();
}
