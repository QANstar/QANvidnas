using Microsoft.AspNetCore.Identity;

namespace QANvidnasServer.Services;

public class PasswordService
{
    private readonly PasswordHasher<string> _hasher = new();

    public string HashPassword(string plainPassword)
    {
        return _hasher.HashPassword(string.Empty, plainPassword);
    }

    public bool VerifyPassword(string plainPassword, string passwordHash)
    {
        var result = _hasher.VerifyHashedPassword(string.Empty, passwordHash, plainPassword);
        return result == PasswordVerificationResult.Success;
    }
}
