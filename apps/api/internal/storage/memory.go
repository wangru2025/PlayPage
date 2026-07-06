package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"ai-static-host/api/internal/domain"
)

type memoryAuthCode struct {
	Email     string
	Username  string
	Code      string
	ExpiresAt time.Time
	Used      bool
}

type memoryProjectEmailCode struct {
	ProjectID   string
	OwnerUserID string
	Email       string
	Purpose     string
	Code        string
	ExpiresAt   time.Time
	CreatedAt   time.Time
	Used        bool
}

type memorySession struct {
	UserID    string
	Token     string
	ExpiresAt time.Time
}

type MemoryStore struct {
	publicBase                  string
	mu                          sync.RWMutex
	users                       map[string]domain.User
	usersByMail                 map[string]string
	authCodes                   map[string]memoryAuthCode
	projectEmailCodes           []memoryProjectEmailCode
	projectEmailQuotaDaily      map[string]int
	sessions                    map[string]memorySession
	projects                    map[string]domain.Project
	projectUser                 map[string]string
	projectFavorites            map[string]map[string]bool
	authorFollows               map[string]map[string]bool
	projectNotificationEvents   map[string]bool
	projectDiscussions          map[string]domain.ProjectDiscussion
	projectDiscussionComments   map[string][]domain.ProjectDiscussionComment
	projectProposals            map[string]domain.ProjectProposal
	appBuildSettings            map[string]domain.AppBuildSettings
	appBuildJobs                map[string]domain.AppBuildJob
	appUpdateChannels           map[string]domain.AppUpdateInfo
	collections                 map[string][]domain.Collection
	records                     map[string][]domain.Record
	releases                    map[string][]domain.Release
	publicKeys                  map[string]string
	usage                       map[string]domain.ProjectUsage
	dailyStats                  map[string]domain.ProjectDailyStats
	upgradeRequests             map[string]domain.UpgradeRequest
	projectDomains              map[string]domain.ProjectDomain
	projectDomainDeleteRequests map[string]domain.ProjectDomainDeleteRequest
	contestSubmissions          map[string]domain.ContestSubmission
	repairRequests              map[string]domain.RepairRequest
	repairAIJobs                map[string]domain.RepairAIJob
	repairAIMessages            map[string][]domain.RepairAIMessage
	templateSubmissions         map[string]domain.TemplateSubmission
	reservedDomains             map[string]bool
}

func NewMemoryStore(publicBase string) *MemoryStore {
	return &MemoryStore{
		publicBase:                  publicBase,
		users:                       map[string]domain.User{},
		usersByMail:                 map[string]string{},
		authCodes:                   map[string]memoryAuthCode{},
		projectEmailCodes:           []memoryProjectEmailCode{},
		projectEmailQuotaDaily:      map[string]int{},
		sessions:                    map[string]memorySession{},
		projects:                    map[string]domain.Project{},
		projectUser:                 map[string]string{},
		projectFavorites:            map[string]map[string]bool{},
		authorFollows:               map[string]map[string]bool{},
		projectNotificationEvents:   map[string]bool{},
		projectDiscussions:          map[string]domain.ProjectDiscussion{},
		projectDiscussionComments:   map[string][]domain.ProjectDiscussionComment{},
		projectProposals:            map[string]domain.ProjectProposal{},
		appBuildSettings:            map[string]domain.AppBuildSettings{},
		appBuildJobs:                map[string]domain.AppBuildJob{},
		appUpdateChannels:           map[string]domain.AppUpdateInfo{},
		collections:                 map[string][]domain.Collection{},
		records:                     map[string][]domain.Record{},
		releases:                    map[string][]domain.Release{},
		publicKeys:                  map[string]string{},
		usage:                       map[string]domain.ProjectUsage{},
		dailyStats:                  map[string]domain.ProjectDailyStats{},
		upgradeRequests:             map[string]domain.UpgradeRequest{},
		projectDomains:              map[string]domain.ProjectDomain{},
		projectDomainDeleteRequests: map[string]domain.ProjectDomainDeleteRequest{},
		contestSubmissions:          map[string]domain.ContestSubmission{},
		repairRequests:              map[string]domain.RepairRequest{},
		repairAIJobs:                map[string]domain.RepairAIJob{},
		repairAIMessages:            map[string][]domain.RepairAIMessage{},
		templateSubmissions:         map[string]domain.TemplateSubmission{},
		reservedDomains: map[string]bool{
			"www": true, "web": true, "game": true, "a11y": true, "save": true, "api": true, "admin": true,
			"mail": true, "smtp": true, "static": true, "assets": true, "auth": true, "login": true, "pay": true,
		},
	}
}

func (s *MemoryStore) GetUserByEmail(_ context.Context, email string) (domain.User, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.usersByMail[email]
	if !ok {
		return domain.User{}, false, nil
	}
	return s.users[id], true, nil
}

func (s *MemoryStore) GetUserBySessionToken(_ context.Context, token string) (domain.User, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[token]
	if !ok || session.ExpiresAt.Before(time.Now().UTC()) {
		return domain.User{}, false, nil
	}
	return s.users[session.UserID], true, nil
}

func (s *MemoryStore) GetUserByID(_ context.Context, userID string) (domain.User, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[userID]
	if !ok {
		return domain.User{}, false, nil
	}
	return user, true, nil
}

func (s *MemoryStore) UpdateUserUsername(_ context.Context, userID, username string) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.users[userID]
	user.Username = username
	s.users[userID] = user
	return user, nil
}

func (s *MemoryStore) UpdateUserRoleAndPlan(_ context.Context, userID, role, planCode string) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.users[userID]
	user.Role = role
	user.PlanCode = planCode
	s.users[userID] = user
	return user, nil
}

func (s *MemoryStore) SyncProjectUsernames(_ context.Context, userID, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for projectID, ownerUserID := range s.projectUser {
		if ownerUserID != userID {
			continue
		}
		project := s.projects[projectID]
		project.Username = username
		project.PublicURL = buildPublicURL(s.publicBase, username, project.Slug)
		s.projects[projectID] = project
	}
	return nil
}

func (s *MemoryStore) CreateOrRefreshAuthCode(_ context.Context, input domain.AuthCodeRequestInput, code string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authCodes[input.Email] = memoryAuthCode{
		Email:     input.Email,
		Username:  input.Username,
		Code:      code,
		ExpiresAt: expiresAt,
	}
	return nil
}

func (s *MemoryStore) ConsumeAuthCode(_ context.Context, input domain.AuthCodeVerifyInput, now time.Time) (domain.User, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	authCode, ok := s.authCodes[input.Email]
	if !ok || authCode.Used || authCode.Code != input.Code || authCode.ExpiresAt.Before(now) {
		return domain.User{}, false, nil
	}
	authCode.Used = true
	s.authCodes[input.Email] = authCode

	if userID, ok := s.usersByMail[input.Email]; ok {
		user := s.users[userID]
		if shouldReplaceUsername(user, authCode.Username) {
			user.Username = authCode.Username
			s.users[userID] = user
		}
		return user, true, nil
	}

	user := domain.User{
		ID:        fmt.Sprintf("user_%d", time.Now().UnixNano()),
		Username:  authCode.Username,
		Email:     authCode.Email,
		Status:    "active",
		Role:      domain.RoleUser,
		PlanCode:  domain.PlanFree,
		CreatedAt: now,
	}
	s.users[user.ID] = user
	s.usersByMail[user.Email] = user.ID
	return user, true, nil
}

func (s *MemoryStore) CreateProjectEmailCode(_ context.Context, projectID, ownerUserID, email, purpose, code string, expiresAt, now time.Time, dailyLimit int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.projectEmailCodes {
		if item.ProjectID == projectID && item.Email == email && item.Purpose == purpose && item.CreatedAt.After(now.Add(-60*time.Second)) {
			return 0, fmt.Errorf("同一个邮箱 60 秒内只能发送一次验证码")
		}
	}
	day := now.Format("2006-01-02")
	key := ownerUserID + ":" + day
	used := s.projectEmailQuotaDaily[key]
	if dailyLimit > 0 && used >= dailyLimit {
		return used, fmt.Errorf("这个账号今天的作品验证码邮件额度已经用完了")
	}
	used++
	s.projectEmailQuotaDaily[key] = used
	s.projectEmailCodes = append(s.projectEmailCodes, memoryProjectEmailCode{
		ProjectID:   projectID,
		OwnerUserID: ownerUserID,
		Email:       email,
		Purpose:     purpose,
		Code:        code,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
	})
	return used, nil
}

func (s *MemoryStore) ConsumeProjectEmailCode(_ context.Context, projectID, email, purpose, code string, now time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.projectEmailCodes) - 1; i >= 0; i-- {
		item := s.projectEmailCodes[i]
		if item.ProjectID == projectID && item.Email == email && item.Purpose == purpose && item.Code == code && !item.Used && item.ExpiresAt.After(now) {
			s.projectEmailCodes[i].Used = true
			return true, nil
		}
	}
	return false, nil
}

func (s *MemoryStore) CreateSession(_ context.Context, userID, token string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = memorySession{UserID: userID, Token: token, ExpiresAt: expiresAt}
	return nil
}

func (s *MemoryStore) DeleteSession(_ context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
	return nil
}

func (s *MemoryStore) ListProjects(_ context.Context, userID string) ([]domain.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]domain.Project, 0, len(s.projects))
	for _, project := range s.projects {
		if s.projectUser[project.ID] == userID {
			items = append(items, s.enrichProjectLocked(project, userID))
		}
	}

	return items, nil
}

func (s *MemoryStore) ListAdminUsers(_ context.Context) ([]domain.AdminUserSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]domain.AdminUserSummary, 0, len(s.users))
	for _, user := range s.users {
		count := 0
		for _, ownerID := range s.projectUser {
			if ownerID == user.ID {
				count += 1
			}
		}
		items = append(items, domain.AdminUserSummary{
			ID:           user.ID,
			Email:        user.Email,
			Username:     user.Username,
			Status:       user.Status,
			Role:         user.Role,
			PlanCode:     user.PlanCode,
			ProjectCount: count,
			CreatedAt:    user.CreatedAt,
		})
	}
	return items, nil
}

func (s *MemoryStore) ListPublicProjects(_ context.Context) ([]domain.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]domain.Project, 0, len(s.projects))
	for _, project := range s.projects {
		if project.CurrentRelease != "" && project.ShowOnProfile {
			items = append(items, s.enrichProjectLocked(project, ""))
		}
	}

	return items, nil
}

func (s *MemoryStore) enrichProjectLocked(project domain.Project, viewerUserID string) domain.Project {
	project.PublicURL = buildPublicURL(s.publicBase, project.Username, project.Slug)
	project.FavoritesCount = len(s.projectFavorites[project.ID])
	project.ForksCount = 0
	for _, item := range s.projects {
		if item.ForkedFromProjectID == project.ID {
			project.ForksCount++
		}
	}
	if viewerUserID != "" {
		project.FavoritedByMe = s.projectFavorites[project.ID][viewerUserID]
	}
	if project.ForkedFromProjectID != "" {
		if source, ok := s.projects[project.ForkedFromProjectID]; ok {
			project.ForkedFromUsername = source.Username
			project.ForkedFromProjectName = source.Name
			project.ForkedFromProjectURL = buildPublicURL(s.publicBase, source.Username, source.Slug)
			project.CanSubmitProposal = source.CurrentRelease != "" && source.AllowForks
		} else {
			project.ForkedFromUsername = project.ForkedFromSnapshotOwner
			project.ForkedFromProjectName = project.ForkedFromSnapshotName
		}
	}
	return project
}

func (s *MemoryStore) GetPublicProject(_ context.Context, projectID string) (domain.Project, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	project, ok := s.projects[projectID]
	if !ok || project.Visibility != "public" || project.CurrentRelease == "" {
		return domain.Project{}, false, nil
	}
	return s.enrichProjectLocked(project, ""), true, nil
}

func (s *MemoryStore) GetAuthorProfile(_ context.Context, username string) (domain.AuthorProfile, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var user domain.User
	found := false
	for _, item := range s.users {
		if item.Username == username {
			user = item
			found = true
			break
		}
	}
	if !found {
		return domain.AuthorProfile{}, false, nil
	}
	profile := domain.AuthorProfile{Username: user.Username, DisplayName: user.Username, JoinedAt: user.CreatedAt}
	for followerID, targets := range s.authorFollows {
		if targets[user.ID] {
			profile.FollowersCount++
		}
		if followerID == user.ID {
			profile.FollowingCount = len(targets)
		}
	}
	for _, project := range s.projects {
		if s.projectUser[project.ID] == user.ID && project.CurrentRelease != "" && project.ShowOnProfile {
			project = s.enrichProjectLocked(project, "")
			profile.Projects = append(profile.Projects, project)
			profile.ProjectCount++
			profile.FavoritesCount += project.FavoritesCount
			profile.ForksCount += project.ForksCount
		}
	}
	return profile, true, nil
}

func (s *MemoryStore) IsFollowingAuthor(_ context.Context, followerUserID, targetUsername string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	targetID := ""
	for _, user := range s.users {
		if user.Username == targetUsername {
			targetID = user.ID
			break
		}
	}
	if targetID == "" {
		return false, nil
	}
	return s.authorFollows[followerUserID][targetID], nil
}

func (s *MemoryStore) FollowAuthor(_ context.Context, followerUserID, targetUsername string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	targetID := ""
	for _, user := range s.users {
		if user.Username == targetUsername {
			targetID = user.ID
			break
		}
	}
	if targetID == "" || targetID == followerUserID {
		return nil
	}
	if _, ok := s.authorFollows[followerUserID]; !ok {
		s.authorFollows[followerUserID] = map[string]bool{}
	}
	s.authorFollows[followerUserID][targetID] = true
	return nil
}

func (s *MemoryStore) UnfollowAuthor(_ context.Context, followerUserID, targetUsername string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	targetID := ""
	for _, user := range s.users {
		if user.Username == targetUsername {
			targetID = user.ID
			break
		}
	}
	delete(s.authorFollows[followerUserID], targetID)
	return nil
}

func (s *MemoryStore) ListFollowedAuthors(_ context.Context, followerUserID string) ([]domain.AuthorSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []domain.AuthorSummary{}
	for targetID := range s.authorFollows[followerUserID] {
		user, ok := s.users[targetID]
		if !ok {
			continue
		}
		item := domain.AuthorSummary{UserID: user.ID, Username: user.Username, DisplayName: user.Username, JoinedAt: user.CreatedAt}
		for _, project := range s.projects {
			if s.projectUser[project.ID] == targetID && project.CurrentRelease != "" && project.ShowOnProfile {
				item.ProjectCount++
			}
		}
		for _, targets := range s.authorFollows {
			if targets[targetID] {
				item.FollowersCount++
			}
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].JoinedAt.After(items[j].JoinedAt) })
	return items, nil
}

func (s *MemoryStore) ListAuthorFollowerRecipients(_ context.Context, authorUsername string) ([]domain.NotificationRecipient, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	authorID := ""
	for _, user := range s.users {
		if user.Username == authorUsername {
			authorID = user.ID
			break
		}
	}
	if authorID == "" {
		return []domain.NotificationRecipient{}, nil
	}
	items := []domain.NotificationRecipient{}
	for followerID, targets := range s.authorFollows {
		if !targets[authorID] {
			continue
		}
		follower, ok := s.users[followerID]
		if ok && follower.Status == "active" {
			items = append(items, domain.NotificationRecipient{Email: follower.Email, Username: follower.Username})
		}
	}
	return items, nil
}

func (s *MemoryStore) TryCreateProjectNotificationEvent(_ context.Context, projectID, eventKey string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := projectID + ":" + eventKey
	if s.projectNotificationEvents[key] {
		return false, nil
	}
	s.projectNotificationEvents[key] = true
	return true, nil
}

func (s *MemoryStore) IsProjectFavorited(_ context.Context, userID, projectID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.projectFavorites[projectID][userID], nil
}

func (s *MemoryStore) AddProjectFavorite(_ context.Context, userID, projectID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projectFavorites[projectID]; !ok {
		s.projectFavorites[projectID] = map[string]bool{}
	}
	s.projectFavorites[projectID][userID] = true
	return nil
}

func (s *MemoryStore) RemoveProjectFavorite(_ context.Context, userID, projectID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.projectFavorites[projectID], userID)
	return nil
}

func (s *MemoryStore) ListUserFavoriteProjects(_ context.Context, userID string) ([]domain.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []domain.Project{}
	for projectID, users := range s.projectFavorites {
		if !users[userID] {
			continue
		}
		project, ok := s.projects[projectID]
		if !ok || project.Visibility != "public" || project.CurrentRelease == "" {
			continue
		}
		project = s.enrichProjectLocked(project, userID)
		project.FavoritedByMe = true
		items = append(items, project)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (s *MemoryStore) ListProjectDiscussions(_ context.Context, projectID, status, searchQuery string) ([]domain.ProjectDiscussion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []domain.ProjectDiscussion{}
	searchQuery = strings.ToLower(strings.TrimSpace(searchQuery))
	for _, item := range s.projectDiscussions {
		if item.ProjectID != projectID {
			continue
		}
		if status != "" && item.Status != status {
			continue
		}
		if searchQuery != "" {
			haystack := strings.ToLower(item.Title + "\n" + item.Body + "\n" + item.AuthorUsername)
			if !strings.Contains(haystack, searchQuery) {
				continue
			}
		}
		item.CommentsCount = len(s.projectDiscussionComments[item.ID])
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].LastCommentedAt.After(items[j].LastCommentedAt) })
	return items, nil
}

func (s *MemoryStore) GetProjectDiscussion(_ context.Context, discussionID string) (domain.ProjectDiscussion, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.projectDiscussions[discussionID]
	if !ok {
		return domain.ProjectDiscussion{}, false, nil
	}
	item.CommentsCount = len(s.projectDiscussionComments[item.ID])
	return item, true, nil
}

func (s *MemoryStore) CreateProjectDiscussion(_ context.Context, projectID, authorUserID string, input domain.ProjectDiscussionCreateInput) (domain.ProjectDiscussion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	project := s.projects[projectID]
	user := s.users[authorUserID]
	item := domain.ProjectDiscussion{
		ID:              fmt.Sprintf("discussion_%d", now.UnixNano()),
		ProjectID:       projectID,
		ProjectName:     project.Name,
		ProjectURL:      buildPublicURL(s.publicBase, project.Username, project.Slug),
		AuthorUserID:    authorUserID,
		AuthorUsername:  user.Username,
		AuthorEmail:     user.Email,
		Title:           input.Title,
		Body:            input.Body,
		Status:          "open",
		LastCommentedAt: now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	s.projectDiscussions[item.ID] = item
	return item, nil
}

func (s *MemoryStore) UpdateProjectDiscussionStatus(_ context.Context, discussionID, status, operatorUserID string) (domain.ProjectDiscussion, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.projectDiscussions[discussionID]
	if !ok {
		return domain.ProjectDiscussion{}, false, nil
	}
	now := time.Now().UTC()
	item.Status = status
	item.UpdatedAt = now
	if status == "closed" {
		user := s.users[operatorUserID]
		item.ClosedByUserID = operatorUserID
		item.ClosedByUsername = user.Username
		item.ClosedAt = now
	} else {
		item.ClosedByUserID = ""
		item.ClosedByUsername = ""
		item.ClosedAt = time.Time{}
	}
	s.projectDiscussions[discussionID] = item
	return item, true, nil
}

func (s *MemoryStore) ListProjectDiscussionComments(_ context.Context, discussionID string) ([]domain.ProjectDiscussionComment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := append([]domain.ProjectDiscussionComment{}, s.projectDiscussionComments[discussionID]...)
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}

func (s *MemoryStore) CreateProjectDiscussionComment(_ context.Context, discussionID, authorUserID string, input domain.ProjectDiscussionCommentCreateInput) (domain.ProjectDiscussionComment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	user := s.users[authorUserID]
	item := domain.ProjectDiscussionComment{
		ID:             fmt.Sprintf("discussion_comment_%d", now.UnixNano()),
		DiscussionID:   discussionID,
		AuthorUserID:   authorUserID,
		AuthorUsername: user.Username,
		AuthorEmail:    user.Email,
		Body:           input.Body,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	s.projectDiscussionComments[discussionID] = append(s.projectDiscussionComments[discussionID], item)
	discussion := s.projectDiscussions[discussionID]
	discussion.LastCommentedAt = now
	discussion.UpdatedAt = now
	s.projectDiscussions[discussionID] = discussion
	return item, nil
}

func (s *MemoryStore) enrichProjectProposalLocked(item domain.ProjectProposal) domain.ProjectProposal {
	source := s.projects[item.SourceProjectID]
	target := s.projects[item.TargetProjectID]
	author := s.users[item.AuthorUserID]
	owner := s.users[item.TargetOwnerUserID]
	item.SourceProjectName = source.Name
	item.SourceProjectURL = buildPublicURL(s.publicBase, source.Username, source.Slug)
	item.TargetProjectName = target.Name
	item.TargetProjectURL = buildPublicURL(s.publicBase, target.Username, target.Slug)
	item.AuthorUsername = author.Username
	item.AuthorEmail = author.Email
	item.TargetOwnerUsername = owner.Username
	item.TargetOwnerEmail = owner.Email
	return item
}

func (s *MemoryStore) ListProjectProposalsForSource(_ context.Context, sourceProjectID string) ([]domain.ProjectProposal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []domain.ProjectProposal{}
	for _, item := range s.projectProposals {
		if item.SourceProjectID == sourceProjectID {
			items = append(items, s.enrichProjectProposalLocked(item))
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (s *MemoryStore) ListProjectProposalsForTarget(_ context.Context, targetProjectID, status string) ([]domain.ProjectProposal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status = strings.TrimSpace(status)
	items := []domain.ProjectProposal{}
	for _, item := range s.projectProposals {
		if item.TargetProjectID != targetProjectID {
			continue
		}
		if status != "" && item.Status != status {
			continue
		}
		items = append(items, s.enrichProjectProposalLocked(item))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (s *MemoryStore) ListUserProjectProposals(_ context.Context, userID string) ([]domain.ProjectProposal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []domain.ProjectProposal{}
	for _, item := range s.projectProposals {
		if item.AuthorUserID == userID || item.TargetOwnerUserID == userID {
			items = append(items, s.enrichProjectProposalLocked(item))
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (s *MemoryStore) GetProjectProposal(_ context.Context, proposalID string) (domain.ProjectProposal, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.projectProposals[proposalID]
	if !ok {
		return domain.ProjectProposal{}, false, nil
	}
	return s.enrichProjectProposalLocked(item), true, nil
}

func (s *MemoryStore) CreateProjectProposal(_ context.Context, input domain.ProjectProposal) (domain.ProjectProposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if input.ID == "" {
		input.ID = fmt.Sprintf("proposal_%d", now.UnixNano())
	}
	input.Status = "open"
	input.CreatedAt = now
	input.UpdatedAt = now
	s.projectProposals[input.ID] = input
	return s.enrichProjectProposalLocked(input), nil
}

func (s *MemoryStore) ReviewProjectProposal(_ context.Context, proposalID, status, note, reviewedByUserID, mergedReleaseID string) (domain.ProjectProposal, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.projectProposals[proposalID]
	if !ok {
		return domain.ProjectProposal{}, false, nil
	}
	now := time.Now().UTC()
	item.Status = status
	item.ReviewNote = note
	item.ReviewedBy = reviewedByUserID
	item.MergedReleaseID = mergedReleaseID
	item.ReviewedAt = now
	item.UpdatedAt = now
	s.projectProposals[proposalID] = item
	return s.enrichProjectProposalLocked(item), true, nil
}

func (s *MemoryStore) ListAdminProjects(_ context.Context) ([]domain.AdminProjectSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]domain.AdminProjectSummary, 0, len(s.projects))
	for _, project := range s.projects {
		ownerID := s.projectUser[project.ID]
		owner := s.users[ownerID]
		items = append(items, domain.AdminProjectSummary{
			ID:               project.ID,
			Name:             project.Name,
			Slug:             project.Slug,
			Username:         project.Username,
			OwnerUserID:      ownerID,
			OwnerEmail:       owner.Email,
			Interactive:      project.Interactive,
			AnalyticsEnabled: project.AnalyticsEnabled,
			Visibility:       project.Visibility,
			CurrentRelease:   project.CurrentRelease,
			PublicURL:        project.PublicURL,
			CreatedAt:        project.CreatedAt,
		})
	}
	return items, nil
}

func (s *MemoryStore) CreateProject(_ context.Context, userID string, input domain.ProjectCreateInput) (domain.Project, error) {
	projectID := fmt.Sprintf("proj_%d", time.Now().UnixNano())
	project := domain.Project{
		ID:               projectID,
		Name:             input.Name,
		Username:         input.Username,
		Slug:             input.Slug,
		Interactive:      input.Interactive,
		AnalyticsEnabled: input.AnalyticsEnabled,
		ShowOnProfile:    input.ShowOnProfile,
		AllowForks:       input.AllowForks,
		Visibility:       "unlisted",
		PublicURL:        buildPublicURL(s.publicBase, input.Username, input.Slug),
		CreatedAt:        time.Now().UTC(),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.projects[project.ID] = project
	s.projectUser[project.ID] = userID
	s.publicKeys[project.ID] = fmt.Sprintf("pk_%d", time.Now().UnixNano())
	return project, nil
}

func (s *MemoryStore) CreateForkProject(_ context.Context, userID string, input domain.ProjectCreateInput, source domain.Project) (domain.Project, error) {
	projectID := fmt.Sprintf("proj_%d", time.Now().UnixNano())
	project := domain.Project{
		ID:                      projectID,
		Name:                    input.Name,
		Username:                input.Username,
		Slug:                    input.Slug,
		Interactive:             input.Interactive,
		AnalyticsEnabled:        input.AnalyticsEnabled,
		ShowOnProfile:           input.ShowOnProfile,
		AllowForks:              input.AllowForks,
		Visibility:              "unlisted",
		PublicURL:               buildPublicURL(s.publicBase, input.Username, input.Slug),
		ForkedFromProjectID:     source.ID,
		ForkedFromReleaseID:     source.CurrentRelease,
		ForkedFromSnapshotName:  source.Name,
		ForkedFromSnapshotOwner: source.Username,
		ForkedFromProjectURL:    source.PublicURL,
		CreatedAt:               time.Now().UTC(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projects[project.ID] = project
	s.projectUser[project.ID] = userID
	s.publicKeys[project.ID] = fmt.Sprintf("pk_%d", time.Now().UnixNano())
	return s.enrichProjectLocked(project, userID), nil
}

func (s *MemoryStore) GetProject(_ context.Context, userID, projectID string) (domain.Project, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	project, ok := s.projects[projectID]
	if !ok || s.projectUser[projectID] != userID {
		return domain.Project{}, false, nil
	}
	return s.enrichProjectLocked(project, userID), true, nil
}

func (s *MemoryStore) DeleteProject(_ context.Context, userID, projectID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	project, ok := s.projects[projectID]
	if !ok || s.projectUser[projectID] != userID {
		return false, nil
	}

	delete(s.projects, projectID)
	delete(s.projectUser, projectID)
	delete(s.publicKeys, projectID)
	delete(s.collections, projectID)
	delete(s.releases, projectID)

	for key := range s.records {
		if strings.HasPrefix(key, projectID+":") {
			delete(s.records, key)
		}
	}
	_ = project
	return true, nil
}

func (s *MemoryStore) GetProjectPublicAccess(_ context.Context, projectID string) (domain.PublicProjectAccess, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	project, ok := s.projects[projectID]
	if !ok {
		return domain.PublicProjectAccess{}, false, nil
	}
	return domain.PublicProjectAccess{
		Project:     project,
		PublicKey:   s.publicKeys[projectID],
		OwnerUserID: s.projectUser[projectID],
		OwnerRole:   s.users[s.projectUser[projectID]].Role,
		OwnerPlan:   s.users[s.projectUser[projectID]].PlanCode,
	}, true, nil
}

func (s *MemoryStore) CreateContestSubmission(_ context.Context, input domain.ContestSubmission) (domain.ContestSubmission, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.contestSubmissions {
		if item.UserID == input.UserID && item.ProjectID == input.ProjectID {
			return domain.ContestSubmission{}, fmt.Errorf("这个作品已经提交过比赛")
		}
	}
	now := time.Now().UTC()
	input.ID = fmt.Sprintf("contest_%d", now.UnixNano())
	input.Status = "pending"
	input.CreatedAt = now
	input.UpdatedAt = now
	s.contestSubmissions[input.ID] = input
	return input, nil
}

func (s *MemoryStore) GetContestSubmissionByUserProject(_ context.Context, userID, projectID string) (domain.ContestSubmission, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.contestSubmissions {
		if item.UserID == userID && item.ProjectID == projectID {
			return item, true, nil
		}
	}
	return domain.ContestSubmission{}, false, nil
}

func (s *MemoryStore) ListAdminContestSubmissions(_ context.Context, status string) ([]domain.ContestSubmission, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []domain.ContestSubmission{}
	for _, item := range s.contestSubmissions {
		if status == "" || item.Status == status {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (s *MemoryStore) UpdateContestSubmissionReview(_ context.Context, submissionID, status, adminNote, reviewedBy string) (domain.ContestSubmission, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.contestSubmissions[submissionID]
	if !ok {
		return domain.ContestSubmission{}, fmt.Errorf("找不到这个参赛作品")
	}
	item.Status = status
	item.AdminNote = adminNote
	item.ReviewedBy = reviewedBy
	item.ReviewedAt = time.Now().UTC()
	item.UpdatedAt = item.ReviewedAt
	s.contestSubmissions[submissionID] = item
	return item, nil
}

func (s *MemoryStore) UpdateProjectVisibility(_ context.Context, userID, projectID, visibility string) (domain.Project, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	project, ok := s.projects[projectID]
	if !ok || s.projectUser[projectID] != userID {
		return domain.Project{}, false, nil
	}
	project.Visibility = visibility
	s.projects[projectID] = project
	return project, true, nil
}

func (s *MemoryStore) UpdateProjectPath(_ context.Context, userID, projectID, slug string) (domain.Project, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	project, ok := s.projects[projectID]
	if !ok || s.projectUser[projectID] != userID {
		return domain.Project{}, false, nil
	}
	project.Slug = slug
	project.PublicURL = buildPublicURL(s.publicBase, project.Username, slug)
	s.projects[projectID] = project
	return project, true, nil
}

func (s *MemoryStore) UpdateProjectSettings(_ context.Context, userID, projectID string, input domain.ProjectSettingsUpdateInput) (domain.Project, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	project, ok := s.projects[projectID]
	if !ok || s.projectUser[projectID] != userID {
		return domain.Project{}, false, nil
	}
	project.Name = input.Name
	project.Slug = input.Slug
	project.Interactive = input.Interactive
	project.AnalyticsEnabled = input.AnalyticsEnabled
	project.ShowOnProfile = input.ShowOnProfile
	project.AllowForks = input.AllowForks
	project.PublicURL = buildPublicURL(s.publicBase, project.Username, input.Slug)
	s.projects[projectID] = project
	return project, true, nil
}

func (s *MemoryStore) CreateCollection(_ context.Context, projectID string, input domain.CollectionCreateInput) (domain.Collection, error) {
	collection := domain.Collection{
		ID:          fmt.Sprintf("col_%d", time.Now().UnixNano()),
		ProjectID:   projectID,
		Name:        input.Name,
		Permissions: normalizeStoredPermissions(input.Permissions),
		Fields:      input.Fields,
		CreatedAt:   time.Now().UTC(),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.collections[collection.ProjectID] = append(s.collections[collection.ProjectID], collection)
	return collection, nil
}

func (s *MemoryStore) GetAppBuildSettings(_ context.Context, userID, projectID string) (domain.AppBuildSettings, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.appBuildSettings[projectID]
	if !ok || item.UserID != userID {
		return domain.AppBuildSettings{}, false, nil
	}
	return item, true, nil
}

func (s *MemoryStore) UpsertAppBuildSettings(_ context.Context, userID, projectID string, input domain.AppBuildSettingsInput) (domain.AppBuildSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	item := s.appBuildSettings[projectID]
	if item.ID == "" {
		item.ID = fmt.Sprintf("abs_%d", now.UnixNano())
		item.UserID = userID
		item.ProjectID = projectID
		item.CreatedAt = now
	}
	item.AppName = input.AppName
	item.AndroidEnabled = input.AndroidEnabled
	item.WindowsEnabled = input.WindowsEnabled
	item.AutoUpdate = input.AutoUpdate
	item.AndroidPackageName = input.AndroidPackageName
	item.WindowsPackageName = input.WindowsPackageName
	item.UpdatedAt = now
	s.appBuildSettings[projectID] = item
	return item, nil
}

func (s *MemoryStore) CreateAppBuildJob(_ context.Context, input domain.AppBuildJob) (domain.AppBuildJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if input.ID == "" {
		input.ID = fmt.Sprintf("abj_%d", now.UnixNano())
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = now
	}
	input.UpdatedAt = now
	s.appBuildJobs[input.ID] = input
	return input, nil
}

func (s *MemoryStore) GetAppBuildJob(_ context.Context, jobID string) (domain.AppBuildJob, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.appBuildJobs[jobID]
	return item, ok, nil
}

func (s *MemoryStore) ListAppBuildJobs(_ context.Context, userID, projectID string) ([]domain.AppBuildJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []domain.AppBuildJob{}
	for _, item := range s.appBuildJobs {
		if item.UserID == userID && item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (s *MemoryStore) CountAppBuildJobsForUserSince(_ context.Context, userID, platform string, since time.Time) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, item := range s.appBuildJobs {
		if item.UserID == userID && item.Platform == platform && !item.CreatedAt.Before(since) &&
			(item.Status == "pending" || item.Status == "building" || item.Status == "succeeded") {
			count++
		}
	}
	return count, nil
}

func (s *MemoryStore) CompleteAppBuildJob(_ context.Context, jobID string, input domain.AppBuildCompleteInput) (domain.AppBuildJob, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.appBuildJobs[jobID]
	if !ok {
		return domain.AppBuildJob{}, false, nil
	}
	item.Status = input.Status
	item.ArtifactPath = input.ArtifactPath
	item.ArtifactSHA256 = input.ArtifactSHA256
	item.ArtifactSize = input.ArtifactSize
	item.GitHubRunID = input.GitHubRunID
	item.ErrorMessage = input.ErrorMessage
	item.UpdatedAt = time.Now().UTC()
	s.appBuildJobs[jobID] = item
	if item.Status == "succeeded" && item.ArtifactPath != "" {
		updateDownloadURL := item.ArtifactPath
		if item.Platform == "windows" {
			updateDownloadURL = item.ArtifactPath + "?kind=update"
		}
		s.appUpdateChannels[item.ProjectID+"|"+item.Platform] = domain.AppUpdateInfo{
			HasUpdate:         false,
			LatestVersionCode: item.VersionCode,
			LatestVersionName: item.VersionName,
			ReleaseNote:       "作者更新了作品内容",
			DownloadURL:       updateDownloadURL,
			SHA256:            item.ArtifactSHA256,
			Size:              item.ArtifactSize,
		}
	}
	return item, true, nil
}

func (s *MemoryStore) GetAppUpdateInfo(_ context.Context, projectID, platform string, versionCode int) (domain.AppUpdateInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	info := s.appUpdateChannels[projectID+"|"+platform]
	info.HasUpdate = info.LatestVersionCode > versionCode && info.DownloadURL != ""
	return info, nil
}

func (s *MemoryStore) ListCollections(_ context.Context, projectID string) ([]domain.Collection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := s.collections[projectID]
	if items == nil {
		return []domain.Collection{}, nil
	}
	return normalizeCollectionList(items), nil
}

func (s *MemoryStore) GetCollectionByName(_ context.Context, projectID, collectionName string) (domain.Collection, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, collection := range s.collections[projectID] {
		if collection.Name == collectionName {
			collection.Permissions = normalizeStoredPermissions(collection.Permissions)
			return collection, true, nil
		}
	}
	return domain.Collection{}, false, nil
}

func (s *MemoryStore) UpdateCollection(_ context.Context, projectID, collectionName string, input domain.CollectionUpdateInput) (domain.Collection, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := s.collections[projectID]
	for index, collection := range items {
		if collection.Name == collectionName {
			collection.Permissions = normalizeStoredPermissions(input.Permissions)
			collection.Fields = input.Fields
			items[index] = collection
			s.collections[projectID] = items
			return collection, true, nil
		}
	}
	return domain.Collection{}, false, nil
}

func (s *MemoryStore) DeleteCollection(_ context.Context, projectID, collectionName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := s.collections[projectID]
	next := make([]domain.Collection, 0, len(items))
	var targetID string
	for _, item := range items {
		if item.Name == collectionName {
			targetID = item.ID
			continue
		}
		next = append(next, item)
	}
	if targetID == "" {
		return fmt.Errorf("找不到这个作品数据表")
	}
	s.collections[projectID] = next

	key := projectID + ":" + collectionName
	delete(s.records, key)
	_ = targetID
	return nil
}

func (s *MemoryStore) ListRecords(_ context.Context, projectID, collectionName string) ([]domain.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := projectID + ":" + collectionName
	items := s.records[key]
	if items == nil {
		return []domain.Record{}, nil
	}
	return items, nil
}

func (s *MemoryStore) CreateRecord(_ context.Context, projectID, collectionName, createdByUserID string, input domain.RecordCreateInput) (domain.Record, error) {
	collection, ok, _ := s.GetCollectionByName(context.Background(), projectID, collectionName)
	if !ok {
		return domain.Record{}, fmt.Errorf("找不到这个集合")
	}

	record := domain.Record{
		ID:           fmt.Sprintf("rec_%d", time.Now().UnixNano()),
		ProjectID:    projectID,
		CollectionID: collection.ID,
		Data:         input.Data,
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	_ = createdByUserID

	s.mu.Lock()
	defer s.mu.Unlock()
	key := projectID + ":" + collectionName
	s.records[key] = append(s.records[key], record)
	return record, nil
}

func (s *MemoryStore) GetRecord(_ context.Context, projectID, collectionName, recordID string) (domain.Record, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := projectID + ":" + collectionName
	for _, item := range s.records[key] {
		if item.ID == recordID {
			return item, true, nil
		}
	}
	return domain.Record{}, false, nil
}

func (s *MemoryStore) UpdateRecord(_ context.Context, projectID, collectionName, recordID string, input domain.RecordUpdateInput) (domain.Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := projectID + ":" + collectionName
	items := s.records[key]
	for index, item := range items {
		if item.ID != recordID {
			continue
		}
		item.Data = input.Data
		item.UpdatedAt = time.Now().UTC()
		items[index] = item
		s.records[key] = items
		return item, true, nil
	}
	return domain.Record{}, false, nil
}

func (s *MemoryStore) DeleteRecord(_ context.Context, projectID, collectionName, recordID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := projectID + ":" + collectionName
	items := s.records[key]
	next := make([]domain.Record, 0, len(items))
	found := false
	for _, item := range items {
		if item.ID == recordID {
			found = true
			continue
		}
		next = append(next, item)
	}
	if found {
		s.records[key] = next
	}
	return found, nil
}

func (s *MemoryStore) ProjectDataStats(_ context.Context, projectID string) (domain.ProjectDataStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var recordCount int64
	var dataBytes int64
	for key, items := range s.records {
		if !strings.HasPrefix(key, projectID+":") {
			continue
		}
		recordCount += int64(len(items))
		for _, item := range items {
			body, _ := json.Marshal(item.Data)
			dataBytes += int64(len(body))
		}
	}
	return domain.ProjectDataStats{
		ProjectID:       projectID,
		RecordCount:     recordCount,
		DataBytes:       dataBytes,
		CollectionCount: int64(len(s.collections[projectID])),
	}, nil
}

func (s *MemoryStore) GetMonthlyProjectUsage(_ context.Context, projectID, monthKey string) (domain.ProjectUsage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := projectID + ":" + monthKey
	if usage, ok := s.usage[key]; ok {
		return usage, nil
	}
	return domain.ProjectUsage{ProjectID: projectID, MonthKey: monthKey}, nil
}

func (s *MemoryStore) IncrementProjectUsage(_ context.Context, projectID, monthKey string, queryDelta, writeDelta int64) (domain.ProjectUsage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := projectID + ":" + monthKey
	usage := s.usage[key]
	usage.ProjectID = projectID
	usage.MonthKey = monthKey
	usage.QueryCount += queryDelta
	usage.WriteCount += writeDelta
	usage.UpdatedAt = time.Now().UTC()
	s.usage[key] = usage
	return usage, nil
}

func (s *MemoryStore) IncrementProjectDailyStats(_ context.Context, projectID, day string, pageViews, apiRequests, apiSuccesses, apiFailures int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := projectID + ":" + day
	item := s.dailyStats[key]
	item.Date = day
	item.PageViews += pageViews
	item.APIRequests += apiRequests
	item.APISuccesses += apiSuccesses
	item.APIFailures += apiFailures
	s.dailyStats[key] = item
	return nil
}

func (s *MemoryStore) ProjectStats(_ context.Context, projectID, from, to string) (domain.ProjectStatsSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	summary := domain.ProjectStatsSummary{ProjectID: projectID, From: from, To: to, Items: []domain.ProjectDailyStats{}}
	fromDay, _ := time.Parse("2006-01-02", from)
	toDay, _ := time.Parse("2006-01-02", to)
	for key, item := range s.dailyStats {
		if !strings.HasPrefix(key, projectID+":") {
			continue
		}
		day, err := time.Parse("2006-01-02", item.Date)
		if err != nil || day.Before(fromDay) || day.After(toDay) {
			continue
		}
		summary.TotalPageViews += item.PageViews
		summary.TotalAPIRequests += item.APIRequests
		summary.TotalAPISuccesses += item.APISuccesses
		summary.TotalAPIFailures += item.APIFailures
		summary.Items = append(summary.Items, item)
	}
	if summary.TotalAPIRequests > 0 {
		summary.APISuccessRate = float64(summary.TotalAPISuccesses) / float64(summary.TotalAPIRequests)
		summary.APIFailureRate = float64(summary.TotalAPIFailures) / float64(summary.TotalAPIRequests)
	}
	return summary, nil
}

func (s *MemoryStore) CreateProjectDomain(_ context.Context, input domain.ProjectDomain) (domain.ProjectDomain, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.projectDomains {
		if (item.Status == "pending" || item.Status == "active") && item.ProjectID == input.ProjectID {
			return domain.ProjectDomain{}, fmt.Errorf("这个作品已经有待审核或已通过的独立网址申请")
		}
		if (item.Status == "pending" || item.Status == "active") && item.Subdomain == input.Subdomain {
			return domain.ProjectDomain{}, fmt.Errorf("这个独立网址已经被申请或使用")
		}
	}
	input.ID = fmt.Sprintf("domain_%d", time.Now().UnixNano())
	input.CreatedAt = time.Now().UTC()
	input.UpdatedAt = input.CreatedAt
	s.projectDomains[input.ID] = input
	return input, nil
}

func (s *MemoryStore) ListProjectDomains(_ context.Context, projectID string) ([]domain.ProjectDomain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []domain.ProjectDomain{}
	for _, item := range s.projectDomains {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *MemoryStore) ListAdminProjectDomains(_ context.Context, status string) ([]domain.ProjectDomain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []domain.ProjectDomain{}
	for _, item := range s.projectDomains {
		if status != "" && item.Status != status {
			continue
		}
		if project, ok := s.projects[item.ProjectID]; ok {
			item.ProjectName = project.Name
			item.ProjectPublicURL = project.PublicURL
		}
		if user, ok := s.users[item.OwnerUserID]; ok {
			item.OwnerEmail = user.Email
			item.Username = user.Username
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *MemoryStore) GetActiveProjectDomainAccessByDomain(_ context.Context, host string) (domain.ProjectDomain, domain.Project, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, item := range s.projectDomains {
		if item.Domain != host || item.Status != "active" {
			continue
		}
		project, ok := s.projects[item.ProjectID]
		if !ok {
			return domain.ProjectDomain{}, domain.Project{}, false, nil
		}
		item.ProjectName = project.Name
		item.ProjectPublicURL = project.PublicURL
		return item, project, true, nil
	}
	return domain.ProjectDomain{}, domain.Project{}, false, nil
}

func (s *MemoryStore) UpdateProjectDomainReview(_ context.Context, domainID, status, rejectReason, adminNote, reviewedBy string) (domain.ProjectDomain, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.projectDomains[domainID]
	if !ok {
		return domain.ProjectDomain{}, fmt.Errorf("找不到这个独立网址申请")
	}
	item.Status = status
	item.RejectReason = rejectReason
	item.AdminNote = adminNote
	item.ReviewedBy = reviewedBy
	item.ReviewedAt = time.Now().UTC()
	item.UpdatedAt = item.ReviewedAt
	s.projectDomains[domainID] = item
	if project, ok := s.projects[item.ProjectID]; ok {
		item.ProjectName = project.Name
		item.ProjectPublicURL = project.PublicURL
	}
	if user, ok := s.users[item.OwnerUserID]; ok {
		item.OwnerEmail = user.Email
		item.Username = user.Username
	}
	return item, nil
}

func (s *MemoryStore) CreateProjectDomainDeleteRequest(_ context.Context, input domain.ProjectDomainDeleteRequest) (domain.ProjectDomainDeleteRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.projectDomainDeleteRequests {
		if item.DomainID == input.DomainID && item.Status == "pending" {
			return domain.ProjectDomainDeleteRequest{}, fmt.Errorf("这个独立网址已经有待处理的删除申请")
		}
	}
	now := time.Now().UTC()
	input.ID = fmt.Sprintf("domain_delete_%d", now.UnixNano())
	if input.Status == "" {
		input.Status = "pending"
	}
	input.CreatedAt = now
	input.UpdatedAt = now
	s.projectDomainDeleteRequests[input.ID] = input
	if project, ok := s.projects[input.ProjectID]; ok {
		input.ProjectName = project.Name
		input.ProjectPublicURL = project.PublicURL
	}
	if user, ok := s.users[input.OwnerUserID]; ok {
		input.OwnerEmail = user.Email
		input.Username = user.Username
	}
	return input, nil
}

func (s *MemoryStore) ListProjectDomainDeleteRequests(_ context.Context, projectID string) ([]domain.ProjectDomainDeleteRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []domain.ProjectDomainDeleteRequest{}
	for _, item := range s.projectDomainDeleteRequests {
		if item.ProjectID != projectID {
			continue
		}
		if project, ok := s.projects[item.ProjectID]; ok {
			item.ProjectName = project.Name
			item.ProjectPublicURL = project.PublicURL
		}
		if user, ok := s.users[item.OwnerUserID]; ok {
			item.OwnerEmail = user.Email
			item.Username = user.Username
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (s *MemoryStore) ListAdminProjectDomainDeleteRequests(_ context.Context, status string) ([]domain.ProjectDomainDeleteRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []domain.ProjectDomainDeleteRequest{}
	for _, item := range s.projectDomainDeleteRequests {
		if status != "" && item.Status != status {
			continue
		}
		if project, ok := s.projects[item.ProjectID]; ok {
			item.ProjectName = project.Name
			item.ProjectPublicURL = project.PublicURL
		}
		if user, ok := s.users[item.OwnerUserID]; ok {
			item.OwnerEmail = user.Email
			item.Username = user.Username
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (s *MemoryStore) UpdateProjectDomainDeleteRequest(_ context.Context, requestID, status, adminNote, reviewedBy string) (domain.ProjectDomainDeleteRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.projectDomainDeleteRequests[requestID]
	if !ok {
		return domain.ProjectDomainDeleteRequest{}, fmt.Errorf("找不到这个独立网址删除申请")
	}
	now := time.Now().UTC()
	item.Status = status
	item.AdminNote = adminNote
	item.ReviewedBy = reviewedBy
	item.ReviewedAt = now
	item.UpdatedAt = now
	s.projectDomainDeleteRequests[requestID] = item
	if status == "completed" && item.DomainID != "" {
		if domainItem, ok := s.projectDomains[item.DomainID]; ok {
			domainItem.Status = "disabled"
			domainItem.AdminNote = adminNote
			domainItem.UpdatedAt = now
			s.projectDomains[item.DomainID] = domainItem
		}
	}
	if project, ok := s.projects[item.ProjectID]; ok {
		item.ProjectName = project.Name
		item.ProjectPublicURL = project.PublicURL
	}
	if user, ok := s.users[item.OwnerUserID]; ok {
		item.OwnerEmail = user.Email
		item.Username = user.Username
	}
	return item, nil
}

func (s *MemoryStore) CreateRepairRequest(_ context.Context, input domain.RepairRequest) (domain.RepairRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.repairRequests {
		if item.ProjectID == input.ProjectID && (item.Status == "pending" || item.Status == "processing" || item.Status == "need_info") {
			return domain.RepairRequest{}, fmt.Errorf("这个作品已经有待处理的修复申请")
		}
	}
	input.ID = fmt.Sprintf("repair_%d", time.Now().UnixNano())
	input.CreatedAt = time.Now().UTC()
	input.UpdatedAt = input.CreatedAt
	s.repairRequests[input.ID] = input
	return input, nil
}

func (s *MemoryStore) ListProjectRepairRequests(_ context.Context, projectID string) ([]domain.RepairRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []domain.RepairRequest{}
	for _, item := range s.repairRequests {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *MemoryStore) ListAdminRepairRequests(_ context.Context, status string) ([]domain.RepairRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []domain.RepairRequest{}
	for _, item := range s.repairRequests {
		if status != "" && item.Status != status {
			continue
		}
		if project, ok := s.projects[item.ProjectID]; ok {
			item.ProjectName = project.Name
			item.ProjectPublicURL = project.PublicURL
		}
		if user, ok := s.users[item.OwnerUserID]; ok {
			item.OwnerEmail = user.Email
			item.Username = user.Username
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *MemoryStore) UpdateRepairRequest(_ context.Context, requestID, status, adminReply, reviewedBy string) (domain.RepairRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.repairRequests[requestID]
	if !ok {
		return domain.RepairRequest{}, fmt.Errorf("找不到这个修复申请")
	}
	item.Status = status
	item.AdminReply = adminReply
	item.ReviewedBy = reviewedBy
	item.ReviewedAt = time.Now().UTC()
	item.UpdatedAt = item.ReviewedAt
	s.repairRequests[requestID] = item
	if project, ok := s.projects[item.ProjectID]; ok {
		item.ProjectName = project.Name
		item.ProjectPublicURL = project.PublicURL
	}
	if user, ok := s.users[item.OwnerUserID]; ok {
		item.OwnerEmail = user.Email
		item.Username = user.Username
	}
	return item, nil
}

func (s *MemoryStore) ReplyRepairRequest(_ context.Context, projectID, requestID, reply string) (domain.RepairRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.repairRequests[requestID]
	if !ok || item.ProjectID != projectID {
		return domain.RepairRequest{}, fmt.Errorf("找不到这个修复申请")
	}
	if item.Status != "need_info" {
		return domain.RepairRequest{}, fmt.Errorf("这个申请暂时不能补充信息")
	}
	if strings.TrimSpace(item.UserReply) != "" {
		return domain.RepairRequest{}, fmt.Errorf("你已经补充过一次信息")
	}
	now := time.Now().UTC()
	item.UserReply = reply
	item.UserRepliedAt = now
	item.Status = "processing"
	item.UpdatedAt = now
	s.repairRequests[requestID] = item
	return item, nil
}

func (s *MemoryStore) GetRepairRequest(_ context.Context, projectID, requestID string) (domain.RepairRequest, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.repairRequests[requestID]
	if !ok || item.ProjectID != projectID {
		return domain.RepairRequest{}, false, nil
	}
	return item, true, nil
}

func (s *MemoryStore) CreateRepairAIJob(_ context.Context, input domain.RepairAIJob) (domain.RepairAIJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	input.ID = fmt.Sprintf("repair_ai_%d", now.UnixNano())
	input.CreatedAt = now
	input.UpdatedAt = now
	if input.Status == "" {
		input.Status = "running"
	}
	if input.Round <= 0 {
		input.Round = 1
	}
	s.repairAIJobs[input.ID] = input
	return input, nil
}

func (s *MemoryStore) GetRepairAIJob(_ context.Context, jobID string) (domain.RepairAIJob, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.repairAIJobs[jobID]
	return item, ok, nil
}

func (s *MemoryStore) GetLatestRepairAIJob(_ context.Context, repairRequestID string) (domain.RepairAIJob, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var latest domain.RepairAIJob
	ok := false
	for _, item := range s.repairAIJobs {
		if item.RepairRequestID != repairRequestID {
			continue
		}
		if !ok || item.CreatedAt.After(latest.CreatedAt) {
			latest = item
			ok = true
		}
	}
	return latest, ok, nil
}

func (s *MemoryStore) ListRepairAIJobs(_ context.Context, repairRequestID string) ([]domain.RepairAIJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []domain.RepairAIJob{}
	for _, item := range s.repairAIJobs {
		if item.RepairRequestID == repairRequestID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Round != items[j].Round {
			return items[i].Round < items[j].Round
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items, nil
}

func (s *MemoryStore) GetActiveRepairAIJobByUser(_ context.Context, userID string) (domain.RepairAIJob, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.repairAIJobs {
		if item.OwnerUserID == userID && (item.Status == "queued" || item.Status == "running") {
			return item, true, nil
		}
	}
	return domain.RepairAIJob{}, false, nil
}

func (s *MemoryStore) CountRepairAIJobsForUserSince(_ context.Context, userID string, since time.Time) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, item := range s.repairAIJobs {
		if item.OwnerUserID == userID && !item.CreatedAt.Before(since) {
			count++
		}
	}
	return count, nil
}

func (s *MemoryStore) UpdateRepairAIJob(_ context.Context, job domain.RepairAIJob) (domain.RepairAIJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.repairAIJobs[job.ID]
	if !ok {
		return domain.RepairAIJob{}, fmt.Errorf("找不到 AI 圆桌")
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = current.CreatedAt
	}
	job.UpdatedAt = time.Now().UTC()
	s.repairAIJobs[job.ID] = job
	return job, nil
}

func (s *MemoryStore) CreateRepairAIMessage(_ context.Context, input domain.RepairAIMessage) (domain.RepairAIMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	input.ID = fmt.Sprintf("repair_ai_msg_%d", now.UnixNano())
	input.CreatedAt = now
	input.MessageSeq = len(s.repairAIMessages[input.JobID]) + 1
	s.repairAIMessages[input.JobID] = append(s.repairAIMessages[input.JobID], input)
	return input, nil
}

func (s *MemoryStore) ListRepairAIMessages(_ context.Context, jobID string) ([]domain.RepairAIMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := append([]domain.RepairAIMessage{}, s.repairAIMessages[jobID]...)
	return items, nil
}

func (s *MemoryStore) IsReservedSubdomain(_ context.Context, subdomain string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reservedDomains[subdomain], nil
}

func (s *MemoryStore) CreateTemplateSubmission(_ context.Context, input domain.TemplateSubmission) (domain.TemplateSubmission, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range s.templateSubmissions {
		if item.Slug == input.Slug {
			return domain.TemplateSubmission{}, fmt.Errorf("这个模板链接名已经被使用")
		}
	}
	now := time.Now().UTC()
	input.ID = fmt.Sprintf("tpl_%d", now.UnixNano())
	input.Status = "pending"
	input.CreatedAt = now
	input.UpdatedAt = now
	if user, ok := s.users[input.AuthorUserID]; ok {
		input.AuthorEmail = user.Email
		input.AuthorName = user.Username
		if input.AuthorName == "" {
			input.AuthorName = user.Email
		}
	}
	s.templateSubmissions[input.ID] = input
	return input, nil
}

func (s *MemoryStore) ListPublishedTemplateSubmissions(_ context.Context) ([]domain.TemplateSubmission, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []domain.TemplateSubmission{}
	for _, item := range s.templateSubmissions {
		if item.Status == "published" {
			items = append(items, s.decorateTemplateSubmissionLocked(item))
		}
	}
	sortTemplateSubmissions(items)
	return items, nil
}

func (s *MemoryStore) ListMyTemplateSubmissions(_ context.Context, userID string) ([]domain.TemplateSubmission, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []domain.TemplateSubmission{}
	for _, item := range s.templateSubmissions {
		if item.AuthorUserID == userID {
			items = append(items, s.decorateTemplateSubmissionLocked(item))
		}
	}
	sortTemplateSubmissions(items)
	return items, nil
}

func (s *MemoryStore) ListAdminTemplateSubmissions(_ context.Context, status string) ([]domain.TemplateSubmission, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := []domain.TemplateSubmission{}
	for _, item := range s.templateSubmissions {
		if status != "" && item.Status != status {
			continue
		}
		items = append(items, s.decorateTemplateSubmissionLocked(item))
	}
	sortTemplateSubmissions(items)
	return items, nil
}

func (s *MemoryStore) GetTemplateSubmission(_ context.Context, submissionID string) (domain.TemplateSubmission, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.templateSubmissions[submissionID]
	if !ok {
		return domain.TemplateSubmission{}, false, nil
	}
	return s.decorateTemplateSubmissionLocked(item), true, nil
}

func (s *MemoryStore) GetPublishedTemplateSubmissionByIDOrSlug(_ context.Context, idOrSlug string) (domain.TemplateSubmission, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	idOrSlug = strings.TrimPrefix(idOrSlug, "submission:")
	for _, item := range s.templateSubmissions {
		if item.Status == "published" && (item.ID == idOrSlug || item.Slug == idOrSlug) {
			return s.decorateTemplateSubmissionLocked(item), true, nil
		}
	}
	return domain.TemplateSubmission{}, false, nil
}

func (s *MemoryStore) UpdateTemplateSubmissionReview(_ context.Context, submissionID, status, adminNote, reviewedBy string) (domain.TemplateSubmission, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.templateSubmissions[submissionID]
	if !ok {
		return domain.TemplateSubmission{}, fmt.Errorf("找不到这个模板投稿")
	}
	now := time.Now().UTC()
	item.Status = status
	item.AdminNote = adminNote
	item.ReviewedBy = reviewedBy
	item.ReviewedAt = now
	item.UpdatedAt = now
	s.templateSubmissions[submissionID] = item
	return s.decorateTemplateSubmissionLocked(item), nil
}

func (s *MemoryStore) decorateTemplateSubmissionLocked(item domain.TemplateSubmission) domain.TemplateSubmission {
	if user, ok := s.users[item.AuthorUserID]; ok {
		item.AuthorEmail = user.Email
		item.AuthorName = user.Username
		if item.AuthorName == "" {
			item.AuthorName = user.Email
		}
	}
	return item
}

func sortTemplateSubmissions(items []domain.TemplateSubmission) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
}

func (s *MemoryStore) CreateUpgradeRequest(_ context.Context, input domain.UpgradeRequest) (domain.UpgradeRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	input.ID = fmt.Sprintf("upgrade_%d", time.Now().UnixNano())
	input.CreatedAt = time.Now().UTC()
	s.upgradeRequests[input.ID] = input
	return input, nil
}

func (s *MemoryStore) ListUpgradeRequests(_ context.Context) ([]domain.UpgradeRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.UpgradeRequest, 0, len(s.upgradeRequests))
	for _, item := range s.upgradeRequests {
		items = append(items, item)
	}
	return items, nil
}

func (s *MemoryStore) ListUserUpgradeRequests(_ context.Context, userID string) ([]domain.UpgradeRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.UpgradeRequest, 0)
	for _, item := range s.upgradeRequests {
		if item.UserID == userID {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *MemoryStore) UpdateUpgradeRequest(_ context.Context, requestID, status, adminNote, reviewedBy string) (domain.UpgradeRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.upgradeRequests[requestID]
	item.Status = status
	item.AdminNote = adminNote
	item.ReviewedBy = reviewedBy
	item.ReviewedAt = time.Now().UTC()
	s.upgradeRequests[requestID] = item
	return item, nil
}

func (s *MemoryStore) CreateRelease(_ context.Context, release domain.Release) (domain.Release, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if release.ID == "" {
		release.ID = fmt.Sprintf("rel_%d", time.Now().UnixNano())
	}
	if release.CreatedAt.IsZero() {
		release.CreatedAt = time.Now().UTC()
	}
	s.releases[release.ProjectID] = append(s.releases[release.ProjectID], release)
	project := s.projects[release.ProjectID]
	project.CurrentRelease = release.ID
	s.projects[release.ProjectID] = project
	return release, nil
}

func (s *MemoryStore) ListReleases(_ context.Context, projectID string) ([]domain.Release, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := append([]domain.Release{}, s.releases[projectID]...)
	if items == nil {
		return []domain.Release{}, nil
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (s *MemoryStore) SetCurrentRelease(_ context.Context, projectID, releaseID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	project := s.projects[projectID]
	project.CurrentRelease = releaseID
	s.projects[projectID] = project
	return nil
}

func shouldReplaceUsername(user domain.User, candidate string) bool {
	if candidate == "" {
		return false
	}
	return user.Username == "" || user.Username == strings.Split(user.Email, "@")[0]
}
