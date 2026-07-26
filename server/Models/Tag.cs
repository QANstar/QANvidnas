using System.ComponentModel.DataAnnotations;

namespace QANvidnasServer.Models;

public class Tag
{
    [Key]
    public long Id { get; set; }

    [Required]
    public string Name { get; set; } = string.Empty;

    public string Color { get; set; } = "#6C5CE7";

    public List<MediaTag> MediaTags { get; set; } = new();
}
