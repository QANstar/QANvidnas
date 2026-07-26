using System.IdentityModel.Tokens.Jwt;
using System.Security.Claims;
using System.Text;
using Microsoft.IdentityModel.Tokens;
using QANvidnasServer.Models;

namespace QANvidnasServer.Services;

public class JwtService
{
    private readonly IConfiguration _config;

    public JwtService(IConfiguration config)
    {
        _config = config;
    }

    public string GenerateToken(User user)
    {
        var key = _config.GetValue<string>("Jwt:Key")!;
        var issuer = _config.GetValue<string>("Jwt:Issuer")!;
        var audience = _config.GetValue<string>("Jwt:Audience")!;
        var expireHours = _config.GetValue<int>("Jwt:ExpireHours", 168);

        var securityKey = new SymmetricSecurityKey(Encoding.UTF8.GetBytes(key));
        var credentials = new SigningCredentials(securityKey, SecurityAlgorithms.HmacSha256);

        var claims = new[]
        {
            new Claim("user_id", user.Id.ToString()),
            new Claim("is_admin", user.IsAdmin.ToString().ToLower()),
            new Claim(ClaimTypes.Name, user.Username),
        };

        var token = new JwtSecurityToken(
            issuer: issuer,
            audience: audience,
            claims: claims,
            expires: DateTime.UtcNow.AddHours(expireHours),
            signingCredentials: credentials
        );

        return new JwtSecurityTokenHandler().WriteToken(token);
    }
}
