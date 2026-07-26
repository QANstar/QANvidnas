using Microsoft.EntityFrameworkCore;
using QANvidnasServer.Data;
using QANvidnasServer.DTOs;

namespace QANvidnasServer.Endpoints;

public static class UploadEndpoints
{
    private static readonly HashSet<string> SupportedExtensions = new(StringComparer.OrdinalIgnoreCase)
    {
        ".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm", ".m4v", ".mpg", ".mpeg", ".ts",
        ".mp3", ".flac", ".aac", ".ogg", ".wav", ".m4a", ".wma", ".opus",
    };

    private const long MaxUploadSize = 10L * 1024 * 1024 * 1024; // 10GB

    public static void Map(WebApplication app)
    {
        var group = app.MapGroup("/api/upload").RequireAuthorization();

        group.MapPost("/", async (HttpRequest request, AppDbContext db) =>
        {
            var form = await request.ReadFormAsync();
            var targetDir = form["target_dir"].FirstOrDefault();
            if (string.IsNullOrWhiteSpace(targetDir))
                return Results.Json(new { error = "请指定目标目录" }, statusCode: 400);

            // Validate target directory is within scan folders
            var scanFolders = await db.ScanFolders.Select(f => f.Path).ToListAsync();
            var valid = scanFolders.Any(sf =>
                targetDir.StartsWith(sf, StringComparison.OrdinalIgnoreCase));
            if (!valid)
                return Results.Json(new { error = "目标目录不在扫描范围内" }, statusCode: 400);

            Directory.CreateDirectory(targetDir);

            var file = form.Files.GetFile("file");
            if (file == null || file.Length == 0)
                return Results.Json(new { error = "请选择文件" }, statusCode: 400);

            var ext = Path.GetExtension(file.FileName).ToLowerInvariant();
            if (!SupportedExtensions.Contains(ext))
                return Results.Json(new { error = "不支持的格式: " + ext }, statusCode: 400);

            if (file.Length > MaxUploadSize)
                return Results.Json(new
                {
                    error = $"文件过大，最大支持 10GB（当前: {file.Length / (1024.0 * 1024 * 1024):F1}GB）"
                }, statusCode: 400);

            var destPath = Path.Combine(targetDir, file.FileName);
            await using var stream = new FileStream(destPath, FileMode.Create);
            await file.CopyToAsync(stream);

            return Results.Ok(new UploadResponse("上传成功", destPath, file.Length, file.FileName));
        }).DisableAntiforgery();
    }
}
