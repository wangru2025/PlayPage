alter table projects
  add column if not exists analytics_enabled boolean not null default false;

create table if not exists project_stats_daily (
  project_id uuid not null references projects(id) on delete cascade,
  day date not null,
  page_views bigint not null default 0,
  api_requests bigint not null default 0,
  api_successes bigint not null default 0,
  api_failures bigint not null default 0,
  updated_at timestamptz not null default now(),
  primary key (project_id, day)
);

create index if not exists idx_project_stats_daily_day
  on project_stats_daily(day);
