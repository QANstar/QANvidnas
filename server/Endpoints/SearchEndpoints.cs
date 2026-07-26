using Microsoft.EntityFrameworkCore;
using QANvidnasServer.Data;
using QANvidnasServer.DTOs;
using QANvidnasServer.Models;

namespace QANvidnasServer.Endpoints;

public static class SearchEndpoints
{
    public static void Map(WebApplication app)
    {
        var protectedGroup = app.MapGroup("/api").RequireAuthorization();

        // GET /api/search
        protectedGroup.MapGet("/search", async (
            AppDbContext db,
            string? q, string? tags, string? type,
            int page = 1, int pageSize = 50) =>
        {
            var query = db.Media.AsQueryable().Where(m => !m.Deleted);

            if (!string.IsNullOrWhiteSpace(q))
            {
                var searchTerm = q.Trim();
                query = query.Where(m =>
                    m.Title.Contains(searchTerm) ||
                    m.Description.Contains(searchTerm) ||
                    m.MediaTags.Any(mt => mt.Tag.Name.Contains(searchTerm)));
            }

            if (!string.IsNullOrWhiteSpace(tags))
            {
                var tagList = tags.Split(',').Select(t => t.Trim()).Where(t => t.Length > 0).ToList();
                foreach (var tag in tagList)
                {
                    query = query.Where(m => m.MediaTags.Any(mt => mt.Tag.Name == tag));
                }
            }

            if (type == "video" || type == "audio")
                query = query.Where(m => m.Type == type);

            var total = await query.CountAsync();

            var items = await query
                .OrderByDescending(m => m.CreatedAt)
                .Skip((page - 1) * pageSize).Take(pageSize)
                .Include(m => m.MediaTags).ThenInclude(mt => mt.Tag)
                .Select(m => new SearchItem
                {
                    Id = m.Id,
                    Title = m.Title,
                    Description = m.Description,
                    Type = m.Type,
                    Path = m.Path,
                    Duration = m.Duration,
                    Resolution = m.Resolution,
                    CoverPath = m.CoverPath,
                    FileSize = m.FileSize,
                    Codec = m.Codec,
                    Bitrate = m.Bitrate,
                    CreatedAt = m.CreatedAt.ToString("yyyy-MM-ddTHH:mm:ssZ")
                })
                .ToListAsync();

            return Results.Ok(new SearchResponse
            {
                Items = items,
                Total = total,
                Page = page,
                Query = q ?? string.Empty
            });
        });

        // GET /api/tags
        protectedGroup.MapGet("/tags", async (AppDbContext db) =>
        {
            var tags = await db.Tags.OrderBy(t => t.Name)
                .Select(t => new TagDto { Id = t.Id, Name = t.Name, Color = t.Color })
                .ToListAsync();

            return Results.Ok(new { items = tags });
        });

        // POST /api/tags
        protectedGroup.MapPost("/tags", async (CreateTagRequest req, AppDbContext db) =>
        {
            var color = string.IsNullOrWhiteSpace(req.Color) ? "#6C5CE7" : req.Color;

            var exists = await db.Tags.AnyAsync(t => t.Name == req.Name);
            if (exists)
                return Results.Json(new { error = "标签名已存在" }, statusCode: 409);

            var tag = new Tag { Name = req.Name, Color = color };
            db.Tags.Add(tag);
            await db.SaveChangesAsync();

            return Results.Ok(new TagDto { Id = tag.Id, Name = tag.Name, Color = tag.Color });
        });

        // DELETE /api/tags/{id}
        protectedGroup.MapDelete("/tags/{id:long}", async (long id, AppDbContext db) =>
        {
            var tag = await db.Tags.FindAsync(id);
            if (tag != null)
            {
                db.Tags.Remove(tag);
                await db.SaveChangesAsync();
            }
            return Results.Ok(new { message = "已删除" });
        });
    }
}
