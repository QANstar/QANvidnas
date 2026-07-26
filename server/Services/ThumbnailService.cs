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

        // Try multiple positions to avoid black frames (intros, scene transitions)
        var positions = duration switch
        {
            < 10   => new[] { duration * 0.5 },
            < 60   => new[] { duration * 0.35, duration * 0.55, duration * 0.75 },
            < 600  => new[] { duration * 0.3, duration * 0.5, duration * 0.7 },
            _      => new[] { duration * 0.25, duration * 0.5, duration * 0.75 },
        };

        foreach (var pos in positions)
        {
            if (await ExtractFrameAsync(filePath, outputPath, pos, 480))
            {
                // Check if the frame is likely a black/dark frame (very small file size)
                try
                {
                    var info = new FileInfo(outputPath);
                    if (info.Length < 4096) // < 4KB is suspiciously small, likely black/dark
                    {
                        File.Delete(outputPath);
                        continue; // try next position
                    }
                }
                catch { continue; }
                break; // got a good frame
            }
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
            // Fast keyframe seek to ~10s before target, then accurate seek from there
            var preSeek = Math.Max(0, timestamp - 10);
            var accurateSeek = timestamp - preSeek;

            using var process = new Process
            {
                StartInfo = new ProcessStartInfo
                {
                    FileName = "ffmpeg",
                    Arguments = $"-y -ss {preSeek:F2} -i \"{input}\" -ss {accurateSeek:F2} -vframes 1 -vf scale={width}:-1 -q:v 3 \"{output}\"",
                    RedirectStandardError = true,
                    UseShellExecute = false,
                    CreateNoWindow = true,
                }
            };
            process.Start();

            // Timeout after 30 seconds — long videos can hang on seek
            using var cts = new CancellationTokenSource(TimeSpan.FromSeconds(30));
            try
            {
                await process.WaitForExitAsync(cts.Token);
            }
            catch (OperationCanceledException)
            {
                try { process.Kill(entireProcessTree: true); } catch { }
                try { if (File.Exists(output)) File.Delete(output); } catch { }
                return false;
            }

            // Clean up 0-byte files from failed extraction
            if (process.ExitCode != 0 && File.Exists(output))
            {
                try
                {
                    var info = new FileInfo(output);
                    if (info.Length == 0)
                        File.Delete(output);
                }
                catch { }
                return false;
            }

            return File.Exists(output);
        }
        catch
        {
            return false;
        }
    }
}
