alter table projects add column if not exists forked_from_project_id uuid null references projects(id) on delete set null;
alter table projects add column if not exists forked_from_release_id uuid null references project_releases(id) on delete set null;
alter table projects add column if not exists forked_from_user_id uuid null references users(id) on delete set null;
alter table projects add column if not exists forked_from_snapshot_name text not null default '';
alter table projects add column if not exists forked_from_snapshot_owner text not null default '';
alter table projects add column if not exists forked_from_snapshot_url text not null default '';
alter table projects add column if not exists allow_forks boolean not null default true;

create table if not exists project_favorites (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  project_id uuid not null references projects(id) on delete cascade,
  created_at timestamptz not null default now(),
  unique(user_id, project_id)
);

create index if not exists project_favorites_project_idx on project_favorites(project_id);
create index if not exists projects_forked_from_project_idx on projects(forked_from_project_id);
