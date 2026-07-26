using Microsoft.EntityFrameworkCore;
using QANvidnasServer.Data;
using QANvidnasServer.Services;

var builder = WebApplication.CreateBuilder(args);

// ─── 数据库 (SQLite) ───
var dataDir = builder.Configuration.GetValue<string>("Storage:DataDir") ?? "data";
Directory.CreateDirectory(dataDir);
var dbPath = Path.Combine(dataDir, "qanvidnas.db");
builder.Services.AddDbContext<AppDbContext>(options =>
    options.UseSqlite($"Data Source={dbPath}"));

// ─── JWT 认证 ───
var jwtKey = builder.Configuration.GetValue<string>("Jwt:Key")!;
var jwtIssuer = builder.Configuration.GetValue<string>("Jwt:Issuer")!;
var jwtAudience = builder.Configuration.GetValue<string>("Jwt:Audience")!;
builder.Services.AddAuthentication().AddJwtBearer(options =>
{
    options.TokenValidationParameters = new Microsoft.IdentityModel.Tokens.TokenValidationParameters
    {
        ValidateIssuer = true,
        ValidateAudience = true,
        ValidateLifetime = true,
        ValidateIssuerSigningKey = true,
        ValidIssuer = jwtIssuer,
        ValidAudience = jwtAudience,
        IssuerSigningKey = new Microsoft.IdentityModel.Tokens.SymmetricSecurityKey(
            System.Text.Encoding.UTF8.GetBytes(jwtKey))
    };
});
builder.Services.AddAuthorization();

// ─── CORS ───
builder.Services.AddCors(options =>
{
    options.AddDefaultPolicy(policy =>
    {
        policy.AllowAnyOrigin()
              .AllowAnyMethod()
              .AllowAnyHeader()
              .WithExposedHeaders("Content-Range", "Accept-Ranges", "Content-Length");
    });
});

// ─── 服务注册 ───
builder.Services.AddSingleton<PasswordService>();
builder.Services.AddSingleton<JwtService>();
builder.Services.AddSingleton<ScannerService>();
builder.Services.AddSingleton<ThumbnailService>();

// ─── 文件上传配置 ───
builder.Services.Configure<Microsoft.AspNetCore.Http.Features.FormOptions>(options =>
{
    options.MultipartBodyLengthLimit = 10L * 1024 * 1024 * 1024; // 10GB
});
builder.WebHost.ConfigureKestrel(options =>
{
    options.Limits.MaxRequestBodySize = 10L * 1024 * 1024 * 1024;
    options.Limits.RequestHeadersTimeout = TimeSpan.FromMinutes(10);
    options.Limits.KeepAliveTimeout = TimeSpan.FromMinutes(10);
});

var app = builder.Build();

// ─── 中间件管道 ───
app.UseCors();

// 自动创建数据库
using (var scope = app.Services.CreateScope())
{
    var db = scope.ServiceProvider.GetRequiredService<AppDbContext>();
    db.Database.EnsureCreated();
}

// ─── 健康检查 ───
app.MapGet("/api/health", () => Results.Ok(new { status = "ok", serverTime = DateTime.UtcNow.ToString("o") }));

// ─── 路由映射 ───
QANvidnasServer.Endpoints.AuthEndpoints.Map(app);
QANvidnasServer.Endpoints.MediaEndpoints.Map(app);
QANvidnasServer.Endpoints.StreamEndpoints.Map(app);
QANvidnasServer.Endpoints.SearchEndpoints.Map(app);
QANvidnasServer.Endpoints.PlaylistEndpoints.Map(app);
QANvidnasServer.Endpoints.UploadEndpoints.Map(app);

app.Run();
