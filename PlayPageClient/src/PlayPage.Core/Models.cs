namespace PlayPage.Core;

public sealed class LoginRequest
{
    public string Email { get; set; } = "";
    public string Password { get; set; } = "";
}

public sealed class AuthResult
{
    public string Token { get; set; } = "";
    public UserProfile User { get; set; } = new UserProfile();
}

public sealed class UserProfile
{
    public string Id { get; set; } = "";
    public string Email { get; set; } = "";
    public string DisplayName { get; set; } = "";
    public string Role { get; set; } = "";
}

public sealed class ProjectSummary
{
    public string Id { get; set; } = "";
    public string Name { get; set; } = "";
    public string Slug { get; set; } = "";
    public string PublicUrl { get; set; } = "";
    public bool Interactive { get; set; }
    public bool AnalyticsEnabled { get; set; }
}
