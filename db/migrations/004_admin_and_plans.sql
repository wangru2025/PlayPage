alter table users
  add column if not exists role text not null default 'user';

alter table users
  add column if not exists plan_code text not null default 'free';

create table if not exists project_usage_monthly (
  project_id uuid not null references projects(id) on delete cascade,
  month_key text not null,
  query_count bigint not null default 0,
  write_count bigint not null default 0,
  updated_at timestamptz not null default now(),
  primary key (project_id, month_key)
);

create index if not exists idx_project_usage_monthly_month_key
  on project_usage_monthly (month_key);
