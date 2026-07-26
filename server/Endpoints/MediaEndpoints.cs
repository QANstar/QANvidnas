using System.Text.Json;
using Microsoft.EntityFrameworkCore;
using QANvidnasServer.Data;
using QANvidnasServer.DTOs;
using QANvidnasServer.Models;
using QANvidnasServer.Services;

namespace QANvidnasServer.Endpoints;

public static class MediaEndpoints
{
    public static void Map(WebApplication app)
    {
        var protectedGroup = app.MapGroup("/api/media").RequireAuthorization();
        var adminGroup = app.MapGroup("/api/admin").RequireAuthorization();

        // GET /api/media
        protectedGroup.MapGet("", async (
            AppDbContext db, HttpContext http,
            string? type, string? sort, string? order,
            int page = 1, int pageSize = 50) =>
        {
            if (pageSize > 200) pageSize = 200;

            var query = db.Media.Where(m => !m.Deleted);

            if (type == "video" || type == "audio")
                query = query.Where(m => m.Type == type);

            query = (sort, order) switch
            {
                ("title", "asc") => query.OrderBy(m => m.Title),
                ("title", "desc") => query.OrderByDescending(m => m.Title),
                ("duration", "asc") => query.OrderBy(m => m.Duration),
                ("duration", "desc") => query.OrderByDescending(m => m.Duration),
                ("file_size", "asc") => query.OrderBy(m => m.FileSize),
                ("file_size", "desc") => query.OrderByDescending(m => m.FileSize),
                ("created_at", "asc") => query.OrderBy(m => m.CreatedAt),
                _ => query.OrderByDescending(m => m.CreatedAt),
            };

            var total = await query.CountAsync();
            var items = await query.Skip((page - 1) * pageSize).Take(pageSize)
                .Include(m => m.MediaTags).ThenInclude(mt => mt.Tag)
                .ToListAsync();

            var dtos = items.Select(MapToDto).ToList();

            return Results.Ok(new MediaListResponse
            {
                Items = dtos,
                Total = total,
                Page = page,
                PageSize = pageSize,
                TotalPages = (int)Math.Ceiling((double)total / pageSize)
            });
        });

        // GET /api/media/{id}
        protectedGroup.MapGet("/{id:long}", async (long id, AppDbContext db) =>
        {
            var media = await db.Media.Include(m => m.MediaTags).ThenInclude(mt => mt.Tag)
                .FirstOrDefaultAsync(m => m.Id == id && !m.Deleted);

            if (media == null)
                return Results.Json(new { error = "媒体不存在" }, statusCode: 404);

            return Results.Ok(MapToDto(media));
        });

        // PUT /api/media/{id}
        protectedGroup.MapPut("/{id:long}", async (long id, MediaUpdateRequest req, AppDbContext db) =>
        {
            var media = await db.Media.Include(m => m.MediaTags)
                .FirstOrDefaultAsync(m => m.Id == id && !m.Deleted);

            if (media == null)
                return Results.Json(new { error = "媒体不存在" }, statusCode: 404);

            if (req.Title != null)
                media.Title = req.Title;
            if (req.Description != null)
                media.Description = req.Description;
            media.UpdatedAt = DateTime.UtcNow;

            if (req.TagIds != null)
            {
                db.MediaTags.RemoveRange(media.MediaTags);
                foreach (var tagId in req.TagIds)
                {
                    db.MediaTags.Add(new MediaTag { MediaId = id, TagId = tagId });
                }
            }

            await db.SaveChangesAsync();
            return Results.Ok(new { message = "更新成功" });
        });

        // POST /api/media/{id}/cover
        protectedGroup.MapPost("/{id:long}/cover", async (long id, HttpRequest request, AppDbContext db, IConfiguration config) =>
        {
            var media = await db.Media.FirstOrDefaultAsync(m => m.Id == id && !m.Deleted);
            if (media == null)
                return Results.Json(new { error = "媒体不存在" }, statusCode: 404);

            var file = request.Form.Files.GetFile("cover");
            if (file == null || file.Length == 0)
                return Results.Json(new { error = "请选择封面图片" }, statusCode: 400);

            var ext = Path.GetExtension(file.FileName).ToLowerInvariant();
            if (ext != ".jpg" && ext != ".jpeg" && ext != ".png")
                return Results.Json(new { error = "仅支持 JPG 和 PNG 格式" }, statusCode: 400);

            var dataDir = config.GetValue<string>("Storage:DataDir") ?? "data";
            var coverDir = Path.Combine(dataDir, "covers");
            Directory.CreateDirectory(coverDir);

            var fileName = $"{id}_custom{ext}";
            var filePath = Path.Combine(coverDir, fileName);

            using var stream = new FileStream(filePath, FileMode.Create);
            await file.CopyToAsync(stream);

            media.CoverPath = Path.Combine("covers", fileName).Replace('\\', '/');
            media.UpdatedAt = DateTime.UtcNow;
            await db.SaveChangesAsync();

            return Results.Ok(new { message = "封面上传成功", cover_path = media.CoverPath });
        });

        // DELETE /api/media/{id}/cover
        protectedGroup.MapDelete("/{id:long}/cover", async (long id, AppDbContext db, IConfiguration config) =>
        {
            var media = await db.Media.FindAsync(id);
            if (media == null)
                return Results.Json(new { error = "媒体不存在" }, statusCode: 404);

            if (!string.IsNullOrEmpty(media.CoverPath))
            {
                var dataDir = config.GetValue<string>("Storage:DataDir") ?? "data";
                var fullPath = Path.Combine(dataDir, media.CoverPath);
                if (File.Exists(fullPath)) File.Delete(fullPath);
            }

            media.CoverPath = string.Empty;
            await db.SaveChangesAsync();

            return Results.Ok(new { message = "已恢复自动封面" });
        });

        // GET /api/media/{id}/sprite
        protectedGroup.MapGet("/{id:long}/sprite", async (long id, AppDbContext db) =>
        {
            var media = await db.Media.FirstOrDefaultAsync(m => m.Id == id && !m.Deleted);
            if (media == null)
                return Results.Json(new { error = "媒体不存在" }, statusCode: 404);

            if (string.IsNullOrEmpty(media.SpriteMeta))
                return Results.Ok(new { sprite_url = "", meta = (object?)null });

            return Results.Ok(new
            {
                sprite_url = $"/api/stream/cover/{id}?type=sprite",
                meta = JsonSerializer.Deserialize<object>(media.SpriteMeta)
            });
        });

        // POST /api/media/scan
        protectedGroup.MapPost("/scan", (ScannerService scanner) =>
        {
            _ = scanner.FullScanAsync();
            return Results.Ok(new { message = "扫描已启动", status = "scanning" });
        });

        // GET /api/media/scan/progress
        protectedGroup.MapGet("/scan/progress", (ScannerService scanner) =>
        {
            return Results.Ok(scanner.GetProgress());
        });

        // ─── Admin: Scan Folders ───
        adminGroup.MapGet("/scan-folders", async (AppDbContext db, HttpContext http) =>
        {
            if (!IsAdmin(http)) return Results.Json(new { error = "需要管理员权限" }, statusCode: 403);

            var folders = await db.ScanFolders.OrderBy(f => f.Id).Select(f => new
            {
                f.Id, f.Path, last_scan_at = f.LastScanAt, f.Status
            }).ToListAsync();

            return Results.Ok(new { items = folders });
        });

        adminGroup.MapPost("/scan-folders", async (ScanFolderRequest req, AppDbContext db, HttpContext http, ScannerService scanner) =>
        {
            if (!IsAdmin(http)) return Results.Json(new { error = "需要管理员权限" }, statusCode: 403);

            if (!Directory.Exists(req.Path))
                return Results.Json(new { error = "路径不存在或无法访问" }, statusCode: 400);

            var exists = await db.ScanFolders.AnyAsync(f => f.Path == req.Path);
            if (!exists)
            {
                db.ScanFolders.Add(new ScanFolder { Path = req.Path });
                await db.SaveChangesAsync();
            }

            _ = scanner.FullScanAsync();
            return Results.Ok(new { message = "添加成功，扫描已启动" });
        });

        adminGroup.MapDelete("/scan-folders/{id:long}", async (long id, AppDbContext db, HttpContext http) =>
        {
            if (!IsAdmin(http)) return Results.Json(new { error = "需要管理员权限" }, statusCode: 403);

            var folder = await db.ScanFolders.FindAsync(id);
            if (folder != null)
            {
                db.ScanFolders.Remove(folder);
                await db.SaveChangesAsync();
            }
            return Results.Ok(new { message = "已删除" });
        });
    }

    private static MediaDto MapToDto(Media m)
    {
        return new MediaDto
        {
            Id = m.Id,
            Title = m.Title,
            Description = m.Description,
            Type = m.Type,
            Path = m.Path,
            Duration = m.Duration,
            Resolution = m.Resolution,
            CoverPath = m.CoverPath,
            SpritePath = m.SpritePath,
            SpriteMeta = m.SpriteMeta,
            FileSize = m.FileSize,
            Codec = m.Codec,
            Bitrate = m.Bitrate,
            CreatedAt = m.CreatedAt,
            UpdatedAt = m.UpdatedAt,
            Tags = m.MediaTags.Select(mt => new TagDto
            {
                Id = mt.TagId,
                Name = mt.Tag?.Name ?? "",
                Color = mt.Tag?.Color ?? "#6C5CE7"
            }).ToList()
        };
    }

    private static bool IsAdmin(HttpContext http)
    {
        return http.User.FindFirst("is_admin")?.Value == "true";
    }
}
