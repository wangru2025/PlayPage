using System;
using System.Collections.Generic;
using System.Text.Json.Serialization;

namespace PlayPage.Core;

public sealed class AuthCodeRequest
{
    public string Email { get; set; } = "";
    public string Username { get; set; } = "";
}

public sealed class AuthCodeResponse
{
    public string Status { get; set; } = "";
    public string Email { get; set; } = "";
    public int ExpiresIn { get; set; }
    public string? DevCode { get; set; }
}

public sealed class AuthCodeVerifyRequest
{
    public string Email { get; set; } = "";
    public string Code { get; set; } = "";
}

public sealed class AuthResult
{
    public string Status { get; set; } = "";
    public UserProfile User { get; set; } = new UserProfile();
    public string CsrfToken { get; set; } = "";
    public string AccessToken { get; set; } = "";
    public string TokenType { get; set; } = "Bearer";
    public DateTimeOffset ExpiresAt { get; set; }
    public int ExpiresInSeconds { get; set; }
}

public sealed class UserProfile
{
    public string Id { get; set; } = "";
    public string Username { get; set; } = "";
    public string Email { get; set; } = "";
    public string Status { get; set; } = "";
    public string Role { get; set; } = "";
    public string PlanCode { get; set; } = "";
    public DateTimeOffset CreatedAt { get; set; }
}

public sealed class UserProfileUpdateRequest
{
    public string Username { get; set; } = "";
}

public sealed class ProjectSummary
{
    public string Id { get; set; } = "";
    public string Username { get; set; } = "";
    public string Slug { get; set; } = "";
    public string Name { get; set; } = "";
    public bool Interactive { get; set; }
    public bool AnalyticsEnabled { get; set; }
    public string Visibility { get; set; } = "";
    [JsonPropertyName("currentReleaseId")]
    public string CurrentReleaseId { get; set; } = "";
    public string PublicUrl { get; set; } = "";
    public DateTimeOffset CreatedAt { get; set; }
}

public sealed class ProjectDetails
{
    public ProjectSummary Project { get; set; } = new ProjectSummary();
    public string PublicKey { get; set; } = "";
    public Dictionary<string, string> ReleaseLayout { get; set; } = new Dictionary<string, string>();
}

public sealed class ProjectCreateRequest
{
    public string Name { get; set; } = "";
    public string Username { get; set; } = "";
    public string Slug { get; set; } = "";
    public bool Interactive { get; set; }
    public bool AnalyticsEnabled { get; set; }
}

public sealed class ProjectSettingsRequest
{
    public string Name { get; set; } = "";
    public string Slug { get; set; } = "";
    public bool Interactive { get; set; }
    public bool AnalyticsEnabled { get; set; }
}

public sealed class ProjectVisibilityRequest
{
    public string Visibility { get; set; } = "public";
}

public sealed class ReleaseInfo
{
    public string Id { get; set; } = "";
    public string ProjectId { get; set; } = "";
    public string Status { get; set; } = "";
    public string ArchivePath { get; set; } = "";
    public string PublicPath { get; set; } = "";
    public string EntryFile { get; set; } = "";
    public string ChangeNote { get; set; } = "";
    public List<string> Warnings { get; set; } = new List<string>();
    public DateTimeOffset CreatedAt { get; set; }
}

public sealed class PermissionSet
{
    public bool PublicRead { get; set; }
    public bool PublicWrite { get; set; }
}

public sealed class FieldSchema
{
    public string Name { get; set; } = "";
    public string Type { get; set; } = "string";
    public bool Required { get; set; }
    public bool IsList { get; set; }
    public string Reference { get; set; } = "";
}

public sealed class CollectionInfo
{
    public string Id { get; set; } = "";
    public string ProjectId { get; set; } = "";
    public string Name { get; set; } = "";
    public PermissionSet Permissions { get; set; } = new PermissionSet();
    public List<FieldSchema> Fields { get; set; } = new List<FieldSchema>();
    public DateTimeOffset CreatedAt { get; set; }
}

public sealed class CollectionCreateRequest
{
    public string Name { get; set; } = "";
    public PermissionSet Permissions { get; set; } = new PermissionSet();
    public List<FieldSchema> Fields { get; set; } = new List<FieldSchema>();
}

public sealed class CollectionUpdateRequest
{
    public PermissionSet Permissions { get; set; } = new PermissionSet();
    public List<FieldSchema> Fields { get; set; } = new List<FieldSchema>();
}

public sealed class RecordInfo
{
    public string Id { get; set; } = "";
    public string ProjectId { get; set; } = "";
    public string CollectionId { get; set; } = "";
    public Dictionary<string, object?> Data { get; set; } = new Dictionary<string, object?>();
    public string Status { get; set; } = "";
    public DateTimeOffset CreatedAt { get; set; }
    public DateTimeOffset UpdatedAt { get; set; }
}

public sealed class RecordWriteRequest
{
    public Dictionary<string, object?> Data { get; set; } = new Dictionary<string, object?>();
}

public sealed class ProjectDomainInfo
{
    public string Id { get; set; } = "";
    public string ProjectId { get; set; } = "";
    public string ProjectName { get; set; } = "";
    public string ProjectPublicUrl { get; set; } = "";
    public string OwnerUserId { get; set; } = "";
    public string OwnerEmail { get; set; } = "";
    public string Username { get; set; } = "";
    public string Subdomain { get; set; } = "";
    public string Domain { get; set; } = "";
    public string Status { get; set; } = "";
    public string RejectReason { get; set; } = "";
    public string AdminNote { get; set; } = "";
    public DateTimeOffset CreatedAt { get; set; }
    public DateTimeOffset UpdatedAt { get; set; }
}

public sealed class ProjectDomainCreateRequest { public string Subdomain { get; set; } = ""; }

public sealed class ProjectDomainDeleteRequestInfo
{
    public string Id { get; set; } = "";
    public string DomainId { get; set; } = "";
    public string ProjectId { get; set; } = "";
    public string Domain { get; set; } = "";
    public string Reason { get; set; } = "";
    public string Status { get; set; } = "";
    public string AdminNote { get; set; } = "";
    public DateTimeOffset CreatedAt { get; set; }
    public DateTimeOffset UpdatedAt { get; set; }
}

public sealed class ProjectDomainDeleteCreateRequest
{
    public string DomainId { get; set; } = "";
    public string Reason { get; set; } = "";
}

public sealed class RepairRequestInfo
{
    public string Id { get; set; } = "";
    public string ProjectId { get; set; } = "";
    public string ProjectName { get; set; } = "";
    public string ProjectPublicUrl { get; set; } = "";
    public string IssueType { get; set; } = "";
    public string Description { get; set; } = "";
    public string Expected { get; set; } = "";
    public bool AllowAdminEdit { get; set; }
    public string Contact { get; set; } = "";
    public string Status { get; set; } = "";
    public string AdminReply { get; set; } = "";
    public string UserReply { get; set; } = "";
    public DateTimeOffset CreatedAt { get; set; }
    public DateTimeOffset UpdatedAt { get; set; }
}

public sealed class RepairRequestCreateRequest
{
    public string IssueType { get; set; } = "other";
    public string Description { get; set; } = "";
    public string Expected { get; set; } = "";
    public bool AllowAdminEdit { get; set; }
    public string Contact { get; set; } = "";
}

public sealed class RepairReplyRequest { public string Reply { get; set; } = ""; }
public sealed class RepairAIFeedbackRequest { public string Feedback { get; set; } = ""; }

public sealed class RepairAIJobInfo
{
    public string Id { get; set; } = "";
    public string RepairRequestId { get; set; } = "";
    public string ProjectId { get; set; } = "";
    public string Status { get; set; } = "";
    public int Round { get; set; }
    public string Feedback { get; set; } = "";
    public string PreviewUrl { get; set; } = "";
    public string ErrorMessage { get; set; } = "";
    public DateTimeOffset CreatedAt { get; set; }
    public DateTimeOffset UpdatedAt { get; set; }
}

public sealed class RepairAIMessageInfo
{
    public string Id { get; set; } = "";
    public string JobId { get; set; } = "";
    public string AgentName { get; set; } = "";
    public string Role { get; set; } = "";
    public string MessageType { get; set; } = "";
    public string Content { get; set; } = "";
    public int MessageSeq { get; set; }
    public DateTimeOffset CreatedAt { get; set; }
}

public sealed class TemplateInfo
{
    public string Id { get; set; } = "";
    public string Slug { get; set; } = "";
    public string Name { get; set; } = "";
    public string Category { get; set; } = "";
    public string CategoryLabel { get; set; } = "";
    public string Description { get; set; } = "";
    public string Summary { get; set; } = "";
    public List<string> Tags { get; set; } = new List<string>();
    public string AuthorName { get; set; } = "";
    public bool InteractiveRequired { get; set; }
    public bool AnalyticsRecommended { get; set; }
    public int UsageCount { get; set; }
    public List<TemplateConfigField> ConfigFields { get; set; } = new List<TemplateConfigField>();
    public List<TemplateCollectionDefinition> Collections { get; set; } = new List<TemplateCollectionDefinition>();
}

public sealed class TemplateConfigField
{
    public string Name { get; set; } = "";
    public string Label { get; set; } = "";
    public string Type { get; set; } = "text";
    public bool Required { get; set; }
    public string Default { get; set; } = "";
    public string Placeholder { get; set; } = "";
    public string Help { get; set; } = "";
    public List<string> Options { get; set; } = new List<string>();
}

public sealed class TemplateCollectionDefinition
{
    public string Name { get; set; } = "";
    public PermissionSet Permissions { get; set; } = new PermissionSet();
    public List<FieldSchema> Fields { get; set; } = new List<FieldSchema>();
}

public sealed class TemplateCreateReleaseRequest
{
    public string TemplateId { get; set; } = "";
    public Dictionary<string, string> Params { get; set; } = new Dictionary<string, string>();
    public string ChangeNote { get; set; } = "";
}

public sealed class ProjectStatsSummary
{
    public string ProjectId { get; set; } = "";
    public string From { get; set; } = "";
    public string To { get; set; } = "";
    public long TotalPageViews { get; set; }
    public long TotalApiRequests { get; set; }
    public long TotalApiSuccesses { get; set; }
    public long TotalApiFailures { get; set; }
    public double ApiSuccessRate { get; set; }
    public double ApiFailureRate { get; set; }
    public List<ProjectDailyStats> Items { get; set; } = new List<ProjectDailyStats>();
}

public sealed class ProjectDailyStats
{
    public string Date { get; set; } = "";
    public long PageViews { get; set; }
    public long ApiRequests { get; set; }
    public long ApiSuccesses { get; set; }
    public long ApiFailures { get; set; }
}

public sealed class UpgradeRequestCreateRequest
{
    public string TargetPlan { get; set; } = "light";
    public string PaymentMethod { get; set; } = "wechat";
    public string PayerNote { get; set; } = "";
}

public sealed class UpgradeRequestInfo
{
    public string Id { get; set; } = "";
    public string UserEmail { get; set; } = "";
    public string Username { get; set; } = "";
    public string CurrentPlan { get; set; } = "";
    public string TargetPlan { get; set; } = "";
    public string PaymentMethod { get; set; } = "";
    public string PayerNote { get; set; } = "";
    public string Status { get; set; } = "";
    public string AdminNote { get; set; } = "";
    public DateTimeOffset CreatedAt { get; set; }
}

public sealed class ListEnvelope<T>
{
    public List<T> Items { get; set; } = new List<T>();
}

public sealed class StatusEnvelope
{
    public string Status { get; set; } = "";
}
