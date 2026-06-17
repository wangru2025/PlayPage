package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ai-static-host/api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool       *pgxpool.Pool
	publicBase string
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
		select id::text, username, slug, name, interactive, analytics_enabled, visibility, coalesce(current_release_id::text, ''), created_at
		from projects
		where owner_user_id::text = $1
		order by created_at desc
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.Project{}
	for rows.Next() {
		var item domain.Project
		if err := rows.Scan(&item.ID, &item.Username, &item.Slug, &item.Name, &item.Interactive, &item.AnalyticsEnabled, &item.Visibility, &item.CurrentRelease, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.PublicURL = buildPublicURL(s.publicBase, item.Username, item.Slug)
		items = append(items, item)
	}

	return items, rows.Err()
}

func (s *PostgresStore) ListPublicProjects(ctx context.Context) ([]domain.Project, error) {
	rows, err := s.pool.Query(ctx, `
		select id::text, username, slug, name, interactive, analytics_enabled, visibility, coalesce(current_release_id::text, ''), created_at
		from projects
		where visibility = 'public' and current_release_id is not null
		order by created_at desc
		limit 60
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.Project{}
	for rows.Next() {
		var item domain.Project
		if err := rows.Scan(&item.ID, &item.Username, &item.Slug, &item.Name, &item.Interactive, &item.AnalyticsEnabled, &item.Visibility, &item.CurrentRelease, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.PublicURL = buildPublicURL(s.publicBase, item.Username, item.Slug)
		items = append(items, item)
	}

	return items, rows.Err()
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
	var project domain.Project
	err := s.pool.QueryRow(ctx, `
		insert into projects (owner_user_id, username, slug, name, interactive, analytics_enabled, visibility, public_key)
		values ($1::uuid, $2, $3, $4, $5, $6, 'unlisted', encode(gen_random_bytes(24), 'hex'))
		returning id::text, username, slug, name, interactive, analytics_enabled, visibility, coalesce(current_release_id::text, ''), created_at
	`, userID, input.Username, input.Slug, input.Name, input.Interactive, input.AnalyticsEnabled).Scan(
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
		return domain.Project{}, err
	}
	project.PublicURL = buildPublicURL(s.publicBase, project.Username, project.Slug)
	return project, nil
}

func (s *PostgresStore) GetProject(ctx context.Context, userID, projectID string) (domain.Project, bool, error) {
	var project domain.Project
	err := s.pool.QueryRow(ctx, `
		select id::text, username, slug, name, interactive, analytics_enabled, visibility, coalesce(current_release_id::text, ''), created_at
		from projects
		where id::text = $1 and owner_user_id::text = $2
	`, projectID, userID).Scan(
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
		select p.id::text, p.username, p.slug, p.name, p.interactive, p.analytics_enabled, p.visibility, coalesce(p.current_release_id::text, ''), p.created_at, p.public_key, p.owner_user_id::text, u.role, u.plan_code
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
		insert into project_releases (project_id, status, source_archive_path, public_dir_path, entry_file)
		values ($1::uuid, $2, $3, $4, $5)
		returning id::text, project_id::text, status, source_archive_path, public_dir_path, entry_file, created_at
	`, release.ProjectID, release.Status, release.ArchivePath, release.PublicPath, release.EntryFile).Scan(
		&release.ID,
		&release.ProjectID,
		&release.Status,
		&release.ArchivePath,
		&release.PublicPath,
		&release.EntryFile,
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
		select id::text, project_id::text, status, source_archive_path, public_dir_path, entry_file, created_at
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
		if err := rows.Scan(&release.ID, &release.ProjectID, &release.Status, &release.ArchivePath, &release.PublicPath, &release.EntryFile, &release.CreatedAt); err != nil {
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
