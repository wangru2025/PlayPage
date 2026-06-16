package storage

import (
	"context"
	"time"

	"ai-static-host/api/internal/domain"
)

type Store interface {
	GetUserByEmail(ctx context.Context, email string) (domain.User, bool, error)
	GetUserBySessionToken(ctx context.Context, token string) (domain.User, bool, error)
	GetUserByID(ctx context.Context, userID string) (domain.User, bool, error)
	UpdateUserUsername(ctx context.Context, userID, username string) (domain.User, error)
	UpdateUserRoleAndPlan(ctx context.Context, userID, role, planCode string) (domain.User, error)
	SyncProjectUsernames(ctx context.Context, userID, username string) error
	CreateOrRefreshAuthCode(ctx context.Context, input domain.AuthCodeRequestInput, code string, expiresAt time.Time) error
	ConsumeAuthCode(ctx context.Context, input domain.AuthCodeVerifyInput, now time.Time) (domain.User, bool, error)
	CreateSession(ctx context.Context, userID, token string, expiresAt time.Time) error
	DeleteSession(ctx context.Context, token string) error
	ListAdminUsers(ctx context.Context) ([]domain.AdminUserSummary, error)
	ListProjects(ctx context.Context, userID string) ([]domain.Project, error)
	ListPublicProjects(ctx context.Context) ([]domain.Project, error)
	ListAdminProjects(ctx context.Context) ([]domain.AdminProjectSummary, error)
	CreateProject(ctx context.Context, userID string, input domain.ProjectCreateInput) (domain.Project, error)
	GetProject(ctx context.Context, userID, projectID string) (domain.Project, bool, error)
	DeleteProject(ctx context.Context, userID, projectID string) (bool, error)
	GetProjectPublicAccess(ctx context.Context, projectID string) (domain.PublicProjectAccess, bool, error)
	UpdateProjectVisibility(ctx context.Context, userID, projectID, visibility string) (domain.Project, bool, error)
	UpdateProjectPath(ctx context.Context, userID, projectID, slug string) (domain.Project, bool, error)
	ListCollections(ctx context.Context, projectID string) ([]domain.Collection, error)
	CreateCollection(ctx context.Context, projectID string, input domain.CollectionCreateInput) (domain.Collection, error)
	GetCollectionByName(ctx context.Context, projectID, collectionName string) (domain.Collection, bool, error)
	UpdateCollection(ctx context.Context, projectID, collectionName string, input domain.CollectionUpdateInput) (domain.Collection, bool, error)
	DeleteCollection(ctx context.Context, projectID, collectionName string) error
	ListRecords(ctx context.Context, projectID, collectionName string) ([]domain.Record, error)
	CreateRecord(ctx context.Context, projectID, collectionName, createdByUserID string, input domain.RecordCreateInput) (domain.Record, error)
	GetRecord(ctx context.Context, projectID, collectionName, recordID string) (domain.Record, bool, error)
	UpdateRecord(ctx context.Context, projectID, collectionName, recordID string, input domain.RecordUpdateInput) (domain.Record, bool, error)
	DeleteRecord(ctx context.Context, projectID, collectionName, recordID string) (bool, error)
	ProjectDataStats(ctx context.Context, projectID string) (domain.ProjectDataStats, error)
	GetMonthlyProjectUsage(ctx context.Context, projectID, monthKey string) (domain.ProjectUsage, error)
	IncrementProjectUsage(ctx context.Context, projectID, monthKey string, queryDelta, writeDelta int64) (domain.ProjectUsage, error)
	IncrementProjectDailyStats(ctx context.Context, projectID, day string, pageViews, apiRequests, apiSuccesses, apiFailures int64) error
	ProjectStats(ctx context.Context, projectID, from, to string) (domain.ProjectStatsSummary, error)
	CreateProjectDomain(ctx context.Context, input domain.ProjectDomain) (domain.ProjectDomain, error)
	ListProjectDomains(ctx context.Context, projectID string) ([]domain.ProjectDomain, error)
	ListAdminProjectDomains(ctx context.Context, status string) ([]domain.ProjectDomain, error)
	UpdateProjectDomainReview(ctx context.Context, domainID, status, rejectReason, adminNote, reviewedBy string) (domain.ProjectDomain, error)
	CreateRepairRequest(ctx context.Context, input domain.RepairRequest) (domain.RepairRequest, error)
	ListProjectRepairRequests(ctx context.Context, projectID string) ([]domain.RepairRequest, error)
	ListAdminRepairRequests(ctx context.Context, status string) ([]domain.RepairRequest, error)
	UpdateRepairRequest(ctx context.Context, requestID, status, adminReply, reviewedBy string) (domain.RepairRequest, error)
	ReplyRepairRequest(ctx context.Context, projectID, requestID, reply string) (domain.RepairRequest, error)
	GetRepairRequest(ctx context.Context, projectID, requestID string) (domain.RepairRequest, bool, error)
	CreateRepairAIJob(ctx context.Context, input domain.RepairAIJob) (domain.RepairAIJob, error)
	GetRepairAIJob(ctx context.Context, jobID string) (domain.RepairAIJob, bool, error)
	GetLatestRepairAIJob(ctx context.Context, repairRequestID string) (domain.RepairAIJob, bool, error)
	GetActiveRepairAIJobByUser(ctx context.Context, userID string) (domain.RepairAIJob, bool, error)
	CountRepairAIJobsForUserSince(ctx context.Context, userID string, since time.Time) (int, error)
	UpdateRepairAIJob(ctx context.Context, job domain.RepairAIJob) (domain.RepairAIJob, error)
	CreateRepairAIMessage(ctx context.Context, input domain.RepairAIMessage) (domain.RepairAIMessage, error)
	ListRepairAIMessages(ctx context.Context, jobID string) ([]domain.RepairAIMessage, error)
	IsReservedSubdomain(ctx context.Context, subdomain string) (bool, error)
	CreateUpgradeRequest(ctx context.Context, input domain.UpgradeRequest) (domain.UpgradeRequest, error)
	ListUpgradeRequests(ctx context.Context) ([]domain.UpgradeRequest, error)
	ListUserUpgradeRequests(ctx context.Context, userID string) ([]domain.UpgradeRequest, error)
	UpdateUpgradeRequest(ctx context.Context, requestID, status, adminNote, reviewedBy string) (domain.UpgradeRequest, error)
	CreateRelease(ctx context.Context, release domain.Release) (domain.Release, error)
	ListReleases(ctx context.Context, projectID string) ([]domain.Release, error)
	SetCurrentRelease(ctx context.Context, projectID, releaseID string) error
}
