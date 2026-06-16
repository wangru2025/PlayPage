package storage

import (
	"context"
	"encoding/json"
	"fmt"
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

type memorySession struct {
	UserID    string
	Token     string
	ExpiresAt time.Time
}

type MemoryStore struct {
	publicBase       string
	mu               sync.RWMutex
	users            map[string]domain.User
	usersByMail      map[string]string
	authCodes        map[string]memoryAuthCode
	sessions         map[string]memorySession
	projects         map[string]domain.Project
	projectUser      map[string]string
	collections      map[string][]domain.Collection
	records          map[string][]domain.Record
	releases         map[string][]domain.Release
	publicKeys       map[string]string
	usage            map[string]domain.ProjectUsage
	dailyStats       map[string]domain.ProjectDailyStats
	upgradeRequests  map[string]domain.UpgradeRequest
	projectDomains   map[string]domain.ProjectDomain
	repairRequests   map[string]domain.RepairRequest
	repairAIJobs     map[string]domain.RepairAIJob
	repairAIMessages map[string][]domain.RepairAIMessage
	reservedDomains  map[string]bool
}

func NewMemoryStore(publicBase string) *MemoryStore {
	return &MemoryStore{
		publicBase:       publicBase,
		users:            map[string]domain.User{},
		usersByMail:      map[string]string{},
		authCodes:        map[string]memoryAuthCode{},
		sessions:         map[string]memorySession{},
		projects:         map[string]domain.Project{},
		projectUser:      map[string]string{},
		collections:      map[string][]domain.Collection{},
		records:          map[string][]domain.Record{},
		releases:         map[string][]domain.Release{},
		publicKeys:       map[string]string{},
		usage:            map[string]domain.ProjectUsage{},
		dailyStats:       map[string]domain.ProjectDailyStats{},
		upgradeRequests:  map[string]domain.UpgradeRequest{},
		projectDomains:   map[string]domain.ProjectDomain{},
		repairRequests:   map[string]domain.RepairRequest{},
		repairAIJobs:     map[string]domain.RepairAIJob{},
		repairAIMessages: map[string][]domain.RepairAIMessage{},
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
			items = append(items, project)
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
		if project.Visibility == "public" && project.CurrentRelease != "" {
			items = append(items, project)
		}
	}

	return items, nil
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

func (s *MemoryStore) GetProject(_ context.Context, userID, projectID string) (domain.Project, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	project, ok := s.projects[projectID]
	if !ok || s.projectUser[projectID] != userID {
		return domain.Project{}, false, nil
	}
	return project, true, nil
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
		return domain.Record{}, fmt.Errorf("\u627e\u4e0d\u5230\u8fd9\u4e2a\u96c6\u5408")
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
		return domain.RepairAIJob{}, fmt.Errorf("????? AI ??")
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
	s.releases[release.ProjectID] = append(s.releases[release.ProjectID], release)
	project := s.projects[release.ProjectID]
	project.CurrentRelease = release.ID
	s.projects[release.ProjectID] = project
	return release, nil
}

func (s *MemoryStore) ListReleases(_ context.Context, projectID string) ([]domain.Release, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := s.releases[projectID]
	if items == nil {
		return []domain.Release{}, nil
	}
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
