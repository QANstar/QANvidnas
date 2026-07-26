using System.ComponentModel.DataAnnotations;

namespace QANvidnasServer.Models;

public class User
{
    [Key]
    public long Id { get; set; }

    [Required, MaxLength(32)]
    public string Username { get; set; } = string.Empty;

    [Required]
    public string PasswordHash { get; set; } = string.Empty;

    public bool IsAdmin { get; set; }

    public DateTime CreatedAt { get; set; } = DateTime.UtcNow;
}
