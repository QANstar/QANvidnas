namespace QANvidnasServer.Models;

public class MediaTag
{
    public long MediaId { get; set; }
    public Media Media { get; set; } = null!;

    public long TagId { get; set; }
    public Tag Tag { get; set; } = null!;
}
