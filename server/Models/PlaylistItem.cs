using System.Text.Json.Serialization;

namespace QANvidnasServer.Models;

public class PlaylistItem
{
    public long PlaylistId { get; set; }
    public long MediaId { get; set; }
    public int Position { get; set; }

    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public Media? Media { get; set; }
}
