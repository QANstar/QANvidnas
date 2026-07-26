using System.Text.Json.Serialization;

namespace QANvidnasServer.DTOs;

// ─── Auth ───
public record SetupRequest(string Username, string Password);
public record LoginRequest(string Username, string Password);
public record RegisterRequest(string InviteCode, string Username, string Password);
public record AuthResponse(string Token);
public record UserInfo(long Id, string Username, bool IsAdmin);
public record SetupCheckResponse(bool SetupRequired);
public record DeviceCodeResponse(string DeviceCode, int ExpiresIn);
public record AuthorizeDeviceCodeRequest(string Code);
public record DeviceCodePollResponse(string Status, string? Token = null);
public record ChangePasswordRequest(string OldPassword, string NewPassword);

// ─── Invite Codes ───
public record CreateInviteCodeRequest(string Code, string? Description, int? MaxUses, string? ExpiresAt);
public record InviteCodeInfo(string Code, string Description, int MaxUses, int Used, string ExpiresAt);

// ─── Media ───
public record MediaUpdateRequest(string? Title, string? Description, List<long>? TagIds);
public record ScanFolderRequest(string Path);

public class MediaListResponse
{
    public List<MediaDto> Items { get; set; } = new();
    public long Total { get; set; }
    public int Page { get; set; }
    public int PageSize { get; set; }
    public int TotalPages { get; set; }
}

public class MediaDto
{
    public long Id { get; set; }
    public string Title { get; set; } = string.Empty;
    public string Description { get; set; } = string.Empty;
    public string Type { get; set; } = string.Empty;
    public string Path { get; set; } = string.Empty;
    public double Duration { get; set; }
    public string Resolution { get; set; } = string.Empty;
    public string CoverPath { get; set; } = string.Empty;
    public string SpritePath { get; set; } = string.Empty;
    public string? SpriteMeta { get; set; }
    public long FileSize { get; set; }
    public string Codec { get; set; } = string.Empty;
    public long Bitrate { get; set; }
    public DateTime CreatedAt { get; set; }
    public DateTime UpdatedAt { get; set; }
    public List<TagDto> Tags { get; set; } = new();
}

public class TagDto
{
    public long Id { get; set; }
    public string Name { get; set; } = string.Empty;
    public string Color { get; set; } = string.Empty;
}

// ─── Search ───
public class SearchResponse
{
    public List<SearchItem> Items { get; set; } = new();
    public long Total { get; set; }
    public int Page { get; set; }
    public string Query { get; set; } = string.Empty;
}

public class SearchItem
{
    public long Id { get; set; }
    public string Title { get; set; } = string.Empty;
    public string Description { get; set; } = string.Empty;
    public string Type { get; set; } = string.Empty;
    public string Path { get; set; } = string.Empty;
    public double Duration { get; set; }
    public string Resolution { get; set; } = string.Empty;
    public string CoverPath { get; set; } = string.Empty;
    public long FileSize { get; set; }
    public string Codec { get; set; } = string.Empty;
    public long Bitrate { get; set; }
    public string CreatedAt { get; set; } = string.Empty;
}

public record CreateTagRequest(string Name, string? Color);

// ─── Playlist ───
public record CreatePlaylistRequest(string Name, string FolderPath);
public record UpdatePlaylistRequest(string PlayMode);
public record PlayNowRequest(string FolderPath);
public record PlayNowResponse(long PlaylistId, PlayNowFirstMedia FirstMedia);
public record PlayNowFirstMedia(long Id);

// ─── Upload ───
public record UploadResponse(string Message, string FilePath, long FileSize, string FileName);

// ─── Scan ───
public class ScanProgress
{
    public string Status { get; set; } = "idle"; // idle, scanning, complete, error
    public int Total { get; set; }
    public int Processed { get; set; }
    public string Message { get; set; } = string.Empty;
}
