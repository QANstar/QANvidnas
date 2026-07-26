using System.ComponentModel.DataAnnotations;

namespace QANvidnasServer.Models;

public class InviteCode
{
    [Key]
    public string Code { get; set; } = string.Empty;

    public string Description { get; set; } = string.Empty;

    public int MaxUses { get; set; } = 1;

    public int Used { get; set; }

    public string ExpiresAt { get; set; } = string.Empty;
}
