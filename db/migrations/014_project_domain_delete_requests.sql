create table if not exists project_domain_delete_requests (
  id uuid primary key default gen_random_uuid(),
  domain_id uuid,
  project_id uuid references projects(id) on delete set null,
  owner_user_id uuid references users(id) on delete set null,
  domain text not null,
  reason text not null default '',
  status text not null default 'pending',
  admin_note text not null default '',
  reviewed_by_user_id uuid references users(id),
  reviewed_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists idx_project_domain_delete_requests_project_id on project_domain_delete_requests(project_id, created_at desc);
create index if not exists idx_project_domain_delete_requests_status on project_domain_delete_requests(status, created_at desc);

create unique index if not exists idx_project_domain_delete_requests_one_pending_per_domain
  on project_domain_delete_requests(domain_id)
  where status = 'pending';
