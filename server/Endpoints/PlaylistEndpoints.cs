using System.Security.Claims;
using Microsoft.EntityFrameworkCore;
using QANvidnasServer.Data;
using QANvidnasServer.DTOs;
using QANvidnasServer.Models;

namespace QANvidnasServer.Endpoints;

public static class PlaylistEndpoints
{
    private static readonly HashSet<string> ValidModes = new() { "sequential", "loop", "random", "single-loop" };

    public static void Map(WebApplication app)
    {
        var group = app.MapGroup("/api/playlists").RequireAuthorization();

        // GET /api/playlists
        group.MapGet("", async (AppDbContext db, HttpContext http) =>
        {
            var userId = GetUserId(http);
            var playlists = await db.Playlists
                .Where(p => p.UserId == userId)
                .OrderByDescending(p => p.CreatedAt)
                .Select(p => new
                {
                    p.Id, p.Name, p.FolderPath, p.UserId, p.PlayMode, p.CreatedAt
                })
                .ToListAsync();

            return Results.Ok(new { items = playlists });
        });

        // GET /api/playlists/{id}
        group.MapGet("/{id:long}", async (long id, AppDbContext db, HttpContext http) =>
        {
            var userId = GetUserId(http);
            var playlist = await db.Playlists
                .FirstOrDefaultAsync(p => p.Id == id && p.UserId == userId);

            if (playlist == null)
                return Results.Json(new { error = "播放列表不存在" }, statusCode: 404);

            var items = await db.PlaylistItems
                .Where(pi => pi.PlaylistId == id)
                .OrderBy(pi => pi.Position)
                .Join(db.Media.Where(m => !m.Deleted),
                    pi => pi.MediaId, m => m.Id, (pi, m) => new
                    {
                        pi.PlaylistId,
                        pi.MediaId,
                        pi.Position,
                        Media = new
                        {
                            m.Id, m.Title, m.Description, m.Type,
                            m.Duration, m.Resolution, m.CoverPath, m.FileSize
                        }
                    })
                .ToListAsync();

            if (playlist.PlayMode == "random")
            {
                items = items.OrderBy(_ => Random.Shared.Next()).ToList();
                for (int i = 0; i < items.Count; i++)
                {
                    items[i] = items[i] with { };
                }
            }

            return Results.Ok(new
            {
                playlist.Id, playlist.Name, playlist.FolderPath,
                playlist.UserId, playlist.PlayMode, playlist.CreatedAt,
                Items = items
            });
        });

        // POST /api/playlists
        group.MapPost("", async (CreatePlaylistRequest req, AppDbContext db, HttpContext http) =>
        {
            var userId = GetUserId(http);

            var playlist = new Playlist
            {
                Name = req.Name,
                FolderPath = req.FolderPath,
                UserId = userId,
                PlayMode = "sequential",
            };
            db.Playlists.Add(playlist);
            await db.SaveChangesAsync();

            // Populate playlist with media from folder
            _ = PopulatePlaylistAsync(playlist.Id, req.FolderPath, db);

            return Results.Ok(new
            {
                playlist.Id, playlist.Name, playlist.FolderPath,
                message = "播放列表创建中，媒体文件正在添加"
            });
        });

        // PUT /api/playlists/{id}
        group.MapPut("/{id:long}", async (long id, UpdatePlaylistRequest req, AppDbContext db, HttpContext http) =>
        {
            if (!ValidModes.Contains(req.PlayMode))
                return Results.Json(new { error = "无效的播放模式" }, statusCode: 400);

            var userId = GetUserId(http);
            var playlist = await db.Playlists.FirstOrDefaultAsync(p => p.Id == id && p.UserId == userId);
            if (playlist == null)
                return Results.Json(new { error = "播放列表不存在" }, statusCode: 404);

            playlist.PlayMode = req.PlayMode;
            await db.SaveChangesAsync();

            return Results.Ok(new { message = "更新成功" });
        });

        // DELETE /api/playlists/{id}
        group.MapDelete("/{id:long}", async (long id, AppDbContext db, HttpContext http) =>
        {
            var userId = GetUserId(http);
            var playlist = await db.Playlists.FirstOrDefaultAsync(p => p.Id == id && p.UserId == userId);
            if (playlist != null)
            {
                db.Playlists.Remove(playlist);
                await db.SaveChangesAsync();
            }
            return Results.Ok(new { message = "已删除" });
        });

        // POST /api/playlists/play-now
        group.MapPost("/play-now", async (PlayNowRequest req, AppDbContext db, HttpContext http) =>
        {
            var userId = GetUserId(http);

            var folderName = req.FolderPath;
            var lastSep = folderName.LastIndexOfAny(['/', '\\']);
            if (lastSep >= 0) folderName = folderName[(lastSep + 1)..];

            var playlist = new Playlist
            {
                Name = folderName,
                FolderPath = req.FolderPath,
                UserId = userId,
                PlayMode = "sequential",
            };
            db.Playlists.Add(playlist);
            await db.SaveChangesAsync();

            await PopulatePlaylistAsync(playlist.Id, req.FolderPath, db);

            var firstItem = await db.PlaylistItems
                .Where(pi => pi.PlaylistId == playlist.Id)
                .OrderBy(pi => pi.Position)
                .FirstOrDefaultAsync();

            if (firstItem == null)
            {
                db.Playlists.Remove(playlist);
                await db.SaveChangesAsync();
                return Results.Json(new { error = "文件夹中没有可播放的媒体" }, statusCode: 404);
            }

            return Results.Ok(new PlayNowResponse(playlist.Id, new PlayNowFirstMedia(firstItem.MediaId)));
        });
    }

    private static async Task PopulatePlaylistAsync(long playlistId, string folderPath, AppDbContext db)
    {
        var mediaIds = await db.Media
            .Where(m => !m.Deleted &&
                        (m.Path.StartsWith(folderPath + "/") || m.Path.StartsWith(folderPath + "\\") ||
                         m.Path.StartsWith(folderPath)))
            .OrderBy(m => m.Path)
            .Select(m => m.Id)
            .ToListAsync();

        int pos = 0;
        foreach (var mediaId in mediaIds)
        {
            var exists = await db.PlaylistItems.AnyAsync(pi =>
                pi.PlaylistId == playlistId && pi.MediaId == mediaId);
            if (!exists)
            {
                db.PlaylistItems.Add(new PlaylistItem
                {
                    PlaylistId = playlistId,
                    MediaId = mediaId,
                    Position = pos
                });
            }
            pos++;
        }
        await db.SaveChangesAsync();
    }

    private static long GetUserId(HttpContext http)
    {
        var claim = http.User.FindFirst("user_id")?.Value;
        return claim != null && long.TryParse(claim, out var id) ? id : 0;
    }
}
