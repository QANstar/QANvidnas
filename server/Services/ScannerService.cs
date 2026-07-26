using System.Diagnostics;
using System.Text.Json;
using Microsoft.EntityFrameworkCore;
using QANvidnasServer.Data;
using QANvidnasServer.DTOs;
using QANvidnasServer.Models;

namespace QANvidnasServer.Services;

public class ScannerService
{
    private static readonly Dictionary<string, string> SupportedExtensions = new(StringComparer.OrdinalIgnoreCase)
    {
        [".mp4"] = "video", [".mkv"] = "video", [".avi"] = "video", [".mov"] = "video",
        [".wmv"] = "video", [".flv"] = "video", [".webm"] = "video", [".m4v"] = "video",
        [".mpg"] = "video", [".mpeg"] = "video", [".ts"] = "video",
        [".mp3"] = "audio", [".flac"] = "audio", [".aac"] = "audio", [".ogg"] = "audio",
        [".wav"] = "audio", [".m4a"] = "audio", [".wma"] = "audio", [".opus"] = "audio",
    };

    private readonly IServiceScopeFactory _scopeFactory;
    private ScanProgress _progress = new();
    private readonly object _lock = new();

    public ScannerService(IServiceScopeFactory scopeFactory)
    {
        _scopeFactory = scopeFactory;
    }

    public ScanProgress GetProgress()
    {
        lock (_lock) return new ScanProgress
        {
            Status = _progress.Status,
            Total = _progress.Total,
            Processed = _progress.Processed,
            Message = _progress.Message
        };
    }

    public async Task FullScanAsync()
    {
        lock (_lock) _progress = new ScanProgress { Status = "scanning" };

        using var scope = _scopeFactory.CreateScope();
        var db = scope.ServiceProvider.GetRequiredService<AppDbContext>();
        var thumbnailer = scope.ServiceProvider.GetRequiredService<ThumbnailService>();

        var folders = await db.ScanFolders.ToListAsync();

        // Count total files
        int total = 0;
        foreach (var folder in folders)
        {
            if (Directory.Exists(folder.Path))
            {
                total += Directory.GetFiles(folder.Path, "*.*", SearchOption.AllDirectories)
                    .Count(f => SupportedExtensions.ContainsKey(Path.GetExtension(f)));
            }
        }

        lock (_lock) _progress.Total = total;

        int processed = 0;
        foreach (var folder in folders)
        {
            if (!Directory.Exists(folder.Path)) continue;

            foreach (var filePath in Directory.GetFiles(folder.Path, "*.*", SearchOption.AllDirectories))
            {
                var ext = Path.GetExtension(filePath);
                if (!SupportedExtensions.TryGetValue(ext, out var mediaType)) continue;

                // Check existing
                var exists = await db.Media.AnyAsync(m => m.Path == filePath && !m.Deleted);
                if (exists) { processed++; continue; }

                var meta = ExtractMetadata(filePath, mediaType);
                var fileName = Path.GetFileNameWithoutExtension(filePath);
                var info = new FileInfo(filePath);
                var now = DateTime.UtcNow;

                var media = new Media
                {
                    Title = fileName,
                    Type = mediaType,
                    Path = filePath,
                    Duration = meta.Duration,
                    Resolution = meta.Resolution,
                    Codec = meta.Codec,
                    Bitrate = meta.Bitrate,
                    FileSize = info.Length,
                    CreatedAt = now,
                    UpdatedAt = now,
                };

                db.Media.Add(media);
                await db.SaveChangesAsync();

                // Generate thumbnails async
                if (mediaType == "video")
                {
                    _ = thumbnailer.GenerateCoverAsync(media.Id, filePath, meta.Duration);
                    _ = thumbnailer.GenerateSpriteAsync(media.Id, filePath, meta.Duration);
                }

                processed++;
                lock (_lock)
                {
                    _progress.Processed = processed;
                    _progress.Status = "scanning";
                }
            }

            folder.LastScanAt = DateTime.UtcNow;
            folder.Status = "idle";
        }

        await db.SaveChangesAsync();

        lock (_lock) _progress.Status = "complete";
    }

    private static MediaMeta ExtractMetadata(string path, string mediaType)
    {
        var meta = new MediaMeta();

        try
        {
            using var process = new Process
            {
                StartInfo = new ProcessStartInfo
                {
                    FileName = "ffprobe",
                    Arguments = $"-v quiet -print_format json -show_format -show_streams \"{path}\"",
                    RedirectStandardOutput = true,
                    UseShellExecute = false,
                    CreateNoWindow = true,
                }
            };

            process.Start();
            var output = process.StandardOutput.ReadToEnd();
            process.WaitForExit(5000);

            if (string.IsNullOrEmpty(output)) return meta;

            using var doc = JsonDocument.Parse(output);
            var root = doc.RootElement;

            if (root.TryGetProperty("format", out var format))
            {
                if (format.TryGetProperty("duration", out var dur) && dur.TryGetDouble(out var d))
                    meta.Duration = d;
                if (format.TryGetProperty("bit_rate", out var br) && br.TryGetInt64(out var b))
                    meta.Bitrate = b;
            }

            if (root.TryGetProperty("streams", out var streams))
            {
                foreach (var stream in streams.EnumerateArray())
                {
                    var codecType = stream.TryGetProperty("codec_type", out var ct) ? ct.GetString() : "";
                    var codecName = stream.TryGetProperty("codec_name", out var cn) ? cn.GetString() : "";

                    if (mediaType == "video" && codecType == "video")
                    {
                        meta.Codec = codecName ?? "";
                        var w = stream.TryGetProperty("width", out var sw) ? sw.GetInt32() : 0;
                        var h = stream.TryGetProperty("height", out var sh) ? sh.GetInt32() : 0;
                        if (w > 0 && h > 0) meta.Resolution = $"{w}x{h}";
                        break;
                    }
                    if (mediaType == "audio" && codecType == "audio")
                    {
                        meta.Codec = codecName ?? "";
                        break;
                    }
                }
            }
        }
        catch
        {
            // ffprobe not available or failed
        }

        return meta;
    }

    private class MediaMeta
    {
        public double Duration { get; set; }
        public string Resolution { get; set; } = string.Empty;
        public string Codec { get; set; } = string.Empty;
        public long Bitrate { get; set; }
    }
}
