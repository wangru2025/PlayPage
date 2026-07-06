package domain

import "time"

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	Role      string    `json:"role"`
	PlanCode  string    `json:"planCode"`
	CreatedAt time.Time `json:"createdAt"`
}

type Project struct {
	ID                      string    `json:"id"`
	Username                string    `json:"username"`
	Slug                    string    `json:"slug"`
	Name                    string    `json:"name"`
	Interactive             bool      `json:"interactive"`
	AnalyticsEnabled        bool      `json:"analyticsEnabled"`
	ShowOnProfile           bool      `json:"showOnProfile"`
	AllowForks              bool      `json:"allowForks"`
	Visibility              string    `json:"visibility"`
	CurrentRelease          string    `json:"currentReleaseId"`
	PublicURL               string    `json:"publicUrl"`
	ForkedFromProjectID     string    `json:"forkedFromProjectId,omitempty"`
	ForkedFromReleaseID     string    `json:"forkedFromReleaseId,omitempty"`
	ForkedFromUserID        string    `json:"forkedFromUserId,omitempty"`
	ForkedFromUsername      string    `json:"forkedFromUsername,omitempty"`
	ForkedFromProjectName   string    `json:"forkedFromProjectName,omitempty"`
	ForkedFromProjectURL    string    `json:"forkedFromProjectUrl,omitempty"`
	ForkedFromSnapshotName  string    `json:"forkedFromSnapshotName,omitempty"`
	ForkedFromSnapshotOwner string    `json:"forkedFromSnapshotOwner,omitempty"`
	FavoritesCount          int       `json:"favoritesCount"`
	ForksCount              int       `json:"forksCount"`
	FavoritedByMe           bool      `json:"favoritedByMe,omitempty"`
	CanSubmitProposal       bool      `json:"canSubmitProposal,omitempty"`
	CreatedAt               time.Time `json:"createdAt"`
}

type PublicProjectAccess struct {
	Project     Project
	PublicKey   string
	OwnerUserID string
	OwnerRole   string
	OwnerPlan   string
}

type Collection struct {
	ID          string        `json:"id"`
	ProjectID   string        `json:"projectId"`
	Name        string        `json:"name"`
	Permissions PermissionSet `json:"permissions"`
	Fields      []FieldSchema `json:"fields"`
	CreatedAt   time.Time     `json:"createdAt"`
}

type PermissionSet struct {
	PublicRead  bool `json:"publicRead"`
	PublicWrite bool `json:"publicWrite"`
	// Deprecated compatibility fields. Public API no longer distinguishes logged-in/creator writes.
	AuthWrite   bool `json:"authWrite,omitempty"`
	OwnerWrite  bool `json:"ownerWrite,omitempty"`
	OwnerDelete bool `json:"ownerDelete,omitempty"`
}

type FieldSchema struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Required  bool   `json:"required"`
	IsList    bool   `json:"isList"`
	Reference string `json:"reference,omitempty"`
}

type Record struct {
	ID           string         `json:"id"`
	ProjectID    string         `json:"projectId"`
	CollectionID string         `json:"collectionId"`
	Data         map[string]any `json:"data"`
	Status       string         `json:"status"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

type ProjectCreateInput struct {
	Name             string `json:"name"`
	Username         string `json:"username"`
	Slug             string `json:"slug"`
	Interactive      bool   `json:"interactive"`
	AnalyticsEnabled bool   `json:"analyticsEnabled"`
	ShowOnProfile    bool   `json:"showOnProfile"`
	AllowForks       bool   `json:"allowForks"`
}

type ProjectForkInput struct {
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	Interactive      bool   `json:"interactive"`
	AnalyticsEnabled bool   `json:"analyticsEnabled"`
	ShowOnProfile    bool   `json:"showOnProfile"`
	AllowForks       bool   `json:"allowForks"`
}

type AuthorProfile struct {
	Username       string    `json:"username"`
	DisplayName    string    `json:"displayName"`
	JoinedAt       time.Time `json:"joinedAt"`
	Projects       []Project `json:"projects"`
	ProjectCount   int       `json:"projectCount"`
	FavoritesCount int       `json:"favoritesCount"`
	ForksCount     int       `json:"forksCount"`
	FollowersCount int       `json:"followersCount"`
	FollowingCount int       `json:"followingCount"`
	FollowingByMe  bool      `json:"followingByMe,omitempty"`
}

type AuthorSummary struct {
	UserID         string    `json:"userId"`
	Username       string    `json:"username"`
	DisplayName    string    `json:"displayName"`
	JoinedAt       time.Time `json:"joinedAt"`
	ProjectCount   int       `json:"projectCount"`
	FollowersCount int       `json:"followersCount"`
}

type NotificationRecipient struct {
	Email    string `json:"email"`
	Username string `json:"username"`
}

type ProjectDiscussion struct {
	ID               string    `json:"id"`
	ProjectID        string    `json:"projectId"`
	ProjectName      string    `json:"projectName,omitempty"`
	ProjectURL       string    `json:"projectUrl,omitempty"`
	AuthorUserID     string    `json:"authorUserId"`
	AuthorUsername   string    `json:"authorUsername"`
	AuthorEmail      string    `json:"authorEmail,omitempty"`
	Title            string    `json:"title"`
	Body             string    `json:"body"`
	Status           string    `json:"status"`
	CommentsCount    int       `json:"commentsCount"`
	LastCommentedAt  time.Time `json:"lastCommentedAt"`
	ClosedByUserID   string    `json:"closedByUserId,omitempty"`
	ClosedByUsername string    `json:"closedByUsername,omitempty"`
	ClosedAt         time.Time `json:"closedAt,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type ProjectDiscussionComment struct {
	ID             string    `json:"id"`
	DiscussionID   string    `json:"discussionId"`
	AuthorUserID   string    `json:"authorUserId"`
	AuthorUsername string    `json:"authorUsername"`
	AuthorEmail    string    `json:"authorEmail,omitempty"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type ProjectDiscussionCreateInput struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type ProjectDiscussionCommentCreateInput struct {
	Body string `json:"body"`
}

type ProjectProposal struct {
	ID                  string    `json:"id"`
	SourceProjectID     string    `json:"sourceProjectId"`
	SourceProjectName   string    `json:"sourceProjectName"`
	SourceProjectURL    string    `json:"sourceProjectUrl"`
	TargetProjectID     string    `json:"targetProjectId"`
	TargetProjectName   string    `json:"targetProjectName"`
	TargetProjectURL    string    `json:"targetProjectUrl"`
	AuthorUserID        string    `json:"authorUserId"`
	AuthorUsername      string    `json:"authorUsername"`
	AuthorEmail         string    `json:"authorEmail,omitempty"`
	TargetOwnerUserID   string    `json:"targetOwnerUserId"`
	TargetOwnerUsername string    `json:"targetOwnerUsername"`
	TargetOwnerEmail    string    `json:"targetOwnerEmail,omitempty"`
	Title               string    `json:"title"`
	Body                string    `json:"body"`
	Status              string    `json:"status"`
	SourceReleaseID     string    `json:"sourceReleaseId"`
	MergedReleaseID     string    `json:"mergedReleaseId,omitempty"`
	ReviewNote          string    `json:"reviewNote"`
	ReviewedBy          string    `json:"reviewedBy,omitempty"`
	ReviewedAt          time.Time `json:"reviewedAt,omitempty"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type ProjectProposalCreateInput struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type ProjectProposalReviewInput struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

type ProjectVisibilityUpdateInput struct {
	Visibility string `json:"visibility"`
}

type ProjectPathUpdateInput struct {
	Slug string `json:"slug"`
}

type ProjectSettingsUpdateInput struct {
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	Interactive      bool   `json:"interactive"`
	AnalyticsEnabled bool   `json:"analyticsEnabled"`
	ShowOnProfile    bool   `json:"showOnProfile"`
	AllowForks       bool   `json:"allowForks"`
}

type AppBuildSettings struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"userId"`
	ProjectID          string    `json:"projectId"`
	AppName            string    `json:"appName"`
	AndroidEnabled     bool      `json:"androidEnabled"`
	WindowsEnabled     bool      `json:"windowsEnabled"`
	AutoUpdate         bool      `json:"autoUpdate"`
	AndroidPackageName string    `json:"androidPackageName"`
	WindowsPackageName string    `json:"windowsPackageName"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type AppBuildSettingsInput struct {
	AppName            string `json:"appName"`
	AndroidEnabled     bool   `json:"androidEnabled"`
	WindowsEnabled     bool   `json:"windowsEnabled"`
	AutoUpdate         bool   `json:"autoUpdate"`
	AndroidPackageName string `json:"androidPackageName"`
	WindowsPackageName string `json:"windowsPackageName"`
}

type AppBuildJob struct {
	ID             string    `json:"id"`
	UserID         string    `json:"userId"`
	ProjectID      string    `json:"projectId"`
	ReleaseID      string    `json:"releaseId"`
	Platform       string    `json:"platform"`
	AppName        string    `json:"appName"`
	PackageName    string    `json:"packageName"`
	VersionCode    int       `json:"versionCode"`
	VersionName    string    `json:"versionName"`
	AutoUpdate     bool      `json:"autoUpdate"`
	Status         string    `json:"status"`
	ArtifactPath   string    `json:"artifactPath"`
	ArtifactSHA256 string    `json:"artifactSha256"`
	ArtifactSize   int64     `json:"artifactSize"`
	GitHubRunID    string    `json:"githubRunId"`
	ErrorMessage   string    `json:"errorMessage"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type AppBuildCreateInput struct {
	Platform   string `json:"platform"`
	AutoUpdate bool   `json:"autoUpdate"`
}

type AppBuildCompleteInput struct {
	Status         string `json:"status"`
	ArtifactPath   string `json:"artifactPath"`
	ArtifactSHA256 string `json:"artifactSha256"`
	ArtifactSize   int64  `json:"artifactSize"`
	GitHubRunID    string `json:"githubRunId"`
	ErrorMessage   string `json:"errorMessage"`
}

type AppUpdateInfo struct {
	HasUpdate         bool   `json:"hasUpdate"`
	LatestVersionCode int    `json:"latestVersionCode"`
	LatestVersionName string `json:"latestVersionName"`
	ReleaseNote       string `json:"releaseNote"`
	DownloadURL       string `json:"downloadUrl"`
	SHA256            string `json:"sha256"`
	Size              int64  `json:"size"`
	Force             bool   `json:"force"`
}

type ContestSubmission struct {
	ID            string    `json:"id"`
	UserID        string    `json:"userId"`
	UserEmail     string    `json:"userEmail,omitempty"`
	Username      string    `json:"username"`
	ProjectID     string    `json:"projectId"`
	ProjectName   string    `json:"projectName"`
	ProjectURL    string    `json:"projectUrl"`
	Track         string    `json:"track"`
	Intro         string    `json:"intro"`
	Story         string    `json:"story"`
	AllowShowcase bool      `json:"allowShowcase"`
	Status        string    `json:"status"`
	AdminNote     string    `json:"adminNote"`
	ReviewedBy    string    `json:"reviewedBy"`
	ReviewedAt    time.Time `json:"reviewedAt"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type ContestSubmissionCreateInput struct {
	Track         string `json:"track"`
	Intro         string `json:"intro"`
	Story         string `json:"story"`
	AllowShowcase bool   `json:"allowShowcase"`
}

type ContestSubmissionReviewInput struct {
	Status    string `json:"status"`
	AdminNote string `json:"adminNote"`
}

type CollectionCreateInput struct {
	Name        string        `json:"name"`
	Permissions PermissionSet `json:"permissions"`
	Fields      []FieldSchema `json:"fields"`
}

type CollectionUpdateInput struct {
	Permissions PermissionSet `json:"permissions"`
	Fields      []FieldSchema `json:"fields"`
}

type RecordCreateInput struct {
	Data map[string]any `json:"data"`
}

type RecordUpdateInput struct {
	Data map[string]any `json:"data"`
}

type AuthCodeRequestInput struct {
	Email    string `json:"email"`
	Username string `json:"username"`
}

type AuthCodeVerifyInput struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type UserProfileUpdateInput struct {
	Username string `json:"username"`
}

type ProjectEmailCodeSendInput struct {
	Email   string `json:"email"`
	Purpose string `json:"purpose"`
}

type ProjectEmailCodeVerifyInput struct {
	Email   string `json:"email"`
	Purpose string `json:"purpose"`
	Code    string `json:"code"`
}

type AdminUserSummary struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	Status       string    `json:"status"`
	Role         string    `json:"role"`
	PlanCode     string    `json:"planCode"`
	ProjectCount int       `json:"projectCount"`
	CreatedAt    time.Time `json:"createdAt"`
}

type AdminProjectSummary struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Slug             string    `json:"slug"`
	Username         string    `json:"username"`
	OwnerUserID      string    `json:"ownerUserId"`
	OwnerEmail       string    `json:"ownerEmail"`
	Interactive      bool      `json:"interactive"`
	AnalyticsEnabled bool      `json:"analyticsEnabled"`
	Visibility       string    `json:"visibility"`
	CurrentRelease   string    `json:"currentReleaseId"`
	PublicURL        string    `json:"publicUrl"`
	CreatedAt        time.Time `json:"createdAt"`
}

type ProjectUsage struct {
	ProjectID  string    `json:"projectId"`
	MonthKey   string    `json:"monthKey"`
	QueryCount int64     `json:"queryCount"`
	WriteCount int64     `json:"writeCount"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type ProjectDailyStats struct {
	Date         string `json:"date"`
	PageViews    int64  `json:"pageViews"`
	APIRequests  int64  `json:"apiRequests"`
	APISuccesses int64  `json:"apiSuccesses"`
	APIFailures  int64  `json:"apiFailures"`
}

type ProjectStatsSummary struct {
	ProjectID         string              `json:"projectId"`
	From              string              `json:"from"`
	To                string              `json:"to"`
	TotalPageViews    int64               `json:"totalPageViews"`
	TotalAPIRequests  int64               `json:"totalApiRequests"`
	TotalAPISuccesses int64               `json:"totalApiSuccesses"`
	TotalAPIFailures  int64               `json:"totalApiFailures"`
	APISuccessRate    float64             `json:"apiSuccessRate"`
	APIFailureRate    float64             `json:"apiFailureRate"`
	Items             []ProjectDailyStats `json:"items"`
}

type ProjectDataStats struct {
	ProjectID       string `json:"projectId"`
	RecordCount     int64  `json:"recordCount"`
	DataBytes       int64  `json:"dataBytes"`
	CollectionCount int64  `json:"collectionCount"`
}

type RepairRequest struct {
	ID               string    `json:"id"`
	ProjectID        string    `json:"projectId"`
	ProjectName      string    `json:"projectName,omitempty"`
	ProjectPublicURL string    `json:"projectPublicUrl,omitempty"`
	OwnerUserID      string    `json:"ownerUserId"`
	OwnerEmail       string    `json:"ownerEmail,omitempty"`
	Username         string    `json:"username,omitempty"`
	IssueType        string    `json:"issueType"`
	Description      string    `json:"description"`
	Expected         string    `json:"expected"`
	AllowAdminEdit   bool      `json:"allowAdminEdit"`
	Contact          string    `json:"contact"`
	Status           string    `json:"status"`
	AdminReply       string    `json:"adminReply"`
	UserReply        string    `json:"userReply"`
	UserRepliedAt    time.Time `json:"userRepliedAt"`
	ReviewedBy       string    `json:"reviewedBy"`
	ReviewedAt       time.Time `json:"reviewedAt"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type RepairRequestCreateInput struct {
	IssueType      string `json:"issueType"`
	Description    string `json:"description"`
	Expected       string `json:"expected"`
	AllowAdminEdit bool   `json:"allowAdminEdit"`
	Contact        string `json:"contact"`
}

type RepairRequestUpdateInput struct {
	Status     string `json:"status"`
	AdminReply string `json:"adminReply"`
}

type RepairRequestUserReplyInput struct {
	Reply string `json:"reply"`
}

type RepairAIJob struct {
	ID              string    `json:"id"`
	RepairRequestID string    `json:"repairRequestId"`
	ProjectID       string    `json:"projectId"`
	OwnerUserID     string    `json:"ownerUserId"`
	Status          string    `json:"status"`
	Round           int       `json:"round"`
	Feedback        string    `json:"feedback"`
	GeneratedHTML   string    `json:"generatedHtml,omitempty"`
	PreviewURL      string    `json:"previewUrl"`
	ErrorMessage    string    `json:"errorMessage"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	FinishedAt      time.Time `json:"finishedAt"`
}

type RepairAIMessage struct {
	ID              string    `json:"id"`
	JobID           string    `json:"jobId"`
	RepairRequestID string    `json:"repairRequestId"`
	ProjectID       string    `json:"projectId"`
	OwnerUserID     string    `json:"ownerUserId"`
	AgentKey        string    `json:"agentKey"`
	AgentName       string    `json:"agentName"`
	Role            string    `json:"role"`
	Visibility      string    `json:"visibility"`
	MessageType     string    `json:"messageType"`
	Content         string    `json:"content"`
	MetadataJSON    string    `json:"metadataJson,omitempty"`
	MessageSeq      int       `json:"messageSeq"`
	CreatedAt       time.Time `json:"createdAt"`
}

type RepairAIFeedbackInput struct {
	Feedback string `json:"feedback"`
}

type ProjectDomain struct {
	ID               string    `json:"id"`
	ProjectID        string    `json:"projectId"`
	ProjectName      string    `json:"projectName,omitempty"`
	ProjectPublicURL string    `json:"projectPublicUrl,omitempty"`
	OwnerUserID      string    `json:"ownerUserId"`
	OwnerEmail       string    `json:"ownerEmail,omitempty"`
	Username         string    `json:"username,omitempty"`
	Subdomain        string    `json:"subdomain"`
	Domain           string    `json:"domain"`
	Status           string    `json:"status"`
	RejectReason     string    `json:"rejectReason"`
	AdminNote        string    `json:"adminNote"`
	ReviewedBy       string    `json:"reviewedBy"`
	ReviewedAt       time.Time `json:"reviewedAt"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type ProjectDomainCreateInput struct {
	Subdomain string `json:"subdomain"`
}

type ProjectDomainReviewInput struct {
	Status       string `json:"status"`
	RejectReason string `json:"rejectReason"`
	AdminNote    string `json:"adminNote"`
}

type ProjectDomainDeleteRequest struct {
	ID               string    `json:"id"`
	DomainID         string    `json:"domainId"`
	ProjectID        string    `json:"projectId"`
	ProjectName      string    `json:"projectName,omitempty"`
	ProjectPublicURL string    `json:"projectPublicUrl,omitempty"`
	OwnerUserID      string    `json:"ownerUserId"`
	OwnerEmail       string    `json:"ownerEmail,omitempty"`
	Username         string    `json:"username,omitempty"`
	Domain           string    `json:"domain"`
	Reason           string    `json:"reason"`
	Status           string    `json:"status"`
	AdminNote        string    `json:"adminNote"`
	ReviewedBy       string    `json:"reviewedBy"`
	ReviewedAt       time.Time `json:"reviewedAt"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type ProjectDomainDeleteRequestCreateInput struct {
	DomainID string `json:"domainId"`
	Reason   string `json:"reason"`
}

type ProjectDomainDeleteRequestReviewInput struct {
	Status    string `json:"status"`
	AdminNote string `json:"adminNote"`
}

type UpgradeRequest struct {
	ID            string    `json:"id"`
	UserID        string    `json:"userId"`
	UserEmail     string    `json:"userEmail"`
	Username      string    `json:"username"`
	CurrentPlan   string    `json:"currentPlan"`
	TargetPlan    string    `json:"targetPlan"`
	PaymentMethod string    `json:"paymentMethod"`
	PayerNote     string    `json:"payerNote"`
	SystemNote    string    `json:"systemNote"`
	Status        string    `json:"status"`
	AdminNote     string    `json:"adminNote"`
	ReviewedBy    string    `json:"reviewedBy"`
	ReviewedAt    time.Time `json:"reviewedAt"`
	CreatedAt     time.Time `json:"createdAt"`
}

type UpgradeRequestCreateInput struct {
	TargetPlan    string `json:"targetPlan"`
	PaymentMethod string `json:"paymentMethod"`
	PayerNote     string `json:"payerNote"`
}

type UpgradeRequestReviewInput struct {
	Status     string `json:"status"`
	TargetPlan string `json:"targetPlan"`
	AdminNote  string `json:"adminNote"`
}

type AdminUserUpdateInput struct {
	Role     string `json:"role"`
	PlanCode string `json:"planCode"`
}
