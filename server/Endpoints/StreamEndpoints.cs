using Microsoft.EntityFrameworkCore;
using QANvidnasServer.Data;

namespace QANvidnasServer.Endpoints;

public static class StreamEndpoints
{
    public static void Map(WebApplication app)
    {
        var group = app.MapGroup("/api/stream").RequireAuthorization();

        // GET /api/stream/video/{id}
        group.MapGet("/video/{id:long}", (long id, AppDbContext db, HttpContext http) =>
            ServeStream(id, "video", db, http));

        // GET /api/stream/audio/{id}
        group.MapGet("/audio/{id:long}", (long id, AppDbContext db, HttpContext http) =>
            ServeStream(id, "audio", db, http));

        // GET /api/stream/cover/{id}
        group.MapGet("/cover/{id:long}", async (long id, AppDbContext db, IConfiguration config, HttpContext http) =>
        {
            var type = http.Request.Query["type"].FirstOrDefault() ?? "cover";

            var media = await db.Media.FirstOrDefaultAsync(m => m.Id == id && !m.Deleted);
            if (media == null)
                return Results.Json(new { error = "媒体不存在" }, statusCode: 404);

            string? filePath = null;
            if (type == "sprite" && !string.IsNullOrEmpty(media.SpritePath))
                filePath = media.SpritePath;
            else if (!string.IsNullOrEmpty(media.CoverPath))
                filePath = media.CoverPath;

            // Resolve relative paths
            if (filePath != null && !Path.IsPathRooted(filePath) && !filePath.Contains("/media"))
            {
                var dataDir = config.GetValue<string>("Storage:DataDir") ?? "data";
                filePath = Path.Combine(dataDir, filePath);
            }

            if (filePath == null || !File.Exists(filePath))
            {
                http.Response.ContentType = "image/svg+xml";
                return Results.Content(
                    media.Type == "audio" ? AudioPlaceholderSVG : VideoPlaceholderSVG,
                    "image/svg+xml");
            }

            var ext = Path.GetExtension(filePath).ToLowerInvariant();
            var contentType = ext switch
            {
                ".jpg" or ".jpeg" => "image/jpeg",
                ".png" => "image/png",
                _ => "image/jpeg"
            };

            http.Response.Headers.CacheControl = "public, max-age=86400";
            return Results.File(filePath, contentType);
        });
    }

    private static async Task<IResult> ServeStream(long id, string expectedType, AppDbContext db, HttpContext http)
    {
        var media = await db.Media.FirstOrDefaultAsync(m => m.Id == id && !m.Deleted);
        if (media == null)
            return Results.Json(new { error = "媒体不存在" }, statusCode: 404);

        if (media.Type != expectedType)
            return Results.Json(new { error = "媒体类型不匹配" }, statusCode: 400);

        if (media.Path.Contains(".."))
            return Results.Json(new { error = "非法路径" }, statusCode: 403);

        if (!File.Exists(media.Path))
            return Results.Json(new { error = "文件不存在" }, statusCode: 404);

        var fileInfo = new FileInfo(media.Path);
        var fileSize = fileInfo.Length;
        var ext = Path.GetExtension(media.Path).ToLowerInvariant();
        var contentType = ext switch
        {
            ".mp4" => "video/mp4",
            ".mkv" => "video/x-matroska",
            ".avi" => "video/x-msvideo",
            ".mov" => "video/quicktime",
            ".webm" => "video/webm",
            ".flv" => "video/x-flv",
            ".mp3" => "audio/mpeg",
            ".flac" => "audio/flac",
            ".aac" => "audio/aac",
            ".ogg" => "audio/ogg",
            ".wav" => "audio/wav",
            ".m4a" => "audio/mp4",
            _ => "application/octet-stream"
        };

        http.Response.Headers.AcceptRanges = "bytes";

        var rangeHeader = http.Request.Headers.Range.FirstOrDefault();
        if (string.IsNullOrEmpty(rangeHeader))
        {
            http.Response.ContentType = contentType;
            http.Response.Headers.ContentLength = fileSize;
            return Results.File(media.Path, contentType);
        }

        // Parse Range
        var rangeStr = rangeHeader.Replace("bytes=", "");
        var parts = rangeStr.Split('-');
        var start = long.Parse(parts[0]);
        var end = parts.Length > 1 && !string.IsNullOrEmpty(parts[1])
            ? long.Parse(parts[1]) : fileSize - 1;

        if (end >= fileSize) end = fileSize - 1;
        if (start > end || start >= fileSize)
            return Results.StatusCode(416);

        var chunkSize = end - start + 1;

        http.Response.StatusCode = 206;
        http.Response.ContentType = contentType;
        http.Response.Headers.ContentRange = $"bytes {start}-{end}/{fileSize}";
        http.Response.Headers.ContentLength = chunkSize;

        var fileStream = new FileStream(media.Path, FileMode.Open, FileAccess.Read, FileShare.Read);
        fileStream.Seek(start, SeekOrigin.Begin);

        return Results.Stream(fileStream, contentType);
    }

    private const string VideoPlaceholderSVG = @"<svg xmlns=""http://www.w3.org/2000/svg"" width=""480"" height=""270"" viewBox=""0 0 480 270"">
  <rect fill=""#1A1A2E"" width=""480"" height=""270""/>
  <polygon points=""200,90 200,180 310,135"" fill=""#FF6B9D"" opacity=""0.8""/>
  <text x=""240"" y=""230"" text-anchor=""middle"" fill=""#666"" font-size=""14"" font-family=""sans-serif"">暂无封面</text>
</svg>";

    private const string AudioPlaceholderSVG = @"<svg xmlns=""http://www.w3.org/2000/svg"" width=""480"" height=""270"" viewBox=""0 0 480 270"">
  <rect fill=""#1A1A2E"" width=""480"" height=""270""/>
  <circle cx=""240"" cy=""120"" r=""40"" fill=""#6C5CE7"" opacity=""0.8""/>
  <text x=""240"" y=""190"" text-anchor=""middle"" fill=""#FF6B9D"" font-size=""14"" font-family=""sans-serif"">🎵</text>
  <text x=""240"" y=""230"" text-anchor=""middle"" fill=""#666"" font-size=""14"" font-family=""sans-serif"">音频文件</text>
</svg>";
}
