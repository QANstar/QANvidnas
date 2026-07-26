using System.Diagnostics;
using System.Text.Json;
using Microsoft.EntityFrameworkCore;
using QANvidnasServer.Data;

namespace QANvidnasServer.Services;

public class ThumbnailService
{
    private readonly IServiceScopeFactory _scopeFactory;
    private readonly IConfiguration _config;

    public ThumbnailService(IServiceScopeFactory scopeFactory, IConfiguration config)
    {
        _scopeFactory = scopeFactory;
        _config = config;
    }

    private string GetDataDir()
    {
        return _config.GetValue<string>("Storage:DataDir") ?? "data";
    }

    public async Task GenerateCoverAsync(long mediaId, string filePath, double duration)
    {
        var dataDir = GetDataDir();
        var coverDir = Path.Combine(dataDir, "covers");
        Directory.CreateDirectory(coverDir);

        var outputPath = Path.Combine(coverDir, $"{mediaId}_cover.jpg");
        var position = duration < 10 ? duration * 0.5 : duration * 0.1;

        if (!await ExtractFrameAsync(filePath, outputPath, position, 480))
        {
            if (duration >= 10)
                await ExtractFrameAsync(filePath, outputPath, duration * 0.3, 480);
        }

        using var scope = _scopeFactory.CreateScope();
        var db = scope.ServiceProvider.GetRequiredService<AppDbContext>();
        var media = await db.Media.FindAsync(mediaId);
        if (media != null)
        {
            media.CoverPath = Path.Combine("covers", $"{mediaId}_cover.jpg").Replace('\\', '/');
            await db.SaveChangesAsync();
        }
    }

    public async Task GenerateSpriteAsync(long mediaId, string filePath, double duration)
    {
        var dataDir = GetDataDir();
        var spriteDir = Path.Combine(dataDir, "sprites");
        Directory.CreateDirectory(spriteDir);

        var tempDir = Path.Combine(spriteDir, $"temp_{mediaId}");
        Directory.CreateDirectory(tempDir);

        try
        {
            var interval = 10.0;
            var totalFrames = (int)Math.Ceiling(duration / interval);
            if (totalFrames > 720) totalFrames = 720;

            var cols = 10;
            var rows = (int)Math.Ceiling((double)totalFrames / cols);
            var successfulFrames = 0;

            for (int i = 0; i < totalFrames; i++)
            {
                var timestamp = i * interval;
                var framePath = Path.Combine(tempDir, $"frame_{i:D4}.jpg");
                if (await ExtractFrameAsync(filePath, framePath, timestamp, 160))
                    successfulFrames++;
            }

            if (successfulFrames == 0) return;

            var outputPath = Path.Combine(spriteDir, $"{mediaId}_sprite.jpg");

            // Stitch using ffmpeg tile filter
            using var process = new Process
            {
                StartInfo = new ProcessStartInfo
                {
                    FileName = "ffmpeg",
                    Arguments = $"-y -framerate 1/{interval} -start_number 0 " +
                                $"-i \"{Path.Combine(tempDir, "frame_%04d.jpg")}\" " +
                                $"-vf \"tile={cols}x{rows}\" -q:v 5 \"{outputPath}\"",
                    RedirectStandardError = true,
                    UseShellExecute = false,
                    CreateNoWindow = true,
                }
            };
            process.Start();
            await process.WaitForExitAsync();

            var spriteMeta = new Dictionary<string, object>
            {
                ["frames"] = successfulFrames,
                ["cols"] = cols,
                ["rows"] = rows,
                ["fw"] = 160,
                ["fh"] = 90,
                ["interval"] = interval,
            };
            var metaJson = JsonSerializer.Serialize(spriteMeta);

            using var scope = _scopeFactory.CreateScope();
            var db = scope.ServiceProvider.GetRequiredService<AppDbContext>();
            var media = await db.Media.FindAsync(mediaId);
            if (media != null)
            {
                media.SpritePath = Path.Combine("sprites", $"{mediaId}_sprite.jpg").Replace('\\', '/');
                media.SpriteMeta = metaJson;
                await db.SaveChangesAsync();
            }
        }
        finally
        {
            try { Directory.Delete(tempDir, true); } catch { }
        }
    }

    private static async Task<bool> ExtractFrameAsync(string input, string output, double timestamp, int width)
    {
        try
        {
            using var process = new Process
            {
                StartInfo = new ProcessStartInfo
                {
                    FileName = "ffmpeg",
                    Arguments = $"-y -ss {timestamp:F2} -i \"{input}\" -vframes 1 -vf scale={width}:-1 -q:v 3 \"{output}\"",
                    RedirectStandardError = true,
                    UseShellExecute = false,
                    CreateNoWindow = true,
                }
            };
            process.Start();
            await process.WaitForExitAsync();
            return File.Exists(output);
        }
        catch
        {
            return false;
        }
    }
}
