using System.ComponentModel.DataAnnotations;

namespace QANvidnasServer.Models;

public class DeviceCode
{
    [Key]
    public string Code { get; set; } = string.Empty;

    public long? UserId { get; set; }

    public DateTime ExpiresAt { get; set; }

    public bool Used { get; set; }
}
