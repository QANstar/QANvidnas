using System.Security.Claims;
using Microsoft.EntityFrameworkCore;
using QANvidnasServer.Data;
using QANvidnasServer.DTOs;
using QANvidnasServer.Models;
using QANvidnasServer.Services;

namespace QANvidnasServer.Endpoints;

public static class AuthEndpoints
{
    public static void Map(WebApplication app)
    {
        var group = app.MapGroup("/api/auth");
        var adminGroup = app.MapGroup("/api/admin").RequireAuthorization();

        // GET /api/auth/check-setup
        group.MapGet("/check-setup", async (AppDbContext db) =>
        {
            var hasAdmin = await db.Users.AnyAsync(u => u.IsAdmin);
            return Results.Ok(new SetupCheckResponse(!hasAdmin));
        });

        // POST /api/auth/setup
        group.MapPost("/setup", async (SetupRequest req, AppDbContext db, PasswordService pw, JwtService jwt) =>
        {
            if (await db.Users.AnyAsync(u => u.IsAdmin))
                return Results.Json(new { error = "管理员已存在，无法重复初始化" }, statusCode: 403);

            var user = new User
            {
                Username = req.Username,
                PasswordHash = pw.HashPassword(req.Password),
                IsAdmin = true,
            };
            db.Users.Add(user);
            await db.SaveChangesAsync();

            return Results.Ok(new AuthResponse(jwt.GenerateToken(user)));
        });

        // POST /api/auth/login
        group.MapPost("/login", async (LoginRequest req, AppDbContext db, PasswordService pw, JwtService jwt) =>
        {
            var user = await db.Users.FirstOrDefaultAsync(u => u.Username == req.Username);
            if (user == null || !pw.VerifyPassword(req.Password, user.PasswordHash))
                return Results.Json(new { error = "用户名或密码错误" }, statusCode: 401);

            return Results.Ok(new AuthResponse(jwt.GenerateToken(user)));
        });

        // POST /api/auth/register
        group.MapPost("/register", async (RegisterRequest req, AppDbContext db, PasswordService pw, JwtService jwt) =>
        {
            var invite = await db.InviteCodes.FirstOrDefaultAsync(i => i.Code == req.InviteCode);
            if (invite == null)
                return Results.Json(new { error = "无效的注册码" }, statusCode: 400);
            if (invite.MaxUses > 0 && invite.Used >= invite.MaxUses)
                return Results.Json(new { error = "注册码已用完，请联系管理员" }, statusCode: 400);
            if (!string.IsNullOrEmpty(invite.ExpiresAt) &&
                DateTime.TryParse(invite.ExpiresAt, out var expiry) &&
                DateTime.UtcNow > expiry)
                return Results.Json(new { error = "注册码已过期" }, statusCode: 400);

            if (await db.Users.AnyAsync(u => u.Username == req.Username))
                return Results.Json(new { error = "用户名已存在" }, statusCode: 409);

            var user = new User
            {
                Username = req.Username,
                PasswordHash = pw.HashPassword(req.Password),
                IsAdmin = false,
            };

            invite.Used++;

            db.Users.Add(user);
            await db.SaveChangesAsync();

            return Results.Ok(new AuthResponse(jwt.GenerateToken(user)));
        });

        // GET /api/auth/me
        group.MapGet("/me", async (HttpContext http, AppDbContext db) =>
        {
            var userIdClaim = http.User.FindFirst("user_id")?.Value;
            if (userIdClaim == null || !long.TryParse(userIdClaim, out var userId))
                return Results.Json(new { error = "未提供认证令牌" }, statusCode: 401);

            var user = await db.Users.FindAsync(userId);
            if (user == null)
                return Results.Json(new { error = "用户不存在" }, statusCode: 404);

            return Results.Ok(new UserInfo(user.Id, user.Username, user.IsAdmin));
        }).RequireAuthorization();

        // GET /api/auth/device-code
        group.MapGet("/device-code", async (AppDbContext db) =>
        {
            const string charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789";
            var code = new string(Enumerable.Range(0, 4).Select(_ => charset[Random.Shared.Next(charset.Length)]).ToArray());
            var deviceCode = new DeviceCode
            {
                Code = code,
                ExpiresAt = DateTime.UtcNow.AddMinutes(10),
            };
            db.DeviceCodes.Add(deviceCode);
            await db.SaveChangesAsync();
            return Results.Ok(new DeviceCodeResponse(code, 600));
        });

        // POST /api/auth/device-code/authorize
        group.MapPost("/device-code/authorize", async (AuthorizeDeviceCodeRequest req, HttpContext http, AppDbContext db) =>
        {
            var userIdClaim = http.User.FindFirst("user_id")?.Value;
            if (userIdClaim == null || !long.TryParse(userIdClaim, out var userId))
                return Results.Json(new { error = "未提供认证令牌" }, statusCode: 401);

            var dc = await db.DeviceCodes.FirstOrDefaultAsync(d =>
                d.Code == req.Code && !d.Used && d.ExpiresAt > DateTime.UtcNow);
            if (dc == null)
                return Results.Json(new { error = "设备码无效或已过期" }, statusCode: 400);

            dc.UserId = userId;
            dc.Used = true;
            await db.SaveChangesAsync();

            return Results.Ok(new { message = "授权成功" });
        }).RequireAuthorization();

        // GET /api/auth/device-code/poll
        group.MapGet("/device-code/poll", async (string code, AppDbContext db, JwtService jwt) =>
        {
            var dc = await db.DeviceCodes.FirstOrDefaultAsync(d =>
                d.Code == code && d.ExpiresAt > DateTime.UtcNow);

            if (dc == null)
                return Results.Json(new { error = "设备码已过期" }, statusCode: 400);

            if (!dc.Used || dc.UserId == null)
                return Results.Ok(new DeviceCodePollResponse("waiting"));

            var user = await db.Users.FindAsync(dc.UserId.Value);
            if (user == null)
                return Results.Json(new { error = "用户不存在" }, statusCode: 400);

            return Results.Ok(new DeviceCodePollResponse("authorized", jwt.GenerateToken(user)));
        });

        // POST /api/auth/change-password
        group.MapPost("/change-password", async (ChangePasswordRequest req, HttpContext http, AppDbContext db, PasswordService pw) =>
        {
            var userIdClaim = http.User.FindFirst("user_id")?.Value;
            if (userIdClaim == null || !long.TryParse(userIdClaim, out var userId))
                return Results.Json(new { error = "未提供认证令牌" }, statusCode: 401);

            var user = await db.Users.FindAsync(userId);
            if (user == null)
                return Results.Json(new { error = "用户不存在" }, statusCode: 404);

            if (!pw.VerifyPassword(req.OldPassword, user.PasswordHash))
                return Results.Json(new { error = "原密码错误" }, statusCode: 400);

            user.PasswordHash = pw.HashPassword(req.NewPassword);
            await db.SaveChangesAsync();

            return Results.Ok(new { message = "密码修改成功" });
        }).RequireAuthorization();

        // ─── Admin: Invite Codes ───
        adminGroup.MapGet("/invite-codes", async (AppDbContext db, HttpContext http) =>
        {
            if (!IsAdmin(http)) return Results.Json(new { error = "需要管理员权限" }, statusCode: 403);

            var codes = await db.InviteCodes.OrderBy(c => c.Code).Select(c => new InviteCodeInfo(
                c.Code, c.Description, c.MaxUses, c.Used, c.ExpiresAt
            )).ToListAsync();

            return Results.Ok(new { items = codes });
        });

        adminGroup.MapPost("/invite-codes", async (CreateInviteCodeRequest req, AppDbContext db, HttpContext http) =>
        {
            if (!IsAdmin(http)) return Results.Json(new { error = "需要管理员权限" }, statusCode: 403);

            var invite = await db.InviteCodes.FindAsync(req.Code);
            if (invite == null)
            {
                invite = new InviteCode { Code = req.Code };
                db.InviteCodes.Add(invite);
            }
            invite.Description = req.Description ?? "";
            invite.MaxUses = req.MaxUses ?? 1;
            invite.ExpiresAt = req.ExpiresAt ?? "";
            await db.SaveChangesAsync();

            return Results.Ok(new { message = "注册码创建成功" });
        });

        adminGroup.MapDelete("/invite-codes/{code}", async (string code, AppDbContext db, HttpContext http) =>
        {
            if (!IsAdmin(http)) return Results.Json(new { error = "需要管理员权限" }, statusCode: 403);

            var invite = await db.InviteCodes.FindAsync(code);
            if (invite != null)
            {
                db.InviteCodes.Remove(invite);
                await db.SaveChangesAsync();
            }
            return Results.Ok(new { message = "已删除" });
        });

        // ─── Admin: Config ───
        adminGroup.MapGet("/config", (IConfiguration config, HttpContext http) =>
        {
            if (!IsAdmin(http)) return Results.Json(new { error = "需要管理员权限" }, statusCode: 403);
            return Results.Ok(new
            {
                server = new { host = "0.0.0.0", port = "6666" },
                data_dir = config.GetValue<string>("Storage:DataDir") ?? "data"
            });
        });
    }

    private static bool IsAdmin(HttpContext http)
    {
        var claim = http.User.FindFirst("is_admin")?.Value;
        return claim == "true";
    }
}
