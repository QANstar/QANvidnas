using System.ComponentModel.DataAnnotations;

namespace QANvidnasServer.Models;

public class ScanFolder
{
    [Key]
    public long Id { get; set; }

    [Required]
    public string Path { get; set; } = string.Empty;

    public DateTime? LastScanAt { get; set; }

    public string Status { get; set; } = "idle"; // idle, scanning, error
}
