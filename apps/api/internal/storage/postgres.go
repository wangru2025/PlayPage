package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"ai-static-host/api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool       *pgxpool.Pool
	publicBase string
}

type rowScanner interface {
	Scan(dest ...any) error
}

func projectSelectSQL(alias string) string {
	if alias == "" {
		alias = "p"
	}
	return alias + `.id::text, ` + alias + `.username, ` + alias + `.slug, ` + alias + `.name, ` +
		alias + `.interactive, ` + alias + `.analytics_enabled, ` + alias + `.show_on_profile, ` + alias + `.allow_forks, ` + alias + `.visibility, coalesce(` + alias + `.current_release_id::text, ''),
		coalesce(` + alias + `.forked_from_project_id::text, ''), coalesce(` + alias + `.forked_from_release_id::text, ''), coalesce(` + alias + `.forked_from_user_id::text, ''),
		coalesce(src.username, ''), coalesce(src.name, ''), coalesce(src.slug, ''),
		coalesce(` + alias + `.forked_from_snapshot_owner, ''), coalesce(` + alias + `.forked_from_snapshot_name, ''), coalesce(` + alias + `.forked_from_snapshot_url, ''),
		(select count(*) from project_favorites pf where pf.project_id = ` + alias + `.id)::int,
		(select count(*) from projects fp where fp.forked_from_project_id = ` + alias + `.id)::int,
		exists(select 1 from projects original where original.id = ` + alias + `.forked_from_project_id and original.current_release_id is not null and original.allow_forks = true),
		` + alias + `.created_at`
}

func projectSourceJoinSQL(alias string) string {
	if alias == "" {
		alias = "p"
	}
	return ` left join projects src on src.id = ` + alias + `.forked_from_project_id `
}

func (s *PostgresStore) scanProject(row rowScanner) (domain.Project, error) {
	var project domain.Project
	var sourceUsername, sourceName, sourceSlug string
	err := row.Scan(
		&project.ID,
		&project.Username,
		&project.Slug,
		&project.Name,
		&project.Interactive,
		&project.AnalyticsEnabled,
		&project.ShowOnProfile,
		&project.AllowForks,
		&project.Visibility,
		&project.CurrentRelease,
		&project.ForkedFromProjectID,
		&project.ForkedFromReleaseID,
		&project.ForkedFromUserID,
		&sourceUsername,
		&sourceName,
		&sourceSlug,
		&project.ForkedFromSnapshotOwner,
		&project.ForkedFromSnapshotName,
		&project.ForkedFromProjectURL,
		&project.FavoritesCount,
		&project.ForksCount,
		&project.CanSubmitProposal,
		&project.CreatedAt,
	)
	if err != nil {
		return domain.Project{}, err
	}
	project.PublicURL = buildPublicURL(s.publicBase, project.Username, project.Slug)
	if sourceUsername != "" && sourceSlug != "" {
		project.ForkedFromUsername = sourceUsername
		project.ForkedFromProjectName = sourceName
		project.ForkedFromProjectURL = buildPublicURL(s.publicBase, sourceUsername, sourceSlug)
	} else {
		project.ForkedFromUsername = project.ForkedFromSnapshotOwner
		project.ForkedFromProjectName = project.ForkedFromSnapshotName
	}
	return project, nil
}

func NewPostgresStore(ctx context.Context, databaseURL, publicBase string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &PostgresStore{
		pool:       pool,
		publicBase: publicBase,
	}, nil
}

func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (domain.User, bool, error) {
	var user domain.User
	err := s.pool.QueryRow(ctx, `
		select id::text, coalesce(username, ''), email, status, role, plan_code, created_at
		from users
		where email = $1
	`, email).Scan(&user.ID, &user.Username, &user.Email, &user.Status, &user.Role, &user.PlanCode, &user.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.User{}, false, nil
		}
		return domain.User{}, false, err
	}
	return user, true, nil
}

func (s *PostgresStore) GetUserBySessionToken(ctx context.Context, token string) (domain.User, bool, error) {
	var user domain.User
	err := s.pool.QueryRow(ctx, `
		select u.id::text, coalesce(u.username, ''), u.email, u.status, u.role, u.plan_code, u.created_at
		from sessions sess
		join users u on u.id = sess.user_id
		where sess.token = $1 and sess.expires_at > now()
	`, token).Scan(&user.ID, &user.Username, &user.Email, &user.Status, &user.Role, &user.PlanCode, &user.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.User{}, false, nil
		}
		return domain.User{}, false, err
	}
	return user, true, nil
}

func (s *PostgresStore) GetUserByID(ctx context.Context, userID string) (domain.User, bool, error) {
	var user domain.User
	err := s.pool.QueryRow(ctx, `
		select id::text, coalesce(username, ''), email, status, role, plan_code, created_at
		from users
		where id::text = $1
	`, userID).Scan(&user.ID, &user.Username, &user.Email, &user.Status, &user.Role, &user.PlanCode, &user.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.User{}, false, nil
		}
		return domain.User{}, false, err
	}
	return user, true, nil
}

func (s *PostgresStore) UpdateUserUsername(ctx context.Context, userID, username string) (domain.User, error) {
	var user domain.User
	err := s.pool.QueryRow(ctx, `
		update users
		set username = nullif($2, '')
		where id::text = $1
		returning id::text, coalesce(username, ''), email, status, role, plan_code, created_at
	`, userID, username).Scan(&user.ID, &user.Username, &user.Email, &user.Status, &user.Role, &user.PlanCode, &user.CreatedAt)
	if err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (s *PostgresStore) UpdateUserRoleAndPlan(ctx context.Context, userID, role, planCode string) (domain.User, error) {
	var user domain.User
	err := s.pool.QueryRow(ctx, `
		update users
		set role = $2, plan_code = $3
		where id::text = $1
		returning id::text, coalesce(username, ''), email, status, role, plan_code, created_at
	`, userID, role, planCode).Scan(&user.ID, &user.Username, &user.Email, &user.Status, &user.Role, &user.PlanCode, &user.CreatedAt)
	if err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func scanAppBuildSettings(row rowScanner) (domain.AppBuildSettings, error) {
	var item domain.AppBuildSettings
	err := row.Scan(
		&item.ID, &item.UserID, &item.ProjectID, &item.AppName,
		&item.AndroidEnabled, &item.WindowsEnabled, &item.AutoUpdate,
		&item.AndroidPackageName, &item.WindowsPackageName,
		&item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func scanAppBuildJob(row rowScanner) (domain.AppBuildJob, error) {
	var item domain.AppBuildJob
	err := row.Scan(
		&item.ID, &item.UserID, &item.ProjectID, &item.ReleaseID, &item.Platform,
		&item.AppName, &item.PackageName, &item.VersionCode, &item.VersionName, &item.AutoUpdate,
		&item.Status, &item.ArtifactPath, &item.ArtifactSHA256, &item.ArtifactSize,
		&item.GitHubRunID, &item.ErrorMessage, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func (s *PostgresStore) SyncProjectUsernames(ctx context.Context, userID, username string) error {
	_, err := s.pool.Exec(ctx, `
		update projects
		set username = $2
		where owner_user_id::text = $1
	`, userID, username)
	return err
}

func (s *PostgresStore) CreateOrRefreshAuthCode(ctx context.Context, input domain.AuthCodeRequestInput, code string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		insert into auth_codes (email, username, code, purpose, expires_at)
		values ($1, $2, $3, 'login', $4)
	`, input.Email, input.Username, code, expiresAt)
	return err
}

func (s *PostgresStore) ConsumeAuthCode(ctx context.Context, input domain.AuthCodeVerifyInput, now time.Time) (domain.User, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, false, err
	}
	defer tx.Rollback(ctx)

	var email string
	var username string
	err = tx.QueryRow(ctx, `
		select email, username
		from auth_codes
		where email = $1 and code = $2 and consumed_at is null and expires_at > $3
		order by created_at desc
		limit 1
	`, input.Email, input.Code, now).Scan(&email, &username)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.User{}, false, nil
		}
		return domain.User{}, false, err
	}

	_, err = tx.Exec(ctx, `
		update auth_codes
		set consumed_at = $3
		where email = $1 and code = $2 and consumed_at is null
	`, input.Email, input.Code, now)
	if err != nil {
		return domain.User{}, false, err
	}

	var user domain.User
	err = tx.QueryRow(ctx, `
		insert into users (email, username, status, role, plan_code)
		values ($1, nullif($2, ''), 'active', 'user', 'free')
		on conflict (email) do update
		set username = case
			when coalesce(users.username, '') = '' then excluded.username
			when users.username = split_part(users.email, '@', 1) and nullif(excluded.username, '') is not null then excluded.username
			else users.username
		end
		returning id::text, coalesce(username, ''), email, status, role, plan_code, created_at
	`, email, username).Scan(&user.ID, &user.Username, &user.Email, &user.Status, &user.Role, &user.PlanCode, &user.CreatedAt)
	if err != nil {
		return domain.User{}, false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, false, err
	}

	return user, true, nil
}

func (s *PostgresStore) CreateProjectEmailCode(ctx context.Context, projectID, ownerUserID, email, purpose, code string, expiresAt, now time.Time, dailyLimit int) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var recentID string
	err = tx.QueryRow(ctx, `
		select id::text
		from project_email_codes
		where project_id::text = $1 and email = $2 and purpose = $3 and created_at > $4
		order by created_at desc
		limit 1
	`, projectID, email, purpose, now.Add(-60*time.Second)).Scan(&recentID)
	if err == nil {
		return 0, fmt.Errorf("同一个邮箱 60 秒内只能发送一次验证码")
	}
	if err != pgx.ErrNoRows {
		return 0, err
	}

	day := now.Format("2006-01-02")
	var sentCount int
	err = tx.QueryRow(ctx, `
		insert into project_email_quota_daily (owner_user_id, day, sent_count)
		values ($1::uuid, $2::date, 1)
		on conflict (owner_user_id, day)
		do update set sent_count = project_email_quota_daily.sent_count + 1
		returning sent_count
	`, ownerUserID, day).Scan(&sentCount)
	if err != nil {
		return 0, err
	}
	if dailyLimit > 0 && sentCount > dailyLimit {
		_, _ = tx.Exec(ctx, `
			update project_email_quota_daily
			set sent_count = sent_count - 1
			where owner_user_id::text = $1 and day = $2::date and sent_count > 0
		`, ownerUserID, day)
		return sentCount - 1, fmt.Errorf("这个账号今天的作品验证码邮件额度已经用完了")
	}

	_, err = tx.Exec(ctx, `
		insert into project_email_codes (project_id, owner_user_id, email, purpose, code, expires_at, created_at)
		values ($1::uuid, $2::uuid, $3, $4, $5, $6, $7)
	`, projectID, ownerUserID, email, purpose, code, expiresAt, now)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return sentCount, nil
}

func (s *PostgresStore) ConsumeProjectEmailCode(ctx context.Context, projectID, email, purpose, code string, now time.Time) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		update project_email_codes
		set consumed_at = $6
		where id = (
			select id
			from project_email_codes
			where project_id::text = $1
			  and email = $2
			  and purpose = $3
			  and code = $4
			  and consumed_at is null
			  and expires_at > $5
			order by created_at desc
			limit 1
		)
	`, projectID, email, purpose, code, now, now)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *PostgresStore) CreateSession(ctx context.Context, userID, token string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		insert into sessions (user_id, token, expires_at)
		values ($1::uuid, $2, $3)
	`, userID, token, expiresAt)
	return err
}

func (s *PostgresStore) DeleteSession(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `delete from sessions where token = $1`, token)
	return err
}

func (s *PostgresStore) ListAdminUsers(ctx context.Context) ([]domain.AdminUserSummary, error) {
	rows, err := s.pool.Query(ctx, `
		select
			u.id::text,
			u.email,
			coalesce(u.username, ''),
			u.status,
			u.role,
			u.plan_code,
			coalesce(count(p.id), 0)::int,
			u.created_at
		from users u
		left join projects p on p.owner_user_id = u.id
		group by u.id, u.email, u.username, u.status, u.role, u.plan_code, u.created_at
		order by u.created_at desc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.AdminUserSummary{}
	for rows.Next() {
		var item domain.AdminUserSummary
		if err := rows.Scan(&item.ID, &item.Email, &item.Username, &item.Status, &item.Role, &item.PlanCode, &item.ProjectCount, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) ListProjects(ctx context.Context, userID string) ([]domain.Project, error) {
	rows, err := s.pool.Query(ctx, `
		select `+projectSelectSQL("p")+`
		from projects p
		`+projectSourceJoinSQL("p")+`
		where p.owner_user_id::text = $1
		order by p.created_at desc
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.Project{}
	for rows.Next() {
		item, err := s.scanProject(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (s *PostgresStore) ListPublicProjects(ctx context.Context) ([]domain.Project, error) {
	rows, err := s.pool.Query(ctx, `
		select `+projectSelectSQL("p")+`
		from projects p
		`+projectSourceJoinSQL("p")+`
		where p.visibility = 'public' and p.current_release_id is not null
		order by p.created_at desc
		limit 60
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.Project{}
	for rows.Next() {
		item, err := s.scanProject(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (s *PostgresStore) GetPublicProject(ctx context.Context, projectID string) (domain.Project, bool, error) {
	project, err := s.scanProject(s.pool.QueryRow(ctx, `
		select `+projectSelectSQL("p")+`
		from projects p
		`+projectSourceJoinSQL("p")+`
		where p.id::text = $1 and p.visibility = 'public' and p.current_release_id is not null
	`, projectID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Project{}, false, nil
		}
		return domain.Project{}, false, err
	}
	return project, true, nil
}

func (s *PostgresStore) GetAuthorProfile(ctx context.Context, username string) (domain.AuthorProfile, bool, error) {
	var profile domain.AuthorProfile
	var userID string
	err := s.pool.QueryRow(ctx, `
		select id::text, coalesce(username, ''), created_at
		from users
		where username = $1
	`, username).Scan(&userID, &profile.Username, &profile.JoinedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.AuthorProfile{}, false, nil
		}
		return domain.AuthorProfile{}, false, err
	}
	profile.DisplayName = profile.Username
	if err := s.pool.QueryRow(ctx, `
		select
			(select count(*) from author_follows where target_user_id::text = $1)::int,
			(select count(*) from author_follows where follower_user_id::text = $1)::int
	`, userID).Scan(&profile.FollowersCount, &profile.FollowingCount); err != nil {
		return domain.AuthorProfile{}, false, err
	}
	rows, err := s.pool.Query(ctx, `
		select `+projectSelectSQL("p")+`
		from projects p
		`+projectSourceJoinSQL("p")+`
		where p.owner_user_id::text = $1 and p.current_release_id is not null and p.show_on_profile = true
		order by p.created_at desc
	`, userID)
	if err != nil {
		return domain.AuthorProfile{}, false, err
	}
	defer rows.Close()
	for rows.Next() {
		project, err := s.scanProject(rows)
		if err != nil {
			return domain.AuthorProfile{}, false, err
		}
		profile.Projects = append(profile.Projects, project)
		profile.ProjectCount++
		profile.FavoritesCount += project.FavoritesCount
		profile.ForksCount += project.ForksCount
	}
	return profile, true, rows.Err()
}

func (s *PostgresStore) IsFollowingAuthor(ctx context.Context, followerUserID, targetUsername string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		select exists(
			select 1
			from author_follows af
			join users u on u.id = af.target_user_id
			where af.follower_user_id::text = $1 and u.username = $2
		)
	`, followerUserID, targetUsername).Scan(&exists)
	return exists, err
}

func (s *PostgresStore) FollowAuthor(ctx context.Context, followerUserID, targetUsername string) error {
	tag, err := s.pool.Exec(ctx, `
		insert into author_follows (follower_user_id, target_user_id)
		select $1::uuid, u.id
		from users u
		where u.username = $2 and u.id <> $1::uuid
		on conflict (follower_user_id, target_user_id) do nothing
	`, followerUserID, targetUsername)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("没有找到可关注的作者，或不能关注自己")
	}
	return nil
}

func (s *PostgresStore) UnfollowAuthor(ctx context.Context, followerUserID, targetUsername string) error {
	_, err := s.pool.Exec(ctx, `
		delete from author_follows af
		using users u
		where af.target_user_id = u.id and af.follower_user_id::text = $1 and u.username = $2
	`, followerUserID, targetUsername)
	return err
}

func (s *PostgresStore) ListFollowedAuthors(ctx context.Context, followerUserID string) ([]domain.AuthorSummary, error) {
	rows, err := s.pool.Query(ctx, `
		select u.id::text, coalesce(u.username, ''), coalesce(u.username, ''), u.created_at,
			(select count(*) from projects p where p.owner_user_id = u.id and p.visibility = 'public' and p.current_release_id is not null)::int,
			(select count(*) from author_follows af2 where af2.target_user_id = u.id)::int
		from author_follows af
		join users u on u.id = af.target_user_id
		where af.follower_user_id::text = $1
		order by af.created_at desc
	`, followerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.AuthorSummary{}
	for rows.Next() {
		var item domain.AuthorSummary
		if err := rows.Scan(&item.UserID, &item.Username, &item.DisplayName, &item.JoinedAt, &item.ProjectCount, &item.FollowersCount); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) ListAuthorFollowerRecipients(ctx context.Context, authorUsername string) ([]domain.NotificationRecipient, error) {
	rows, err := s.pool.Query(ctx, `
		select follower.email, coalesce(follower.username, '')
		from author_follows af
		join users author on author.id = af.target_user_id
		join users follower on follower.id = af.follower_user_id
		where author.username = $1 and follower.status = 'active'
		order by af.created_at desc
	`, authorUsername)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.NotificationRecipient{}
	for rows.Next() {
		var item domain.NotificationRecipient
		if err := rows.Scan(&item.Email, &item.Username); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) TryCreateProjectNotificationEvent(ctx context.Context, projectID, eventKey string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		insert into project_notification_events (project_id, event_key)
		values ($1::uuid, $2)
		on conflict (project_id, event_key) do nothing
	`, projectID, eventKey)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *PostgresStore) IsProjectFavorited(ctx context.Context, userID, projectID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `select exists(select 1 from project_favorites where user_id::text=$1 and project_id::text=$2)`, userID, projectID).Scan(&exists)
	return exists, err
}

func (s *PostgresStore) AddProjectFavorite(ctx context.Context, userID, projectID string) error {
	_, err := s.pool.Exec(ctx, `
		insert into project_favorites (user_id, project_id)
		values ($1::uuid, $2::uuid)
		on conflict (user_id, project_id) do nothing
	`, userID, projectID)
	return err
}

func (s *PostgresStore) RemoveProjectFavorite(ctx context.Context, userID, projectID string) error {
	_, err := s.pool.Exec(ctx, `delete from project_favorites where user_id::text=$1 and project_id::text=$2`, userID, projectID)
	return err
}

func (s *PostgresStore) ListUserFavoriteProjects(ctx context.Context, userID string) ([]domain.Project, error) {
	rows, err := s.pool.Query(ctx, `
		select `+projectSelectSQL("p")+`
		from project_favorites pf
		join projects p on p.id = pf.project_id
		`+projectSourceJoinSQL("p")+`
		where pf.user_id::text = $1 and p.current_release_id is not null and p.show_on_profile = true
		order by pf.created_at desc
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Project{}
	for rows.Next() {
		project, err := s.scanProject(rows)
		if err != nil {
			return nil, err
		}
		project.FavoritedByMe = true
		items = append(items, project)
	}
	return items, rows.Err()
}

func (s *PostgresStore) ListProjectDiscussions(ctx context.Context, projectID, status, searchQuery string) ([]domain.ProjectDiscussion, error) {
	query := `
		select d.id::text, d.project_id::text, p.name, p.username, p.slug,
			d.author_user_id::text, coalesce(author.username, ''), author.email,
			d.title, d.body, d.status,
			(select count(*) from project_discussion_comments c where c.discussion_id = d.id)::int,
			d.last_commented_at, coalesce(d.closed_by_user_id::text, ''), coalesce(closer.username, ''),
			coalesce(d.closed_at, '0001-01-01T00:00:00Z'::timestamptz), d.created_at, d.updated_at
		from project_discussions d
		join projects p on p.id = d.project_id
		join users author on author.id = d.author_user_id
		left join users closer on closer.id = d.closed_by_user_id
		where d.project_id::text = $1`
	args := []any{projectID}
	if status != "" {
		args = append(args, status)
		query += fmt.Sprintf(" and d.status = $%d", len(args))
	}
	searchQuery = strings.TrimSpace(searchQuery)
	if searchQuery != "" {
		args = append(args, "%"+searchQuery+"%")
		query += fmt.Sprintf(" and (d.title ilike $%d or d.body ilike $%d or coalesce(author.username, '') ilike $%d)", len(args), len(args), len(args))
	}
	query += " order by d.last_commented_at desc"
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.ProjectDiscussion{}
	for rows.Next() {
		item, err := s.scanProjectDiscussion(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) GetProjectDiscussion(ctx context.Context, discussionID string) (domain.ProjectDiscussion, bool, error) {
	item, err := s.scanProjectDiscussion(s.pool.QueryRow(ctx, `
		select d.id::text, d.project_id::text, p.name, p.username, p.slug,
			d.author_user_id::text, coalesce(author.username, ''), author.email,
			d.title, d.body, d.status,
			(select count(*) from project_discussion_comments c where c.discussion_id = d.id)::int,
			d.last_commented_at, coalesce(d.closed_by_user_id::text, ''), coalesce(closer.username, ''),
			coalesce(d.closed_at, '0001-01-01T00:00:00Z'::timestamptz), d.created_at, d.updated_at
		from project_discussions d
		join projects p on p.id = d.project_id
		join users author on author.id = d.author_user_id
		left join users closer on closer.id = d.closed_by_user_id
		where d.id::text = $1
	`, discussionID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.ProjectDiscussion{}, false, nil
		}
		return domain.ProjectDiscussion{}, false, err
	}
	return item, true, nil
}

func (s *PostgresStore) scanProjectDiscussion(row rowScanner) (domain.ProjectDiscussion, error) {
	var item domain.ProjectDiscussion
	var projectUsername, projectSlug string
	if err := row.Scan(
		&item.ID, &item.ProjectID, &item.ProjectName, &projectUsername, &projectSlug,
		&item.AuthorUserID, &item.AuthorUsername, &item.AuthorEmail,
		&item.Title, &item.Body, &item.Status, &item.CommentsCount, &item.LastCommentedAt,
		&item.ClosedByUserID, &item.ClosedByUsername, &item.ClosedAt, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return domain.ProjectDiscussion{}, err
	}
	item.ProjectURL = buildPublicURL(s.publicBase, projectUsername, projectSlug)
	return item, nil
}

func (s *PostgresStore) CreateProjectDiscussion(ctx context.Context, projectID, authorUserID string, input domain.ProjectDiscussionCreateInput) (domain.ProjectDiscussion, error) {
	item, err := s.scanProjectDiscussion(s.pool.QueryRow(ctx, `
		with created as (
			insert into project_discussions (project_id, author_user_id, title, body)
			values ($1::uuid, $2::uuid, $3, $4)
			returning *
		)
		select d.id::text, d.project_id::text, p.name, p.username, p.slug,
			d.author_user_id::text, coalesce(author.username, ''), author.email,
			d.title, d.body, d.status, 0::int,
			d.last_commented_at, coalesce(d.closed_by_user_id::text, ''), ''::text,
			coalesce(d.closed_at, '0001-01-01T00:00:00Z'::timestamptz), d.created_at, d.updated_at
		from created d
		join projects p on p.id = d.project_id
		join users author on author.id = d.author_user_id
	`, projectID, authorUserID, input.Title, input.Body))
	return item, err
}

func (s *PostgresStore) UpdateProjectDiscussionStatus(ctx context.Context, discussionID, status, operatorUserID string) (domain.ProjectDiscussion, bool, error) {
	item, err := s.scanProjectDiscussion(s.pool.QueryRow(ctx, `
		with updated as (
			update project_discussions
			set status = $2,
				closed_by_user_id = case when $2 = 'closed' then $3::uuid else null end,
				closed_at = case when $2 = 'closed' then now() else null end,
				updated_at = now()
			where id::text = $1
			returning *
		)
		select d.id::text, d.project_id::text, p.name, p.username, p.slug,
			d.author_user_id::text, coalesce(author.username, ''), author.email,
			d.title, d.body, d.status,
			(select count(*) from project_discussion_comments c where c.discussion_id = d.id)::int,
			d.last_commented_at, coalesce(d.closed_by_user_id::text, ''), coalesce(closer.username, ''),
			coalesce(d.closed_at, '0001-01-01T00:00:00Z'::timestamptz), d.created_at, d.updated_at
		from updated d
		join projects p on p.id = d.project_id
		join users author on author.id = d.author_user_id
		left join users closer on closer.id = d.closed_by_user_id
	`, discussionID, status, operatorUserID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.ProjectDiscussion{}, false, nil
		}
		return domain.ProjectDiscussion{}, false, err
	}
	return item, true, nil
}

func (s *PostgresStore) ListProjectDiscussionComments(ctx context.Context, discussionID string) ([]domain.ProjectDiscussionComment, error) {
	rows, err := s.pool.Query(ctx, `
		select c.id::text, c.discussion_id::text, c.author_user_id::text, coalesce(u.username, ''), u.email, c.body, c.created_at, c.updated_at
		from project_discussion_comments c
		join users u on u.id = c.author_user_id
		where c.discussion_id::text = $1
		order by c.created_at asc
	`, discussionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.ProjectDiscussionComment{}
	for rows.Next() {
		var item domain.ProjectDiscussionComment
		if err := rows.Scan(&item.ID, &item.DiscussionID, &item.AuthorUserID, &item.AuthorUsername, &item.AuthorEmail, &item.Body, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateProjectDiscussionComment(ctx context.Context, discussionID, authorUserID string, input domain.ProjectDiscussionCommentCreateInput) (domain.ProjectDiscussionComment, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.ProjectDiscussionComment{}, err
	}
	defer tx.Rollback(ctx)
	var item domain.ProjectDiscussionComment
	err = tx.QueryRow(ctx, `
		insert into project_discussion_comments (discussion_id, author_user_id, body)
		values ($1::uuid, $2::uuid, $3)
		returning id::text, discussion_id::text, author_user_id::text, body, created_at, updated_at
	`, discussionID, authorUserID, input.Body).Scan(&item.ID, &item.DiscussionID, &item.AuthorUserID, &item.Body, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return domain.ProjectDiscussionComment{}, err
	}
	_, err = tx.Exec(ctx, `update project_discussions set last_commented_at = now(), updated_at = now() where id::text = $1`, discussionID)
	if err != nil {
		return domain.ProjectDiscussionComment{}, err
	}
	var username, email string
	if err := tx.QueryRow(ctx, `select coalesce(username, ''), email from users where id::text = $1`, authorUserID).Scan(&username, &email); err != nil {
		return domain.ProjectDiscussionComment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.ProjectDiscussionComment{}, err
	}
	item.AuthorUsername = username
	item.AuthorEmail = email
	return item, nil
}

func (s *PostgresStore) scanProjectProposal(row rowScanner) (domain.ProjectProposal, error) {
	var item domain.ProjectProposal
	var sourceUsername, sourceSlug, targetUsername, targetSlug string
	err := row.Scan(
		&item.ID, &item.SourceProjectID, &item.SourceProjectName, &sourceUsername, &sourceSlug,
		&item.TargetProjectID, &item.TargetProjectName, &targetUsername, &targetSlug,
		&item.AuthorUserID, &item.AuthorUsername, &item.AuthorEmail,
		&item.TargetOwnerUserID, &item.TargetOwnerUsername, &item.TargetOwnerEmail,
		&item.Title, &item.Body, &item.Status, &item.SourceReleaseID, &item.MergedReleaseID,
		&item.ReviewNote, &item.ReviewedBy, &item.ReviewedAt, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return domain.ProjectProposal{}, err
	}
	item.SourceProjectURL = buildPublicURL(s.publicBase, sourceUsername, sourceSlug)
	item.TargetProjectURL = buildPublicURL(s.publicBase, targetUsername, targetSlug)
	return item, nil
}

const projectProposalSelectSQL = `
	select pr.id::text,
		pr.source_project_id::text, coalesce(sp.name, ''), coalesce(sp.username, ''), coalesce(sp.slug, ''),
		pr.target_project_id::text, coalesce(tp.name, ''), coalesce(tp.username, ''), coalesce(tp.slug, ''),
		pr.author_user_id::text, coalesce(author.username, ''), coalesce(author.email, ''),
		pr.target_owner_user_id::text, coalesce(owner.username, ''), coalesce(owner.email, ''),
		pr.title, pr.body, pr.status, pr.source_release_id::text, coalesce(pr.merged_release_id::text, ''),
		pr.review_note, coalesce(pr.reviewed_by_user_id::text, ''), coalesce(pr.reviewed_at, '0001-01-01T00:00:00Z'::timestamptz),
		pr.created_at, pr.updated_at
	from project_proposals pr
	join projects sp on sp.id = pr.source_project_id
	join projects tp on tp.id = pr.target_project_id
	join users author on author.id = pr.author_user_id
	join users owner on owner.id = pr.target_owner_user_id`

func (s *PostgresStore) ListProjectProposalsForSource(ctx context.Context, sourceProjectID string) ([]domain.ProjectProposal, error) {
	rows, err := s.pool.Query(ctx, projectProposalSelectSQL+` where pr.source_project_id::text = $1 order by pr.created_at desc`, sourceProjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.ProjectProposal{}
	for rows.Next() {
		item, err := s.scanProjectProposal(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) ListProjectProposalsForTarget(ctx context.Context, targetProjectID, status string) ([]domain.ProjectProposal, error) {
	query := projectProposalSelectSQL + ` where pr.target_project_id::text = $1`
	args := []any{targetProjectID}
	if strings.TrimSpace(status) != "" {
		args = append(args, status)
		query += fmt.Sprintf(" and pr.status = $%d", len(args))
	}
	query += ` order by pr.created_at desc`
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.ProjectProposal{}
	for rows.Next() {
		item, err := s.scanProjectProposal(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) ListUserProjectProposals(ctx context.Context, userID string) ([]domain.ProjectProposal, error) {
	rows, err := s.pool.Query(ctx, projectProposalSelectSQL+` where pr.author_user_id::text = $1 or pr.target_owner_user_id::text = $1 order by pr.created_at desc`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.ProjectProposal{}
	for rows.Next() {
		item, err := s.scanProjectProposal(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) GetProjectProposal(ctx context.Context, proposalID string) (domain.ProjectProposal, bool, error) {
	item, err := s.scanProjectProposal(s.pool.QueryRow(ctx, projectProposalSelectSQL+` where pr.id::text = $1`, proposalID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.ProjectProposal{}, false, nil
		}
		return domain.ProjectProposal{}, false, err
	}
	return item, true, nil
}

func (s *PostgresStore) CreateProjectProposal(ctx context.Context, input domain.ProjectProposal) (domain.ProjectProposal, error) {
	item, err := s.scanProjectProposal(s.pool.QueryRow(ctx, `
		with created as (
			insert into project_proposals (source_project_id, target_project_id, author_user_id, target_owner_user_id, title, body, status, source_release_id)
			values ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, 'open', $7::uuid)
			returning *
		)`+strings.Replace(projectProposalSelectSQL, "from project_proposals pr", "from created pr", 1),
		input.SourceProjectID, input.TargetProjectID, input.AuthorUserID, input.TargetOwnerUserID, input.Title, input.Body, input.SourceReleaseID))
	return item, err
}

func (s *PostgresStore) ReviewProjectProposal(ctx context.Context, proposalID, status, note, reviewedByUserID, mergedReleaseID string) (domain.ProjectProposal, bool, error) {
	item, err := s.scanProjectProposal(s.pool.QueryRow(ctx, `
		with updated as (
			update project_proposals
			set status=$2, review_note=$3, reviewed_by_user_id=nullif($4,'')::uuid,
				merged_release_id=nullif($5,'')::uuid, reviewed_at=now(), updated_at=now()
			where id::text=$1
			returning *
		)`+strings.Replace(projectProposalSelectSQL, "from project_proposals pr", "from updated pr", 1), proposalID, status, note, reviewedByUserID, mergedReleaseID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.ProjectProposal{}, false, nil
		}
		return domain.ProjectProposal{}, false, err
	}
	return item, true, nil
}

func (s *PostgresStore) ListAdminProjects(ctx context.Context) ([]domain.AdminProjectSummary, error) {
	rows, err := s.pool.Query(ctx, `
		select
			p.id::text,
			p.name,
			p.slug,
			p.username,
			p.owner_user_id::text,
			u.email,
			p.interactive,
			p.analytics_enabled,
			p.visibility,
			coalesce(p.current_release_id::text, ''),
			p.created_at
		from projects p
		join users u on u.id = p.owner_user_id
		order by p.created_at desc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.AdminProjectSummary{}
	for rows.Next() {
		var item domain.AdminProjectSummary
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.Username, &item.OwnerUserID, &item.OwnerEmail, &item.Interactive, &item.AnalyticsEnabled, &item.Visibility, &item.CurrentRelease, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.PublicURL = buildPublicURL(s.publicBase, item.Username, item.Slug)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateProject(ctx context.Context, userID string, input domain.ProjectCreateInput) (domain.Project, error) {
	project, err := s.scanProject(s.pool.QueryRow(ctx, `
		with created as (
			insert into projects (owner_user_id, username, slug, name, interactive, analytics_enabled, show_on_profile, allow_forks, visibility, public_key)
			values ($1::uuid, $2, $3, $4, $5, $6, $7, $8, 'unlisted', encode(gen_random_bytes(24), 'hex'))
			returning *
		)
		select `+projectSelectSQL("p")+`
		from created p
		`+projectSourceJoinSQL("p")+`
	`, userID, input.Username, input.Slug, input.Name, input.Interactive, input.AnalyticsEnabled, input.ShowOnProfile, input.AllowForks))
	if err != nil {
		return domain.Project{}, err
	}
	return project, nil
}

func (s *PostgresStore) GetProject(ctx context.Context, userID, projectID string) (domain.Project, bool, error) {
	project, err := s.scanProject(s.pool.QueryRow(ctx, `
		select `+projectSelectSQL("p")+`
		from projects p
		`+projectSourceJoinSQL("p")+`
		where p.id::text = $1 and p.owner_user_id::text = $2
	`, projectID, userID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Project{}, false, nil
		}
		return domain.Project{}, false, err
	}
	return project, true, nil
}

func (s *PostgresStore) CreateForkProject(ctx context.Context, userID string, input domain.ProjectCreateInput, source domain.Project) (domain.Project, error) {
	sourceOwner := strings.TrimSpace(source.Username)
	sourceName := strings.TrimSpace(source.Name)
	sourceURL := strings.TrimSpace(source.PublicURL)
	project, err := s.scanProject(s.pool.QueryRow(ctx, `
		with created as (
			insert into projects (
				owner_user_id, username, slug, name, interactive, analytics_enabled, show_on_profile, allow_forks, visibility, public_key,
				forked_from_project_id, forked_from_release_id, forked_from_user_id,
				forked_from_snapshot_name, forked_from_snapshot_owner, forked_from_snapshot_url
			)
			values (
				$1::uuid, $2, $3, $4, $5, $6, $7, $8, 'unlisted', encode(gen_random_bytes(24), 'hex'),
				nullif($9, '')::uuid, nullif($10, '')::uuid, nullif($11, '')::uuid,
				$12, $13, $14
			)
			returning *
		)
		select `+projectSelectSQL("p")+`
		from created p
		`+projectSourceJoinSQL("p")+`
	`, userID, input.Username, input.Slug, input.Name, input.Interactive, input.AnalyticsEnabled, input.ShowOnProfile, input.AllowForks,
		source.ID, source.CurrentRelease, "", sourceName, sourceOwner, sourceURL))
	if err != nil {
		return domain.Project{}, err
	}
	return project, nil
}

func (s *PostgresStore) DeleteProject(ctx context.Context, userID, projectID string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		delete from projects
		where id::text = $1 and owner_user_id::text = $2
	`, projectID, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *PostgresStore) GetProjectPublicAccess(ctx context.Context, projectID string) (domain.PublicProjectAccess, bool, error) {
	var access domain.PublicProjectAccess
	err := s.pool.QueryRow(ctx, `
		select p.id::text, p.username, p.slug, p.name, p.interactive, p.analytics_enabled, p.show_on_profile, p.allow_forks, p.visibility, coalesce(p.current_release_id::text, ''), p.created_at, p.public_key, p.owner_user_id::text, u.role, u.plan_code
		from projects p
		join users u on u.id = p.owner_user_id
		where p.id::text = $1
	`, projectID).Scan(
		&access.Project.ID,
		&access.Project.Username,
		&access.Project.Slug,
		&access.Project.Name,
		&access.Project.Interactive,
		&access.Project.AnalyticsEnabled,
		&access.Project.ShowOnProfile,
		&access.Project.AllowForks,
		&access.Project.Visibility,
		&access.Project.CurrentRelease,
		&access.Project.CreatedAt,
		&access.PublicKey,
		&access.OwnerUserID,
		&access.OwnerRole,
		&access.OwnerPlan,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.PublicProjectAccess{}, false, nil
		}
		return domain.PublicProjectAccess{}, false, err
	}
	access.Project.PublicURL = buildPublicURL(s.publicBase, access.Project.Username, access.Project.Slug)
	return access, true, nil
}

func (s *PostgresStore) CreateContestSubmission(ctx context.Context, input domain.ContestSubmission) (domain.ContestSubmission, error) {
	var item domain.ContestSubmission
	err := s.pool.QueryRow(ctx, `
		insert into contest_submissions (
			user_id, user_email, username, project_id, project_name, project_url,
			track, intro, story, allow_showcase, status
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'pending')
		returning id::text, user_id::text, user_email, username, project_id::text, project_name, project_url,
			track, intro, story, allow_showcase, status, admin_note, coalesce(reviewed_by::text, ''), coalesce(reviewed_at, '0001-01-01'::timestamptz), created_at, updated_at
	`, input.UserID, input.UserEmail, input.Username, input.ProjectID, input.ProjectName, input.ProjectURL,
		input.Track, input.Intro, input.Story, input.AllowShowcase).Scan(
		&item.ID,
		&item.UserID,
		&item.UserEmail,
		&item.Username,
		&item.ProjectID,
		&item.ProjectName,
		&item.ProjectURL,
		&item.Track,
		&item.Intro,
		&item.Story,
		&item.AllowShowcase,
		&item.Status,
		&item.AdminNote,
		&item.ReviewedBy,
		&item.ReviewedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

func (s *PostgresStore) GetContestSubmissionByUserProject(ctx context.Context, userID, projectID string) (domain.ContestSubmission, bool, error) {
	var item domain.ContestSubmission
	err := s.pool.QueryRow(ctx, `
		select id::text, user_id::text, user_email, username, project_id::text, project_name, project_url,
			track, intro, story, allow_showcase, status, admin_note, coalesce(reviewed_by::text, ''), coalesce(reviewed_at, '0001-01-01'::timestamptz), created_at, updated_at
		from contest_submissions
		where user_id=$1 and project_id=$2
	`, userID, projectID).Scan(
		&item.ID,
		&item.UserID,
		&item.UserEmail,
		&item.Username,
		&item.ProjectID,
		&item.ProjectName,
		&item.ProjectURL,
		&item.Track,
		&item.Intro,
		&item.Story,
		&item.AllowShowcase,
		&item.Status,
		&item.AdminNote,
		&item.ReviewedBy,
		&item.ReviewedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.ContestSubmission{}, false, nil
		}
		return domain.ContestSubmission{}, false, err
	}
	return item, true, nil
}

func (s *PostgresStore) ListAdminContestSubmissions(ctx context.Context, status string) ([]domain.ContestSubmission, error) {
	args := []any{}
	query := `
		select id::text, user_id::text, user_email, username, project_id::text, project_name, project_url,
			track, intro, story, allow_showcase, status, admin_note, coalesce(reviewed_by::text, ''), coalesce(reviewed_at, '0001-01-01'::timestamptz), created_at, updated_at
		from contest_submissions`
	if status != "" {
		args = append(args, status)
		query += ` where status=$1`
	}
	query += ` order by created_at desc`
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.ContestSubmission{}
	for rows.Next() {
		var item domain.ContestSubmission
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.UserEmail,
			&item.Username,
			&item.ProjectID,
			&item.ProjectName,
			&item.ProjectURL,
			&item.Track,
			&item.Intro,
			&item.Story,
			&item.AllowShowcase,
			&item.Status,
			&item.AdminNote,
			&item.ReviewedBy,
			&item.ReviewedAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) UpdateContestSubmissionReview(ctx context.Context, submissionID, status, adminNote, reviewedBy string) (domain.ContestSubmission, error) {
	var item domain.ContestSubmission
	err := s.pool.QueryRow(ctx, `
		update contest_submissions
		set status=$2, admin_note=$3, reviewed_by=$4, reviewed_at=now(), updated_at=now()
		where id=$1
		returning id::text, user_id::text, user_email, username, project_id::text, project_name, project_url,
			track, intro, story, allow_showcase, status, admin_note, coalesce(reviewed_by::text, ''), coalesce(reviewed_at, '0001-01-01'::timestamptz), created_at, updated_at
	`, submissionID, status, adminNote, reviewedBy).Scan(
		&item.ID,
		&item.UserID,
		&item.UserEmail,
		&item.Username,
		&item.ProjectID,
		&item.ProjectName,
		&item.ProjectURL,
		&item.Track,
		&item.Intro,
		&item.Story,
		&item.AllowShowcase,
		&item.Status,
		&item.AdminNote,
		&item.ReviewedBy,
		&item.ReviewedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

func (s *PostgresStore) UpdateProjectVisibility(ctx context.Context, userID, projectID, visibility string) (domain.Project, bool, error) {
	var project domain.Project
	err := s.pool.QueryRow(ctx, `
		update projects
		set visibility = $3
		where id::text = $1 and owner_user_id::text = $2
		returning id::text, username, slug, name, interactive, analytics_enabled, visibility, coalesce(current_release_id::text, ''), created_at
	`, projectID, userID, visibility).Scan(
		&project.ID,
		&project.Username,
		&project.Slug,
		&project.Name,
		&project.Interactive,
		&project.AnalyticsEnabled,
		&project.Visibility,
		&project.CurrentRelease,
		&project.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Project{}, false, nil
		}
		return domain.Project{}, false, err
	}
	project.PublicURL = buildPublicURL(s.publicBase, project.Username, project.Slug)
	return project, true, nil
}

func (s *PostgresStore) UpdateProjectPath(ctx context.Context, userID, projectID, slug string) (domain.Project, bool, error) {
	var project domain.Project
	err := s.pool.QueryRow(ctx, `
		update projects
		set slug = $3
		where id::text = $1 and owner_user_id::text = $2
		returning id::text, username, slug, name, interactive, analytics_enabled, visibility, coalesce(current_release_id::text, ''), created_at
	`, projectID, userID, slug).Scan(
		&project.ID,
		&project.Username,
		&project.Slug,
		&project.Name,
		&project.Interactive,
		&project.AnalyticsEnabled,
		&project.Visibility,
		&project.CurrentRelease,
		&project.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Project{}, false, nil
		}
		return domain.Project{}, false, err
	}
	project.PublicURL = buildPublicURL(s.publicBase, project.Username, project.Slug)
	return project, true, nil
}

func (s *PostgresStore) UpdateProjectSettings(ctx context.Context, userID, projectID string, input domain.ProjectSettingsUpdateInput) (domain.Project, bool, error) {
	project, err := s.scanProject(s.pool.QueryRow(ctx, `
		with updated as (
			update projects
			set name = $3,
				slug = $4,
				interactive = $5,
				analytics_enabled = $6,
				show_on_profile = $7,
				allow_forks = $8
			where id::text = $1 and owner_user_id::text = $2
			returning *
		)
		select `+projectSelectSQL("p")+`
		from updated p
		`+projectSourceJoinSQL("p")+`
	`, projectID, userID, input.Name, input.Slug, input.Interactive, input.AnalyticsEnabled, input.ShowOnProfile, input.AllowForks))
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Project{}, false, nil
		}
		return domain.Project{}, false, err
	}
	return project, true, nil
}

func (s *PostgresStore) GetAppBuildSettings(ctx context.Context, userID, projectID string) (domain.AppBuildSettings, bool, error) {
	item, err := scanAppBuildSettings(s.pool.QueryRow(ctx, `
		select id::text, user_id::text, project_id::text, app_name,
			android_enabled, windows_enabled, auto_update,
			android_package_name, windows_package_name, created_at, updated_at
		from app_build_settings
		where user_id::text = $1 and project_id::text = $2
	`, userID, projectID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.AppBuildSettings{}, false, nil
		}
		return domain.AppBuildSettings{}, false, err
	}
	return item, true, nil
}

func (s *PostgresStore) UpsertAppBuildSettings(ctx context.Context, userID, projectID string, input domain.AppBuildSettingsInput) (domain.AppBuildSettings, error) {
	item, err := scanAppBuildSettings(s.pool.QueryRow(ctx, `
		insert into app_build_settings (
			user_id, project_id, app_name, android_enabled, windows_enabled, auto_update,
			android_package_name, windows_package_name
		)
		values ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8)
		on conflict (project_id) do update set
			app_name = excluded.app_name,
			android_enabled = excluded.android_enabled,
			windows_enabled = excluded.windows_enabled,
			auto_update = excluded.auto_update,
			android_package_name = excluded.android_package_name,
			windows_package_name = excluded.windows_package_name,
			updated_at = now()
		returning id::text, user_id::text, project_id::text, app_name,
			android_enabled, windows_enabled, auto_update,
			android_package_name, windows_package_name, created_at, updated_at
	`, userID, projectID, input.AppName, input.AndroidEnabled, input.WindowsEnabled, input.AutoUpdate, input.AndroidPackageName, input.WindowsPackageName))
	return item, err
}

func (s *PostgresStore) CreateAppBuildJob(ctx context.Context, input domain.AppBuildJob) (domain.AppBuildJob, error) {
	item, err := scanAppBuildJob(s.pool.QueryRow(ctx, `
		insert into app_build_jobs (
			user_id, project_id, release_id, platform, app_name, package_name,
			version_code, version_name, auto_update, status, error_message
		)
		values ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9, $10, $11)
		returning id::text, user_id::text, project_id::text, release_id::text, platform,
			app_name, package_name, version_code, version_name, auto_update, status,
			artifact_path, artifact_sha256, artifact_size, github_run_id, error_message, created_at, updated_at
	`, input.UserID, input.ProjectID, input.ReleaseID, input.Platform, input.AppName, input.PackageName,
		input.VersionCode, input.VersionName, input.AutoUpdate, input.Status, input.ErrorMessage))
	return item, err
}

func (s *PostgresStore) GetAppBuildJob(ctx context.Context, jobID string) (domain.AppBuildJob, bool, error) {
	item, err := scanAppBuildJob(s.pool.QueryRow(ctx, `
		select id::text, user_id::text, project_id::text, release_id::text, platform,
			app_name, package_name, version_code, version_name, auto_update, status,
			artifact_path, artifact_sha256, artifact_size, github_run_id, error_message, created_at, updated_at
		from app_build_jobs
		where id::text = $1
	`, jobID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.AppBuildJob{}, false, nil
		}
		return domain.AppBuildJob{}, false, err
	}
	return item, true, nil
}

func (s *PostgresStore) ListAppBuildJobs(ctx context.Context, userID, projectID string) ([]domain.AppBuildJob, error) {
	rows, err := s.pool.Query(ctx, `
		select id::text, user_id::text, project_id::text, release_id::text, platform,
			app_name, package_name, version_code, version_name, auto_update, status,
			artifact_path, artifact_sha256, artifact_size, github_run_id, error_message, created_at, updated_at
		from app_build_jobs
		where user_id::text = $1 and project_id::text = $2
		order by created_at desc
	`, userID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.AppBuildJob
	for rows.Next() {
		item, err := scanAppBuildJob(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CountAppBuildJobsForUserSince(ctx context.Context, userID, platform string, since time.Time) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `
		select count(*)::int
		from app_build_jobs
		where user_id::text = $1 and platform = $2 and created_at >= $3
			and status in ('pending', 'building', 'succeeded')
	`, userID, platform, since).Scan(&count)
	return count, err
}

func (s *PostgresStore) CompleteAppBuildJob(ctx context.Context, jobID string, input domain.AppBuildCompleteInput) (domain.AppBuildJob, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.AppBuildJob{}, false, err
	}
	defer tx.Rollback(ctx)
	item, err := scanAppBuildJob(tx.QueryRow(ctx, `
		update app_build_jobs
		set status = $2,
			artifact_path = coalesce(nullif($3, ''), artifact_path),
			artifact_sha256 = coalesce(nullif($4, ''), artifact_sha256),
			artifact_size = case when $5 > 0 then $5 else artifact_size end,
			github_run_id = coalesce(nullif($6, ''), github_run_id),
			error_message = $7,
			updated_at = now()
		where id::text = $1
		returning id::text, user_id::text, project_id::text, release_id::text, platform,
			app_name, package_name, version_code, version_name, auto_update, status,
			artifact_path, artifact_sha256, artifact_size, github_run_id, error_message, created_at, updated_at
	`, jobID, input.Status, input.ArtifactPath, input.ArtifactSHA256, input.ArtifactSize, input.GitHubRunID, input.ErrorMessage))
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.AppBuildJob{}, false, nil
		}
		return domain.AppBuildJob{}, false, err
	}
	if item.Status == "succeeded" && item.ArtifactPath != "" {
		updateDownloadURL := item.ArtifactPath
		if item.Platform == "windows" {
			updateDownloadURL = item.ArtifactPath + "?kind=update"
		}
		_, err = tx.Exec(ctx, `
			insert into app_update_channels (
				project_id, platform, channel, latest_job_id, latest_release_id,
				latest_version_code, latest_version_name, download_url, sha256, size, release_note, updated_at
			)
			values ($1::uuid, $2, 'stable', $3::uuid, $4::uuid, $5, $6, $7, $8, $9, $10, now())
			on conflict (project_id, platform, channel) do update set
				latest_job_id = excluded.latest_job_id,
				latest_release_id = excluded.latest_release_id,
				latest_version_code = excluded.latest_version_code,
				latest_version_name = excluded.latest_version_name,
				download_url = excluded.download_url,
				sha256 = excluded.sha256,
				size = excluded.size,
				release_note = excluded.release_note,
				updated_at = now()
		`, item.ProjectID, item.Platform, item.ID, item.ReleaseID, item.VersionCode, item.VersionName, updateDownloadURL, item.ArtifactSHA256, item.ArtifactSize, "作者更新了作品内容")
		if err != nil {
			return domain.AppBuildJob{}, false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.AppBuildJob{}, false, err
	}
	return item, true, nil
}

func (s *PostgresStore) GetAppUpdateInfo(ctx context.Context, projectID, platform string, versionCode int) (domain.AppUpdateInfo, error) {
	var info domain.AppUpdateInfo
	err := s.pool.QueryRow(ctx, `
		select latest_version_code, latest_version_name, release_note, download_url, sha256, size, force_update
		from app_update_channels
		where project_id::text = $1 and platform = $2 and channel = 'stable'
	`, projectID, platform).Scan(&info.LatestVersionCode, &info.LatestVersionName, &info.ReleaseNote, &info.DownloadURL, &info.SHA256, &info.Size, &info.Force)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.AppUpdateInfo{HasUpdate: false}, nil
		}
		return domain.AppUpdateInfo{}, err
	}
	info.HasUpdate = info.LatestVersionCode > versionCode && info.DownloadURL != ""
	return info, nil
}

func (s *PostgresStore) ListCollections(ctx context.Context, projectID string) ([]domain.Collection, error) {
	rows, err := s.pool.Query(ctx, `
		select id::text, project_id::text, name, permissions, created_at
		from collections
		where project_id::text = $1
		order by created_at asc
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.Collection{}
	for rows.Next() {
		var item domain.Collection
		var permissions []byte
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Name, &permissions, &item.CreatedAt); err != nil {
			return nil, err
		}
		if len(permissions) > 0 {
			if err := json.Unmarshal(permissions, &item.Permissions); err != nil {
				return nil, err
			}
		}
		item.Permissions = normalizeStoredPermissions(item.Permissions)
		fields, err := s.listFields(ctx, item.ID)
		if err != nil {
			return nil, err
		}
		item.Fields = fields
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) GetCollectionByName(ctx context.Context, projectID, collectionName string) (domain.Collection, bool, error) {
	var item domain.Collection
	var permissions []byte
	err := s.pool.QueryRow(ctx, `
		select id::text, project_id::text, name, permissions, created_at
		from collections
		where project_id::text = $1 and name = $2
	`, projectID, collectionName).Scan(&item.ID, &item.ProjectID, &item.Name, &permissions, &item.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Collection{}, false, nil
		}
		return domain.Collection{}, false, err
	}
	if len(permissions) > 0 {
		if err := json.Unmarshal(permissions, &item.Permissions); err != nil {
			return domain.Collection{}, false, err
		}
	}
	item.Permissions = normalizeStoredPermissions(item.Permissions)
	fields, err := s.listFields(ctx, item.ID)
	if err != nil {
		return domain.Collection{}, false, err
	}
	item.Fields = fields
	return item, true, nil
}

func (s *PostgresStore) UpdateCollection(ctx context.Context, projectID, collectionName string, input domain.CollectionUpdateInput) (domain.Collection, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Collection{}, false, err
	}
	defer tx.Rollback(ctx)

	normalizedPermissions := normalizeStoredPermissions(input.Permissions)
	permissionsJSON, err := json.Marshal(normalizedPermissions)
	if err != nil {
		return domain.Collection{}, false, err
	}

	var collection domain.Collection
	err = tx.QueryRow(ctx, `
		update collections
		set permissions = $3::jsonb
		where project_id::text = $1 and name = $2
		returning id::text, project_id::text, name, created_at
	`, projectID, collectionName, permissionsJSON).Scan(
		&collection.ID,
		&collection.ProjectID,
		&collection.Name,
		&collection.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Collection{}, false, nil
		}
		return domain.Collection{}, false, err
	}
	collection.Permissions = normalizedPermissions

	if _, err := tx.Exec(ctx, `delete from collection_fields where collection_id::text = $1`, collection.ID); err != nil {
		return domain.Collection{}, false, err
	}
	for index, field := range input.Fields {
		_, err := tx.Exec(ctx, `
			insert into collection_fields (collection_id, name, type, required, is_list, reference_collection, sort_order)
			values ($1::uuid, $2, $3, $4, $5, $6, $7)
		`, collection.ID, field.Name, field.Type, field.Required, field.IsList, field.Reference, index)
		if err != nil {
			return domain.Collection{}, false, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Collection{}, false, err
	}
	collection.Fields = input.Fields
	return collection, true, nil
}

func (s *PostgresStore) DeleteCollection(ctx context.Context, projectID, collectionName string) error {
	result, err := s.pool.Exec(ctx, `
		delete from collections
		where project_id::text = $1 and name = $2
	`, projectID, collectionName)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("鎵句笉鍒拌繖涓綔鍝佹暟鎹〃")
	}
	return nil
}

func (s *PostgresStore) CreateCollection(ctx context.Context, projectID string, input domain.CollectionCreateInput) (domain.Collection, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Collection{}, err
	}
	defer tx.Rollback(ctx)

	normalizedPermissions := normalizeStoredPermissions(input.Permissions)
	permissionsJSON, err := json.Marshal(normalizedPermissions)
	if err != nil {
		return domain.Collection{}, err
	}

	var collection domain.Collection
	err = tx.QueryRow(ctx, `
		insert into collections (project_id, name, permissions)
		values ($1::uuid, $2, $3::jsonb)
		returning id::text, project_id::text, name, created_at
	`, projectID, input.Name, permissionsJSON).Scan(
		&collection.ID,
		&collection.ProjectID,
		&collection.Name,
		&collection.CreatedAt,
	)
	if err != nil {
		return domain.Collection{}, err
	}
	collection.Permissions = normalizedPermissions

	for index, field := range input.Fields {
		_, err := tx.Exec(ctx, `
			insert into collection_fields (collection_id, name, type, required, is_list, reference_collection, sort_order)
			values ($1::uuid, $2, $3, $4, $5, $6, $7)
		`, collection.ID, field.Name, field.Type, field.Required, field.IsList, field.Reference, index)
		if err != nil {
			return domain.Collection{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Collection{}, err
	}
	collection.Fields = input.Fields
	return collection, nil
}

func (s *PostgresStore) listFields(ctx context.Context, collectionID string) ([]domain.FieldSchema, error) {
	rows, err := s.pool.Query(ctx, `
		select name, type, required, is_list, reference_collection
		from collection_fields
		where collection_id::text = $1
		order by sort_order asc
	`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fields := []domain.FieldSchema{}
	for rows.Next() {
		var field domain.FieldSchema
		if err := rows.Scan(&field.Name, &field.Type, &field.Required, &field.IsList, &field.Reference); err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}
	return fields, rows.Err()
}

func (s *PostgresStore) ListRecords(ctx context.Context, projectID, collectionName string) ([]domain.Record, error) {
	collection, ok, err := s.GetCollectionByName(ctx, projectID, collectionName)
	if err != nil {
		return nil, err
	}
	if !ok {
		return []domain.Record{}, nil
	}

	rows, err := s.pool.Query(ctx, `
		select id::text, project_id::text, collection_id::text, data, status, created_at, updated_at
		from records
		where project_id::text = $1 and collection_id::text = $2
		order by created_at desc
	`, projectID, collection.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.Record{}
	for rows.Next() {
		var record domain.Record
		var raw []byte
		if err := rows.Scan(&record.ID, &record.ProjectID, &record.CollectionID, &raw, &record.Status, &record.CreatedAt, &record.UpdatedAt); err != nil {
			return nil, err
		}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &record.Data); err != nil {
				return nil, err
			}
		}
		items = append(items, record)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateRecord(ctx context.Context, projectID, collectionName, createdByUserID string, input domain.RecordCreateInput) (domain.Record, error) {
	collection, ok, err := s.GetCollectionByName(ctx, projectID, collectionName)
	if err != nil {
		return domain.Record{}, err
	}
	if !ok {
		return domain.Record{}, fmt.Errorf("找不到这个集合")
	}

	raw, err := json.Marshal(input.Data)
	if err != nil {
		return domain.Record{}, err
	}

	var record domain.Record
	err = s.pool.QueryRow(ctx, `
		insert into records (project_id, collection_id, created_by_user_id, data, status)
		values ($1::uuid, $2::uuid, nullif($3, '')::uuid, $4::jsonb, 'active')
		returning id::text, project_id::text, collection_id::text, data, status, created_at, updated_at
	`, projectID, collection.ID, createdByUserID, raw).Scan(
		&record.ID,
		&record.ProjectID,
		&record.CollectionID,
		&raw,
		&record.Status,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return domain.Record{}, err
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &record.Data); err != nil {
			return domain.Record{}, err
		}
	}
	return record, nil
}

func (s *PostgresStore) GetRecord(ctx context.Context, projectID, collectionName, recordID string) (domain.Record, bool, error) {
	collection, ok, err := s.GetCollectionByName(ctx, projectID, collectionName)
	if err != nil {
		return domain.Record{}, false, err
	}
	if !ok {
		return domain.Record{}, false, nil
	}

	var record domain.Record
	var raw []byte
	err = s.pool.QueryRow(ctx, `
		select id::text, project_id::text, collection_id::text, data, status, created_at, updated_at
		from records
		where id::text = $1 and project_id::text = $2 and collection_id::text = $3
	`, recordID, projectID, collection.ID).Scan(
		&record.ID,
		&record.ProjectID,
		&record.CollectionID,
		&raw,
		&record.Status,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Record{}, false, nil
		}
		return domain.Record{}, false, err
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &record.Data); err != nil {
			return domain.Record{}, false, err
		}
	}
	return record, true, nil
}

func (s *PostgresStore) UpdateRecord(ctx context.Context, projectID, collectionName, recordID string, input domain.RecordUpdateInput) (domain.Record, bool, error) {
	collection, ok, err := s.GetCollectionByName(ctx, projectID, collectionName)
	if err != nil {
		return domain.Record{}, false, err
	}
	if !ok {
		return domain.Record{}, false, nil
	}

	raw, err := json.Marshal(input.Data)
	if err != nil {
		return domain.Record{}, false, err
	}

	var record domain.Record
	err = s.pool.QueryRow(ctx, `
		update records
		set data = $4::jsonb, updated_at = now()
		where id::text = $1 and project_id::text = $2 and collection_id::text = $3
		returning id::text, project_id::text, collection_id::text, data, status, created_at, updated_at
	`, recordID, projectID, collection.ID, raw).Scan(
		&record.ID,
		&record.ProjectID,
		&record.CollectionID,
		&raw,
		&record.Status,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Record{}, false, nil
		}
		return domain.Record{}, false, err
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &record.Data); err != nil {
			return domain.Record{}, false, err
		}
	}
	return record, true, nil
}

func (s *PostgresStore) DeleteRecord(ctx context.Context, projectID, collectionName, recordID string) (bool, error) {
	collection, ok, err := s.GetCollectionByName(ctx, projectID, collectionName)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	result, err := s.pool.Exec(ctx, `
		delete from records
		where id::text = $1 and project_id::text = $2 and collection_id::text = $3
	`, recordID, projectID, collection.ID)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

func (s *PostgresStore) ProjectDataStats(ctx context.Context, projectID string) (domain.ProjectDataStats, error) {
	var stats domain.ProjectDataStats
	stats.ProjectID = projectID
	err := s.pool.QueryRow(ctx, `
		select
			coalesce((select count(*) from records where project_id::text = $1), 0)::bigint,
			coalesce((select sum(octet_length(data::text)) from records where project_id::text = $1), 0)::bigint,
			coalesce((select count(*) from collections where project_id::text = $1), 0)::bigint
	`, projectID).Scan(&stats.RecordCount, &stats.DataBytes, &stats.CollectionCount)
	if err != nil {
		return domain.ProjectDataStats{}, err
	}
	return stats, nil
}

func (s *PostgresStore) GetMonthlyProjectUsage(ctx context.Context, projectID, monthKey string) (domain.ProjectUsage, error) {
	var usage domain.ProjectUsage
	err := s.pool.QueryRow(ctx, `
		select project_id::text, month_key, query_count, write_count, updated_at
		from project_usage_monthly
		where project_id::text = $1 and month_key = $2
	`, projectID, monthKey).Scan(&usage.ProjectID, &usage.MonthKey, &usage.QueryCount, &usage.WriteCount, &usage.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.ProjectUsage{ProjectID: projectID, MonthKey: monthKey}, nil
		}
		return domain.ProjectUsage{}, err
	}
	return usage, nil
}

func (s *PostgresStore) IncrementProjectUsage(ctx context.Context, projectID, monthKey string, queryDelta, writeDelta int64) (domain.ProjectUsage, error) {
	var usage domain.ProjectUsage
	err := s.pool.QueryRow(ctx, `
		insert into project_usage_monthly (project_id, month_key, query_count, write_count, updated_at)
		values ($1::uuid, $2, $3, $4, now())
		on conflict (project_id, month_key) do update
		set query_count = project_usage_monthly.query_count + excluded.query_count,
			write_count = project_usage_monthly.write_count + excluded.write_count,
			updated_at = now()
		returning project_id::text, month_key, query_count, write_count, updated_at
	`, projectID, monthKey, queryDelta, writeDelta).Scan(&usage.ProjectID, &usage.MonthKey, &usage.QueryCount, &usage.WriteCount, &usage.UpdatedAt)
	if err != nil {
		return domain.ProjectUsage{}, err
	}
	return usage, nil
}

func (s *PostgresStore) IncrementProjectDailyStats(ctx context.Context, projectID, day string, pageViews, apiRequests, apiSuccesses, apiFailures int64) error {
	_, err := s.pool.Exec(ctx, `
		insert into project_stats_daily (project_id, day, page_views, api_requests, api_successes, api_failures, updated_at)
		values ($1::uuid, $2::date, $3, $4, $5, $6, now())
		on conflict (project_id, day) do update
		set page_views = project_stats_daily.page_views + excluded.page_views,
			api_requests = project_stats_daily.api_requests + excluded.api_requests,
			api_successes = project_stats_daily.api_successes + excluded.api_successes,
			api_failures = project_stats_daily.api_failures + excluded.api_failures,
			updated_at = now()
	`, projectID, day, pageViews, apiRequests, apiSuccesses, apiFailures)
	return err
}

func (s *PostgresStore) ProjectStats(ctx context.Context, projectID, from, to string) (domain.ProjectStatsSummary, error) {
	rows, err := s.pool.Query(ctx, `
		select day::text, page_views, api_requests, api_successes, api_failures
		from project_stats_daily
		where project_id::text = $1 and day >= $2::date and day <= $3::date
		order by day asc
	`, projectID, from, to)
	if err != nil {
		return domain.ProjectStatsSummary{}, err
	}
	defer rows.Close()

	summary := domain.ProjectStatsSummary{ProjectID: projectID, From: from, To: to, Items: []domain.ProjectDailyStats{}}
	for rows.Next() {
		var item domain.ProjectDailyStats
		if err := rows.Scan(&item.Date, &item.PageViews, &item.APIRequests, &item.APISuccesses, &item.APIFailures); err != nil {
			return domain.ProjectStatsSummary{}, err
		}
		summary.TotalPageViews += item.PageViews
		summary.TotalAPIRequests += item.APIRequests
		summary.TotalAPISuccesses += item.APISuccesses
		summary.TotalAPIFailures += item.APIFailures
		summary.Items = append(summary.Items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.ProjectStatsSummary{}, err
	}
	if summary.TotalAPIRequests > 0 {
		summary.APISuccessRate = float64(summary.TotalAPISuccesses) / float64(summary.TotalAPIRequests)
		summary.APIFailureRate = float64(summary.TotalAPIFailures) / float64(summary.TotalAPIRequests)
	}
	return summary, nil
}

func (s *PostgresStore) CreateProjectDomain(ctx context.Context, input domain.ProjectDomain) (domain.ProjectDomain, error) {
	err := s.pool.QueryRow(ctx, `
		insert into project_domains (project_id, owner_user_id, subdomain, domain, status, reject_reason, admin_note)
		values ($1::uuid, $2::uuid, $3, $4, $5, $6, $7)
		returning id::text, project_id::text, owner_user_id::text, subdomain, domain, status, reject_reason, admin_note,
			coalesce(reviewed_by_user_id::text, ''), coalesce(reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), created_at, updated_at
	`, input.ProjectID, input.OwnerUserID, input.Subdomain, input.Domain, input.Status, input.RejectReason, input.AdminNote).Scan(
		&input.ID, &input.ProjectID, &input.OwnerUserID, &input.Subdomain, &input.Domain, &input.Status, &input.RejectReason, &input.AdminNote,
		&input.ReviewedBy, &input.ReviewedAt, &input.CreatedAt, &input.UpdatedAt,
	)
	if err != nil {
		return domain.ProjectDomain{}, err
	}
	return input, nil
}

func (s *PostgresStore) ListProjectDomains(ctx context.Context, projectID string) ([]domain.ProjectDomain, error) {
	rows, err := s.pool.Query(ctx, `
		select id::text, project_id::text, owner_user_id::text, subdomain, domain, status, reject_reason, admin_note,
			coalesce(reviewed_by_user_id::text, ''), coalesce(reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), created_at, updated_at
		from project_domains
		where project_id::text = $1
		order by created_at desc
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.ProjectDomain{}
	for rows.Next() {
		var item domain.ProjectDomain
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.OwnerUserID, &item.Subdomain, &item.Domain, &item.Status, &item.RejectReason, &item.AdminNote, &item.ReviewedBy, &item.ReviewedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) ListAdminProjectDomains(ctx context.Context, status string) ([]domain.ProjectDomain, error) {
	query := `
		select d.id::text, d.project_id::text, p.name, p.username, p.slug, d.owner_user_id::text, u.email, coalesce(u.username, ''),
			d.subdomain, d.domain, d.status, d.reject_reason, d.admin_note, coalesce(d.reviewed_by_user_id::text, ''),
			coalesce(d.reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), d.created_at, d.updated_at
		from project_domains d
		join projects p on p.id = d.project_id
		join users u on u.id = d.owner_user_id`
	args := []any{}
	if status != "" {
		query += " where d.status = $1"
		args = append(args, status)
	}
	query += " order by d.created_at desc"
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.ProjectDomain{}
	for rows.Next() {
		var item domain.ProjectDomain
		var projectUsername string
		var projectSlug string
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.ProjectName, &projectUsername, &projectSlug, &item.OwnerUserID, &item.OwnerEmail, &item.Username, &item.Subdomain, &item.Domain, &item.Status, &item.RejectReason, &item.AdminNote, &item.ReviewedBy, &item.ReviewedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.ProjectPublicURL = buildPublicURL(s.publicBase, projectUsername, projectSlug)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) GetActiveProjectDomainAccessByDomain(ctx context.Context, host string) (domain.ProjectDomain, domain.Project, bool, error) {
	var item domain.ProjectDomain
	var project domain.Project
	err := s.pool.QueryRow(ctx, `
		select d.id::text, d.project_id::text, d.owner_user_id::text, d.subdomain, d.domain, d.status,
			d.reject_reason, d.admin_note, coalesce(d.reviewed_by_user_id::text, ''),
			coalesce(d.reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), d.created_at, d.updated_at,
			p.id::text, p.username, p.slug, p.name, p.interactive, p.analytics_enabled, p.visibility,
			coalesce(p.current_release_id::text, ''), p.created_at
		from project_domains d
		join projects p on p.id = d.project_id
		where d.domain = $1 and d.status = 'active'
		order by d.updated_at desc
		limit 1
	`, host).Scan(
		&item.ID, &item.ProjectID, &item.OwnerUserID, &item.Subdomain, &item.Domain, &item.Status,
		&item.RejectReason, &item.AdminNote, &item.ReviewedBy, &item.ReviewedAt, &item.CreatedAt, &item.UpdatedAt,
		&project.ID, &project.Username, &project.Slug, &project.Name, &project.Interactive, &project.AnalyticsEnabled,
		&project.Visibility, &project.CurrentRelease, &project.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.ProjectDomain{}, domain.Project{}, false, nil
		}
		return domain.ProjectDomain{}, domain.Project{}, false, err
	}
	project.PublicURL = buildPublicURL(s.publicBase, project.Username, project.Slug)
	item.ProjectName = project.Name
	item.ProjectPublicURL = project.PublicURL
	return item, project, true, nil
}

func (s *PostgresStore) UpdateProjectDomainReview(ctx context.Context, domainID, status, rejectReason, adminNote, reviewedBy string) (domain.ProjectDomain, error) {
	var item domain.ProjectDomain
	var projectUsername string
	var projectSlug string
	err := s.pool.QueryRow(ctx, `
		with updated as (
			update project_domains
			set status = $2, reject_reason = $3, admin_note = $4, reviewed_by_user_id = nullif($5, '')::uuid, reviewed_at = now(), updated_at = now()
			where id::text = $1
			returning id, project_id, owner_user_id, subdomain, domain, status, reject_reason, admin_note,
				reviewed_by_user_id, reviewed_at, created_at, updated_at
		)
		select d.id::text, d.project_id::text, p.name, p.username, p.slug, d.owner_user_id::text, u.email, coalesce(u.username, ''),
			d.subdomain, d.domain, d.status, d.reject_reason, d.admin_note, coalesce(d.reviewed_by_user_id::text, ''),
			d.reviewed_at, d.created_at, d.updated_at
		from updated d
		join projects p on p.id = d.project_id
		join users u on u.id = d.owner_user_id
	`, domainID, status, rejectReason, adminNote, reviewedBy).Scan(&item.ID, &item.ProjectID, &item.ProjectName, &projectUsername, &projectSlug, &item.OwnerUserID, &item.OwnerEmail, &item.Username, &item.Subdomain, &item.Domain, &item.Status, &item.RejectReason, &item.AdminNote, &item.ReviewedBy, &item.ReviewedAt, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return domain.ProjectDomain{}, err
	}
	item.ProjectPublicURL = buildPublicURL(s.publicBase, projectUsername, projectSlug)
	return item, nil
}

func (s *PostgresStore) CreateProjectDomainDeleteRequest(ctx context.Context, input domain.ProjectDomainDeleteRequest) (domain.ProjectDomainDeleteRequest, error) {
	err := s.pool.QueryRow(ctx, `
		insert into project_domain_delete_requests (domain_id, project_id, owner_user_id, domain, reason, status, admin_note)
		values (nullif($1, '')::uuid, nullif($2, '')::uuid, nullif($3, '')::uuid, $4, $5, $6, $7)
		returning id::text, coalesce(domain_id::text, ''), coalesce(project_id::text, ''), coalesce(owner_user_id::text, ''),
			domain, reason, status, admin_note, coalesce(reviewed_by_user_id::text, ''),
			coalesce(reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), created_at, updated_at
	`, input.DomainID, input.ProjectID, input.OwnerUserID, input.Domain, input.Reason, input.Status, input.AdminNote).Scan(
		&input.ID, &input.DomainID, &input.ProjectID, &input.OwnerUserID, &input.Domain, &input.Reason, &input.Status,
		&input.AdminNote, &input.ReviewedBy, &input.ReviewedAt, &input.CreatedAt, &input.UpdatedAt,
	)
	if err != nil {
		return domain.ProjectDomainDeleteRequest{}, err
	}
	return input, nil
}

func (s *PostgresStore) ListProjectDomainDeleteRequests(ctx context.Context, projectID string) ([]domain.ProjectDomainDeleteRequest, error) {
	rows, err := s.pool.Query(ctx, `
		select r.id::text, coalesce(r.domain_id::text, ''), coalesce(r.project_id::text, ''), coalesce(p.name, ''),
			coalesce(p.username, ''), coalesce(p.slug, ''), coalesce(r.owner_user_id::text, ''), coalesce(u.email, ''), coalesce(u.username, ''),
			r.domain, r.reason, r.status, r.admin_note, coalesce(r.reviewed_by_user_id::text, ''),
			coalesce(r.reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), r.created_at, r.updated_at
		from project_domain_delete_requests r
		left join projects p on p.id = r.project_id
		left join users u on u.id = r.owner_user_id
		where r.project_id::text = $1
		order by r.created_at desc
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProjectDomainDeleteRequests(rows, s.publicBase)
}

func (s *PostgresStore) ListAdminProjectDomainDeleteRequests(ctx context.Context, status string) ([]domain.ProjectDomainDeleteRequest, error) {
	query := `
		select r.id::text, coalesce(r.domain_id::text, ''), coalesce(r.project_id::text, ''), coalesce(p.name, ''),
			coalesce(p.username, ''), coalesce(p.slug, ''), coalesce(r.owner_user_id::text, ''), coalesce(u.email, ''), coalesce(u.username, ''),
			r.domain, r.reason, r.status, r.admin_note, coalesce(r.reviewed_by_user_id::text, ''),
			coalesce(r.reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), r.created_at, r.updated_at
		from project_domain_delete_requests r
		left join projects p on p.id = r.project_id
		left join users u on u.id = r.owner_user_id`
	args := []any{}
	if status != "" {
		query += " where r.status = $1"
		args = append(args, status)
	}
	query += " order by r.created_at desc"
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProjectDomainDeleteRequests(rows, s.publicBase)
}

func (s *PostgresStore) UpdateProjectDomainDeleteRequest(ctx context.Context, requestID, status, adminNote, reviewedBy string) (domain.ProjectDomainDeleteRequest, error) {
	var item domain.ProjectDomainDeleteRequest
	var projectName, projectUsername, projectSlug string
	err := s.pool.QueryRow(ctx, `
		with updated as (
			update project_domain_delete_requests
			set status = $2, admin_note = $3, reviewed_by_user_id = nullif($4, '')::uuid, reviewed_at = now(), updated_at = now()
			where id::text = $1
			returning id, domain_id, project_id, owner_user_id, domain, reason, status, admin_note, reviewed_by_user_id, reviewed_at, created_at, updated_at
		), disabled_domain as (
			update project_domains
			set status = 'disabled', admin_note = case when $2 = 'completed' then $3 else admin_note end, updated_at = now()
			where id in (select domain_id from updated where $2 = 'completed' and domain_id is not null)
		)
		select u.id::text, coalesce(u.domain_id::text, ''), coalesce(u.project_id::text, ''), coalesce(p.name, ''),
			coalesce(p.username, ''), coalesce(p.slug, ''), coalesce(u.owner_user_id::text, ''), coalesce(owner.email, ''), coalesce(owner.username, ''),
			u.domain, u.reason, u.status, u.admin_note, coalesce(u.reviewed_by_user_id::text, ''),
			u.reviewed_at, u.created_at, u.updated_at
		from updated u
		left join projects p on p.id = u.project_id
		left join users owner on owner.id = u.owner_user_id
	`, requestID, status, adminNote, reviewedBy).Scan(
		&item.ID, &item.DomainID, &item.ProjectID, &projectName, &projectUsername, &projectSlug, &item.OwnerUserID,
		&item.OwnerEmail, &item.Username, &item.Domain, &item.Reason, &item.Status, &item.AdminNote, &item.ReviewedBy,
		&item.ReviewedAt, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return domain.ProjectDomainDeleteRequest{}, err
	}
	item.ProjectName = projectName
	if projectUsername != "" && projectSlug != "" {
		item.ProjectPublicURL = buildPublicURL(s.publicBase, projectUsername, projectSlug)
	}
	return item, nil
}

type projectDomainDeleteRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanProjectDomainDeleteRequests(rows projectDomainDeleteRows, publicBase string) ([]domain.ProjectDomainDeleteRequest, error) {
	items := []domain.ProjectDomainDeleteRequest{}
	for rows.Next() {
		var item domain.ProjectDomainDeleteRequest
		var projectUsername, projectSlug string
		if err := rows.Scan(&item.ID, &item.DomainID, &item.ProjectID, &item.ProjectName, &projectUsername, &projectSlug,
			&item.OwnerUserID, &item.OwnerEmail, &item.Username, &item.Domain, &item.Reason, &item.Status, &item.AdminNote,
			&item.ReviewedBy, &item.ReviewedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if projectUsername != "" && projectSlug != "" {
			item.ProjectPublicURL = buildPublicURL(publicBase, projectUsername, projectSlug)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateRepairRequest(ctx context.Context, input domain.RepairRequest) (domain.RepairRequest, error) {
	err := s.pool.QueryRow(ctx, `
		insert into repair_requests (project_id, owner_user_id, issue_type, description, expected, allow_admin_edit, contact, status, admin_reply, user_reply)
		values ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, '')
		returning id::text, project_id::text, owner_user_id::text, issue_type, description, expected, allow_admin_edit, contact,
			status, admin_reply, user_reply, coalesce(user_replied_at, '0001-01-01T00:00:00Z'::timestamptz),
			coalesce(reviewed_by_user_id::text, ''), coalesce(reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), created_at, updated_at
	`, input.ProjectID, input.OwnerUserID, input.IssueType, input.Description, input.Expected, input.AllowAdminEdit, input.Contact, input.Status, input.AdminReply).Scan(
		&input.ID, &input.ProjectID, &input.OwnerUserID, &input.IssueType, &input.Description, &input.Expected, &input.AllowAdminEdit, &input.Contact,
		&input.Status, &input.AdminReply, &input.UserReply, &input.UserRepliedAt, &input.ReviewedBy, &input.ReviewedAt, &input.CreatedAt, &input.UpdatedAt,
	)
	if err != nil {
		return domain.RepairRequest{}, err
	}
	return input, nil
}

func (s *PostgresStore) ListProjectRepairRequests(ctx context.Context, projectID string) ([]domain.RepairRequest, error) {
	rows, err := s.pool.Query(ctx, `
		select id::text, project_id::text, owner_user_id::text, issue_type, description, expected, allow_admin_edit, contact,
			status, admin_reply, user_reply, coalesce(user_replied_at, '0001-01-01T00:00:00Z'::timestamptz),
			coalesce(reviewed_by_user_id::text, ''), coalesce(reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), created_at, updated_at
		from repair_requests
		where project_id::text = $1
		order by created_at desc
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.RepairRequest{}
	for rows.Next() {
		var item domain.RepairRequest
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.OwnerUserID, &item.IssueType, &item.Description, &item.Expected, &item.AllowAdminEdit, &item.Contact, &item.Status, &item.AdminReply, &item.UserReply, &item.UserRepliedAt, &item.ReviewedBy, &item.ReviewedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) ListAdminRepairRequests(ctx context.Context, status string) ([]domain.RepairRequest, error) {
	query := `
		select r.id::text, r.project_id::text, p.name, p.username, p.slug, r.owner_user_id::text, u.email, coalesce(u.username, ''),
			r.issue_type, r.description, r.expected, r.allow_admin_edit, r.contact, r.status, r.admin_reply,
			r.user_reply, coalesce(r.user_replied_at, '0001-01-01T00:00:00Z'::timestamptz),
			coalesce(r.reviewed_by_user_id::text, ''), coalesce(r.reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), r.created_at, r.updated_at
		from repair_requests r
		join projects p on p.id = r.project_id
		join users u on u.id = r.owner_user_id`
	args := []any{}
	if status != "" {
		query += " where r.status = $1"
		args = append(args, status)
	}
	query += " order by r.created_at desc"
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.RepairRequest{}
	for rows.Next() {
		var item domain.RepairRequest
		var projectUsername string
		var projectSlug string
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.ProjectName, &projectUsername, &projectSlug, &item.OwnerUserID, &item.OwnerEmail, &item.Username, &item.IssueType, &item.Description, &item.Expected, &item.AllowAdminEdit, &item.Contact, &item.Status, &item.AdminReply, &item.UserReply, &item.UserRepliedAt, &item.ReviewedBy, &item.ReviewedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.ProjectPublicURL = buildPublicURL(s.publicBase, projectUsername, projectSlug)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) UpdateRepairRequest(ctx context.Context, requestID, status, adminReply, reviewedBy string) (domain.RepairRequest, error) {
	var item domain.RepairRequest
	var projectUsername string
	var projectSlug string
	err := s.pool.QueryRow(ctx, `
		with updated as (
			update repair_requests
			set status = $2, admin_reply = $3, reviewed_by_user_id = nullif($4, '')::uuid, reviewed_at = now(), updated_at = now()
			where id::text = $1
			returning id, project_id, owner_user_id, issue_type, description, expected, allow_admin_edit, contact,
				status, admin_reply, user_reply, user_replied_at, reviewed_by_user_id, reviewed_at, created_at, updated_at
		)
		select r.id::text, r.project_id::text, p.name, p.username, p.slug, r.owner_user_id::text, u.email, coalesce(u.username, ''),
			r.issue_type, r.description, r.expected, r.allow_admin_edit, r.contact, r.status, r.admin_reply, r.user_reply,
			coalesce(r.user_replied_at, '0001-01-01T00:00:00Z'::timestamptz),
			coalesce(r.reviewed_by_user_id::text, ''), r.reviewed_at, r.created_at, r.updated_at
		from updated r
		join projects p on p.id = r.project_id
		join users u on u.id = r.owner_user_id
	`, requestID, status, adminReply, reviewedBy).Scan(&item.ID, &item.ProjectID, &item.ProjectName, &projectUsername, &projectSlug, &item.OwnerUserID, &item.OwnerEmail, &item.Username, &item.IssueType, &item.Description, &item.Expected, &item.AllowAdminEdit, &item.Contact, &item.Status, &item.AdminReply, &item.UserReply, &item.UserRepliedAt, &item.ReviewedBy, &item.ReviewedAt, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return domain.RepairRequest{}, err
	}
	item.ProjectPublicURL = buildPublicURL(s.publicBase, projectUsername, projectSlug)
	return item, nil
}

func (s *PostgresStore) ReplyRepairRequest(ctx context.Context, projectID, requestID, reply string) (domain.RepairRequest, error) {
	var item domain.RepairRequest
	err := s.pool.QueryRow(ctx, `
		update repair_requests
		set user_reply = $3, user_replied_at = now(), status = 'processing', updated_at = now()
		where project_id::text = $1 and id::text = $2 and status = 'need_info' and user_reply = ''
		returning id::text, project_id::text, owner_user_id::text, issue_type, description, expected, allow_admin_edit, contact,
			status, admin_reply, user_reply, user_replied_at,
			coalesce(reviewed_by_user_id::text, ''), coalesce(reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), created_at, updated_at
	`, projectID, requestID, reply).Scan(&item.ID, &item.ProjectID, &item.OwnerUserID, &item.IssueType, &item.Description, &item.Expected, &item.AllowAdminEdit, &item.Contact, &item.Status, &item.AdminReply, &item.UserReply, &item.UserRepliedAt, &item.ReviewedBy, &item.ReviewedAt, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return domain.RepairRequest{}, err
	}
	return item, nil
}

func (s *PostgresStore) GetRepairRequest(ctx context.Context, projectID, requestID string) (domain.RepairRequest, bool, error) {
	var item domain.RepairRequest
	err := s.pool.QueryRow(ctx, `
		select id::text, project_id::text, owner_user_id::text, issue_type, description, expected, allow_admin_edit, contact,
			status, admin_reply, user_reply, coalesce(user_replied_at, '0001-01-01T00:00:00Z'::timestamptz),
			coalesce(reviewed_by_user_id::text, ''), coalesce(reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), created_at, updated_at
		from repair_requests
		where project_id::text = $1 and id::text = $2
	`, projectID, requestID).Scan(&item.ID, &item.ProjectID, &item.OwnerUserID, &item.IssueType, &item.Description, &item.Expected, &item.AllowAdminEdit, &item.Contact, &item.Status, &item.AdminReply, &item.UserReply, &item.UserRepliedAt, &item.ReviewedBy, &item.ReviewedAt, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.RepairRequest{}, false, nil
		}
		return domain.RepairRequest{}, false, err
	}
	return item, true, nil
}

func (s *PostgresStore) CreateRepairAIJob(ctx context.Context, input domain.RepairAIJob) (domain.RepairAIJob, error) {
	if input.Status == "" {
		input.Status = "running"
	}
	if input.Round <= 0 {
		input.Round = 1
	}
	err := s.pool.QueryRow(ctx, `
		insert into repair_ai_jobs (repair_request_id, project_id, owner_user_id, status, round, feedback)
		values ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6)
		returning id::text, repair_request_id::text, project_id::text, owner_user_id::text, status, round, feedback, coalesce(generated_html, ''), coalesce(preview_url, ''), coalesce(error_message, ''), created_at, updated_at, coalesce(finished_at, '0001-01-01T00:00:00Z'::timestamptz)
	`, input.RepairRequestID, input.ProjectID, input.OwnerUserID, input.Status, input.Round, input.Feedback).Scan(&input.ID, &input.RepairRequestID, &input.ProjectID, &input.OwnerUserID, &input.Status, &input.Round, &input.Feedback, &input.GeneratedHTML, &input.PreviewURL, &input.ErrorMessage, &input.CreatedAt, &input.UpdatedAt, &input.FinishedAt)
	return input, err
}

func (s *PostgresStore) GetRepairAIJob(ctx context.Context, jobID string) (domain.RepairAIJob, bool, error) {
	var item domain.RepairAIJob
	err := s.pool.QueryRow(ctx, `
		select id::text, repair_request_id::text, project_id::text, owner_user_id::text, status, round, feedback, coalesce(generated_html, ''), coalesce(preview_url, ''), coalesce(error_message, ''), created_at, updated_at, coalesce(finished_at, '0001-01-01T00:00:00Z'::timestamptz)
		from repair_ai_jobs where id::text = $1
	`, jobID).Scan(&item.ID, &item.RepairRequestID, &item.ProjectID, &item.OwnerUserID, &item.Status, &item.Round, &item.Feedback, &item.GeneratedHTML, &item.PreviewURL, &item.ErrorMessage, &item.CreatedAt, &item.UpdatedAt, &item.FinishedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.RepairAIJob{}, false, nil
		}
		return domain.RepairAIJob{}, false, err
	}
	return item, true, nil
}

func (s *PostgresStore) GetLatestRepairAIJob(ctx context.Context, repairRequestID string) (domain.RepairAIJob, bool, error) {
	var item domain.RepairAIJob
	err := s.pool.QueryRow(ctx, `
		select id::text, repair_request_id::text, project_id::text, owner_user_id::text, status, round, feedback, coalesce(generated_html, ''), coalesce(preview_url, ''), coalesce(error_message, ''), created_at, updated_at, coalesce(finished_at, '0001-01-01T00:00:00Z'::timestamptz)
		from repair_ai_jobs where repair_request_id::text = $1 order by created_at desc limit 1
	`, repairRequestID).Scan(&item.ID, &item.RepairRequestID, &item.ProjectID, &item.OwnerUserID, &item.Status, &item.Round, &item.Feedback, &item.GeneratedHTML, &item.PreviewURL, &item.ErrorMessage, &item.CreatedAt, &item.UpdatedAt, &item.FinishedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.RepairAIJob{}, false, nil
		}
		return domain.RepairAIJob{}, false, err
	}
	return item, true, nil
}

func (s *PostgresStore) ListRepairAIJobs(ctx context.Context, repairRequestID string) ([]domain.RepairAIJob, error) {
	rows, err := s.pool.Query(ctx, `
		select id::text, repair_request_id::text, project_id::text, owner_user_id::text, status, round, feedback, coalesce(generated_html, ''), coalesce(preview_url, ''), coalesce(error_message, ''), created_at, updated_at, coalesce(finished_at, '0001-01-01T00:00:00Z'::timestamptz)
		from repair_ai_jobs where repair_request_id::text = $1 order by round asc, created_at asc
	`, repairRequestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.RepairAIJob{}
	for rows.Next() {
		var item domain.RepairAIJob
		if err := rows.Scan(&item.ID, &item.RepairRequestID, &item.ProjectID, &item.OwnerUserID, &item.Status, &item.Round, &item.Feedback, &item.GeneratedHTML, &item.PreviewURL, &item.ErrorMessage, &item.CreatedAt, &item.UpdatedAt, &item.FinishedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) GetActiveRepairAIJobByUser(ctx context.Context, userID string) (domain.RepairAIJob, bool, error) {
	var item domain.RepairAIJob
	err := s.pool.QueryRow(ctx, `
		select id::text, repair_request_id::text, project_id::text, owner_user_id::text, status, round, feedback, coalesce(generated_html, ''), coalesce(preview_url, ''), coalesce(error_message, ''), created_at, updated_at, coalesce(finished_at, '0001-01-01T00:00:00Z'::timestamptz)
		from repair_ai_jobs where owner_user_id::text = $1 and status in ('queued', 'running') order by created_at desc limit 1
	`, userID).Scan(&item.ID, &item.RepairRequestID, &item.ProjectID, &item.OwnerUserID, &item.Status, &item.Round, &item.Feedback, &item.GeneratedHTML, &item.PreviewURL, &item.ErrorMessage, &item.CreatedAt, &item.UpdatedAt, &item.FinishedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.RepairAIJob{}, false, nil
		}
		return domain.RepairAIJob{}, false, err
	}
	return item, true, nil
}

func (s *PostgresStore) CountRepairAIJobsForUserSince(ctx context.Context, userID string, since time.Time) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `select count(*) from repair_ai_jobs where owner_user_id::text = $1 and created_at >= $2`, userID, since).Scan(&count)
	return count, err
}

func (s *PostgresStore) UpdateRepairAIJob(ctx context.Context, job domain.RepairAIJob) (domain.RepairAIJob, error) {
	err := s.pool.QueryRow(ctx, `
		update repair_ai_jobs
		set status = $2, round = $3, feedback = $4, generated_html = $5, preview_url = $6, error_message = $7, updated_at = now(), finished_at = nullif($8, '0001-01-01T00:00:00Z')::timestamptz
		where id::text = $1
		returning id::text, repair_request_id::text, project_id::text, owner_user_id::text, status, round, feedback, coalesce(generated_html, ''), coalesce(preview_url, ''), coalesce(error_message, ''), created_at, updated_at, coalesce(finished_at, '0001-01-01T00:00:00Z'::timestamptz)
	`, job.ID, job.Status, job.Round, job.Feedback, job.GeneratedHTML, job.PreviewURL, job.ErrorMessage, job.FinishedAt.Format(time.RFC3339)).Scan(&job.ID, &job.RepairRequestID, &job.ProjectID, &job.OwnerUserID, &job.Status, &job.Round, &job.Feedback, &job.GeneratedHTML, &job.PreviewURL, &job.ErrorMessage, &job.CreatedAt, &job.UpdatedAt, &job.FinishedAt)
	return job, err
}

func (s *PostgresStore) CreateRepairAIMessage(ctx context.Context, input domain.RepairAIMessage) (domain.RepairAIMessage, error) {
	err := s.pool.QueryRow(ctx, `
		insert into repair_ai_messages (job_id, repair_request_id, project_id, owner_user_id, agent_key, agent_name, role, visibility, message_type, content, metadata_json, message_seq)
		values ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8, $9, $10, $11, coalesce((select max(message_seq) + 1 from repair_ai_messages where job_id = $1::uuid), 1))
		returning id::text, job_id::text, repair_request_id::text, project_id::text, owner_user_id::text, agent_key, agent_name, role, visibility, message_type, content, metadata_json, message_seq, created_at
	`, input.JobID, input.RepairRequestID, input.ProjectID, input.OwnerUserID, input.AgentKey, input.AgentName, input.Role, input.Visibility, input.MessageType, input.Content, input.MetadataJSON).Scan(&input.ID, &input.JobID, &input.RepairRequestID, &input.ProjectID, &input.OwnerUserID, &input.AgentKey, &input.AgentName, &input.Role, &input.Visibility, &input.MessageType, &input.Content, &input.MetadataJSON, &input.MessageSeq, &input.CreatedAt)
	return input, err
}

func (s *PostgresStore) ListRepairAIMessages(ctx context.Context, jobID string) ([]domain.RepairAIMessage, error) {
	rows, err := s.pool.Query(ctx, `
		select id::text, job_id::text, repair_request_id::text, project_id::text, owner_user_id::text, agent_key, agent_name, role, visibility, message_type, content, coalesce(metadata_json, ''), message_seq, created_at
		from repair_ai_messages where job_id::text = $1 order by message_seq asc
	`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.RepairAIMessage{}
	for rows.Next() {
		var item domain.RepairAIMessage
		if err := rows.Scan(&item.ID, &item.JobID, &item.RepairRequestID, &item.ProjectID, &item.OwnerUserID, &item.AgentKey, &item.AgentName, &item.Role, &item.Visibility, &item.MessageType, &item.Content, &item.MetadataJSON, &item.MessageSeq, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) IsReservedSubdomain(ctx context.Context, subdomain string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `select exists(select 1 from reserved_subdomains where subdomain = $1)`, subdomain).Scan(&exists)
	return exists, err
}

func (s *PostgresStore) CreateTemplateSubmission(ctx context.Context, input domain.TemplateSubmission) (domain.TemplateSubmission, error) {
	tagsJSON, configJSON, collectionsJSON, err := marshalTemplateSubmissionJSON(input)
	if err != nil {
		return domain.TemplateSubmission{}, err
	}
	err = s.pool.QueryRow(ctx, `
		insert into template_submissions (
			author_user_id, slug, name, category, category_label, summary, description,
			tags, interactive_required, analytics_recommended, config_fields, collections,
			html_source, source_type, status
		)
		values ($1::uuid, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10, $11::jsonb, $12::jsonb, $13, $14, 'pending')
		returning id::text, status, created_at, updated_at
	`, input.AuthorUserID, input.Slug, input.Name, input.Category, input.CategoryLabel, input.Summary, input.Description, tagsJSON, input.InteractiveRequired, input.AnalyticsRecommended, configJSON, collectionsJSON, input.HTMLSource, input.SourceType).Scan(&input.ID, &input.Status, &input.CreatedAt, &input.UpdatedAt)
	if err != nil {
		return domain.TemplateSubmission{}, err
	}
	return s.decorateTemplateSubmission(ctx, input)
}

func (s *PostgresStore) ListPublishedTemplateSubmissions(ctx context.Context) ([]domain.TemplateSubmission, error) {
	rows, err := s.pool.Query(ctx, templateSubmissionSelectSQL()+`
		where t.status = 'published'
		order by t.created_at desc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTemplateSubmissions(rows, false)
}

func (s *PostgresStore) ListMyTemplateSubmissions(ctx context.Context, userID string) ([]domain.TemplateSubmission, error) {
	rows, err := s.pool.Query(ctx, templateSubmissionSelectSQL()+`
		where t.author_user_id::text = $1
		order by t.created_at desc
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTemplateSubmissions(rows, true)
}

func (s *PostgresStore) ListAdminTemplateSubmissions(ctx context.Context, status string) ([]domain.TemplateSubmission, error) {
	query := templateSubmissionSelectSQL()
	args := []any{}
	if status != "" {
		query += ` where t.status = $1`
		args = append(args, status)
	}
	query += ` order by t.created_at desc`
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTemplateSubmissions(rows, true)
}

func (s *PostgresStore) GetTemplateSubmission(ctx context.Context, submissionID string) (domain.TemplateSubmission, bool, error) {
	rows, err := s.pool.Query(ctx, templateSubmissionSelectSQL()+`
		where t.id::text = $1
	`, submissionID)
	if err != nil {
		return domain.TemplateSubmission{}, false, err
	}
	defer rows.Close()
	items, err := scanTemplateSubmissions(rows, true)
	if err != nil {
		return domain.TemplateSubmission{}, false, err
	}
	if len(items) == 0 {
		return domain.TemplateSubmission{}, false, nil
	}
	return items[0], true, nil
}

func (s *PostgresStore) GetPublishedTemplateSubmissionByIDOrSlug(ctx context.Context, idOrSlug string) (domain.TemplateSubmission, bool, error) {
	idOrSlug = strings.TrimPrefix(idOrSlug, "submission:")
	rows, err := s.pool.Query(ctx, templateSubmissionSelectSQL()+`
		where t.status = 'published' and (t.id::text = $1 or t.slug = $1)
	`, idOrSlug)
	if err != nil {
		return domain.TemplateSubmission{}, false, err
	}
	defer rows.Close()
	items, err := scanTemplateSubmissions(rows, true)
	if err != nil {
		return domain.TemplateSubmission{}, false, err
	}
	if len(items) == 0 {
		return domain.TemplateSubmission{}, false, nil
	}
	return items[0], true, nil
}

func (s *PostgresStore) UpdateTemplateSubmissionReview(ctx context.Context, submissionID, status, adminNote, reviewedBy string) (domain.TemplateSubmission, error) {
	var item domain.TemplateSubmission
	var tagsJSON, configJSON, collectionsJSON []byte
	err := s.pool.QueryRow(ctx, `
		update template_submissions
		set status = $2, admin_note = $3, reviewed_by_user_id = nullif($4, '')::uuid, reviewed_at = now(), updated_at = now()
		where id::text = $1
		returning id::text, author_user_id::text, slug, name, category, category_label, summary, description,
			tags, interactive_required, analytics_recommended, config_fields, collections,
			html_source, source_type, status, admin_note, coalesce(reviewed_by_user_id::text, ''),
			coalesce(reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), created_at, updated_at
	`, submissionID, status, adminNote, reviewedBy).Scan(
		&item.ID, &item.AuthorUserID, &item.Slug, &item.Name, &item.Category, &item.CategoryLabel, &item.Summary, &item.Description,
		&tagsJSON, &item.InteractiveRequired, &item.AnalyticsRecommended, &configJSON, &collectionsJSON,
		&item.HTMLSource, &item.SourceType, &item.Status, &item.AdminNote, &item.ReviewedBy, &item.ReviewedAt, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return domain.TemplateSubmission{}, err
	}
	if err := unmarshalTemplateSubmissionJSON(&item, tagsJSON, configJSON, collectionsJSON); err != nil {
		return domain.TemplateSubmission{}, err
	}
	return s.decorateTemplateSubmission(ctx, item)
}

func templateSubmissionSelectSQL() string {
	return `
		select
			t.id::text, t.author_user_id::text, u.email, coalesce(u.username, ''),
			t.slug, t.name, t.category, t.category_label, t.summary, t.description,
			t.tags, t.interactive_required, t.analytics_recommended, t.config_fields, t.collections,
			t.html_source, t.source_type, t.status, t.admin_note, coalesce(t.reviewed_by_user_id::text, ''),
			coalesce(t.reviewed_at, '0001-01-01T00:00:00Z'::timestamptz), t.created_at, t.updated_at
		from template_submissions t
		join users u on u.id = t.author_user_id
	`
}

type templateSubmissionRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanTemplateSubmissions(rows templateSubmissionRows, includeHTML bool) ([]domain.TemplateSubmission, error) {
	items := []domain.TemplateSubmission{}
	for rows.Next() {
		var item domain.TemplateSubmission
		var tagsJSON, configJSON, collectionsJSON []byte
		if err := rows.Scan(
			&item.ID, &item.AuthorUserID, &item.AuthorEmail, &item.AuthorName,
			&item.Slug, &item.Name, &item.Category, &item.CategoryLabel, &item.Summary, &item.Description,
			&tagsJSON, &item.InteractiveRequired, &item.AnalyticsRecommended, &configJSON, &collectionsJSON,
			&item.HTMLSource, &item.SourceType, &item.Status, &item.AdminNote, &item.ReviewedBy,
			&item.ReviewedAt, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if item.AuthorName == "" {
			item.AuthorName = item.AuthorEmail
		}
		if err := unmarshalTemplateSubmissionJSON(&item, tagsJSON, configJSON, collectionsJSON); err != nil {
			return nil, err
		}
		if !includeHTML {
			item.HTMLSource = ""
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func marshalTemplateSubmissionJSON(input domain.TemplateSubmission) ([]byte, []byte, []byte, error) {
	tagsJSON, err := json.Marshal(input.Tags)
	if err != nil {
		return nil, nil, nil, err
	}
	configJSON, err := json.Marshal(input.ConfigFields)
	if err != nil {
		return nil, nil, nil, err
	}
	collectionsJSON, err := json.Marshal(input.Collections)
	if err != nil {
		return nil, nil, nil, err
	}
	return tagsJSON, configJSON, collectionsJSON, nil
}

func unmarshalTemplateSubmissionJSON(item *domain.TemplateSubmission, tagsJSON, configJSON, collectionsJSON []byte) error {
	if len(tagsJSON) > 0 {
		if err := json.Unmarshal(tagsJSON, &item.Tags); err != nil {
			return err
		}
	}
	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &item.ConfigFields); err != nil {
			return err
		}
	}
	if len(collectionsJSON) > 0 {
		if err := json.Unmarshal(collectionsJSON, &item.Collections); err != nil {
			return err
		}
	}
	if item.Tags == nil {
		item.Tags = []string{}
	}
	if item.ConfigFields == nil {
		item.ConfigFields = []domain.TemplateConfigField{}
	}
	if item.Collections == nil {
		item.Collections = []domain.TemplateCollectionDefinition{}
	}
	return nil
}

func (s *PostgresStore) decorateTemplateSubmission(ctx context.Context, item domain.TemplateSubmission) (domain.TemplateSubmission, error) {
	user, ok, err := s.GetUserByID(ctx, item.AuthorUserID)
	if err != nil {
		return domain.TemplateSubmission{}, err
	}
	if ok {
		item.AuthorEmail = user.Email
		item.AuthorName = user.Username
		if item.AuthorName == "" {
			item.AuthorName = user.Email
		}
	}
	return item, nil
}

func (s *PostgresStore) CreateUpgradeRequest(ctx context.Context, input domain.UpgradeRequest) (domain.UpgradeRequest, error) {
	err := s.pool.QueryRow(ctx, `
		insert into upgrade_requests (user_id, current_plan, target_plan, payment_method, payer_note, system_note, status, admin_note)
		values ($1::uuid, $2, $3, $4, $5, $6, $7, $8)
		returning id::text, created_at
	`, input.UserID, input.CurrentPlan, input.TargetPlan, input.PaymentMethod, input.PayerNote, input.SystemNote, input.Status, input.AdminNote).Scan(&input.ID, &input.CreatedAt)
	if err != nil {
		return domain.UpgradeRequest{}, err
	}
	return input, nil
}

func (s *PostgresStore) ListUpgradeRequests(ctx context.Context) ([]domain.UpgradeRequest, error) {
	rows, err := s.pool.Query(ctx, `
		select
			r.id::text,
			r.user_id::text,
			u.email,
			coalesce(u.username, ''),
			r.current_plan,
			r.target_plan,
			r.payment_method,
			r.payer_note,
			r.system_note,
			r.status,
			r.admin_note,
			coalesce(r.reviewed_by_user_id::text, ''),
			coalesce(r.reviewed_at, '0001-01-01T00:00:00Z'::timestamptz),
			r.created_at
		from upgrade_requests r
		join users u on u.id = r.user_id
		order by r.created_at desc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.UpgradeRequest{}
	for rows.Next() {
		var item domain.UpgradeRequest
		if err := rows.Scan(&item.ID, &item.UserID, &item.UserEmail, &item.Username, &item.CurrentPlan, &item.TargetPlan, &item.PaymentMethod, &item.PayerNote, &item.SystemNote, &item.Status, &item.AdminNote, &item.ReviewedBy, &item.ReviewedAt, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) ListUserUpgradeRequests(ctx context.Context, userID string) ([]domain.UpgradeRequest, error) {
	rows, err := s.pool.Query(ctx, `
		select
			r.id::text,
			r.user_id::text,
			u.email,
			coalesce(u.username, ''),
			r.current_plan,
			r.target_plan,
			r.payment_method,
			r.payer_note,
			r.system_note,
			r.status,
			r.admin_note,
			coalesce(r.reviewed_by_user_id::text, ''),
			coalesce(r.reviewed_at, '0001-01-01T00:00:00Z'::timestamptz),
			r.created_at
		from upgrade_requests r
		join users u on u.id = r.user_id
		where r.user_id::text = $1
		order by r.created_at desc
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.UpgradeRequest{}
	for rows.Next() {
		var item domain.UpgradeRequest
		if err := rows.Scan(&item.ID, &item.UserID, &item.UserEmail, &item.Username, &item.CurrentPlan, &item.TargetPlan, &item.PaymentMethod, &item.PayerNote, &item.SystemNote, &item.Status, &item.AdminNote, &item.ReviewedBy, &item.ReviewedAt, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) UpdateUpgradeRequest(ctx context.Context, requestID, status, adminNote, reviewedBy string) (domain.UpgradeRequest, error) {
	var item domain.UpgradeRequest
	err := s.pool.QueryRow(ctx, `
		update upgrade_requests
		set status = $2, admin_note = $3, reviewed_by_user_id = nullif($4, '')::uuid, reviewed_at = now()
		where id::text = $1
		returning id::text, user_id::text, current_plan, target_plan, payment_method, payer_note, system_note, status, admin_note, coalesce(reviewed_by_user_id::text, ''), reviewed_at, created_at
	`, requestID, status, adminNote, reviewedBy).Scan(&item.ID, &item.UserID, &item.CurrentPlan, &item.TargetPlan, &item.PaymentMethod, &item.PayerNote, &item.SystemNote, &item.Status, &item.AdminNote, &item.ReviewedBy, &item.ReviewedAt, &item.CreatedAt)
	if err != nil {
		return domain.UpgradeRequest{}, err
	}
	user, ok, err := s.GetUserByID(ctx, item.UserID)
	if err != nil {
		return domain.UpgradeRequest{}, err
	}
	if ok {
		item.UserEmail = user.Email
		item.Username = user.Username
	}
	return item, nil
}

func (s *PostgresStore) CreateRelease(ctx context.Context, release domain.Release) (domain.Release, error) {
	err := s.pool.QueryRow(ctx, `
		insert into project_releases (project_id, status, source_archive_path, public_dir_path, entry_file, change_note)
		values ($1::uuid, $2, $3, $4, $5, $6)
		returning id::text, project_id::text, status, source_archive_path, public_dir_path, entry_file, change_note, created_at
	`, release.ProjectID, release.Status, release.ArchivePath, release.PublicPath, release.EntryFile, release.ChangeNote).Scan(
		&release.ID,
		&release.ProjectID,
		&release.Status,
		&release.ArchivePath,
		&release.PublicPath,
		&release.EntryFile,
		&release.ChangeNote,
		&release.CreatedAt,
	)
	if err != nil {
		return domain.Release{}, err
	}
	if err := s.SetCurrentRelease(ctx, release.ProjectID, release.ID); err != nil {
		return domain.Release{}, err
	}
	return release, nil
}

func (s *PostgresStore) ListReleases(ctx context.Context, projectID string) ([]domain.Release, error) {
	rows, err := s.pool.Query(ctx, `
		select id::text, project_id::text, status, source_archive_path, public_dir_path, entry_file, coalesce(change_note, ''), created_at
		from project_releases
		where project_id::text = $1
		order by created_at desc
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.Release{}
	for rows.Next() {
		var release domain.Release
		if err := rows.Scan(&release.ID, &release.ProjectID, &release.Status, &release.ArchivePath, &release.PublicPath, &release.EntryFile, &release.ChangeNote, &release.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, release)
	}
	return items, rows.Err()
}

func (s *PostgresStore) SetCurrentRelease(ctx context.Context, projectID, releaseID string) error {
	_, err := s.pool.Exec(ctx, `
		update projects
		set current_release_id = $2::uuid
		where id::text = $1
	`, projectID, releaseID)
	return err
}
