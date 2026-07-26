using System.ComponentModel.DataAnnotations;
using System.Text.Json.Serialization;

namespace QANvidnasServer.Models;

public class Playlist
{
    [Key]
    public long Id { get; set; }

    [Required]
    public string Name { get; set; } = string.Empty;

    public string FolderPath { get; set; } = string.Empty;

    public long UserId { get; set; }

    public string PlayMode { get; set; } = "sequential";

    public DateTime CreatedAt { get; set; } = DateTime.UtcNow;

    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public List<PlaylistItem> Items { get; set; } = new();
}
