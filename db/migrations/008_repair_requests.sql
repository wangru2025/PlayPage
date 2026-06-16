create table if not exists repair_requests (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references projects(id) on delete cascade,
  owner_user_id uuid not null references users(id) on delete cascade,
  issue_type text not null default '',
  description text not null default '',
  expected text not null default '',
  allow_admin_edit boolean not null default false,
  contact text not null default '',
  status text not null default 'pending',
  admin_reply text not null default '',
  reviewed_by_user_id uuid references users(id),
  reviewed_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists idx_repair_requests_project_id on repair_requests(project_id, created_at desc);
create index if not exists idx_repair_requests_status on repair_requests(status, created_at desc);

create unique index if not exists idx_repair_requests_one_open_per_project
  on repair_requests(project_id)
  where status in ('pending', 'processing', 'need_info');
