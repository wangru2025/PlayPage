create table if not exists app_build_settings (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  project_id uuid not null references projects(id) on delete cascade,
  app_name text not null default '',
  android_enabled boolean not null default false,
  windows_enabled boolean not null default false,
  auto_update boolean not null default false,
  android_package_name text not null default '',
  windows_package_name text not null default '',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique(project_id)
);

create table if not exists app_build_jobs (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  project_id uuid not null references projects(id) on delete cascade,
  release_id uuid not null references project_releases(id) on delete restrict,
  platform text not null check (platform in ('android', 'windows')),
  app_name text not null,
  package_name text not null default '',
  version_code integer not null default 1,
  version_name text not null default '1.0.0',
  auto_update boolean not null default false,
  status text not null default 'pending' check (status in ('pending', 'building', 'succeeded', 'failed', 'skipped')),
  artifact_path text not null default '',
  artifact_sha256 text not null default '',
  artifact_size bigint not null default 0,
  github_run_id text not null default '',
  error_message text not null default '',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists idx_app_build_jobs_project_created on app_build_jobs(project_id, created_at desc);
create index if not exists idx_app_build_jobs_user_platform_created on app_build_jobs(user_id, platform, created_at desc);

create table if not exists app_update_channels (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references projects(id) on delete cascade,
  platform text not null check (platform in ('android', 'windows')),
  channel text not null default 'stable',
  latest_job_id uuid references app_build_jobs(id) on delete set null,
  latest_release_id uuid references project_releases(id) on delete set null,
  latest_version_code integer not null default 0,
  latest_version_name text not null default '',
  download_url text not null default '',
  sha256 text not null default '',
  size bigint not null default 0,
  force_update boolean not null default false,
  release_note text not null default '',
  updated_at timestamptz not null default now(),
  unique(project_id, platform, channel)
);
