create table if not exists project_email_codes (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references projects(id) on delete cascade,
  owner_user_id uuid not null references users(id) on delete cascade,
  email text not null,
  purpose text not null default 'login',
  code text not null,
  expires_at timestamptz not null,
  consumed_at timestamptz,
  created_at timestamptz not null default now()
);

create index if not exists idx_project_email_codes_lookup
  on project_email_codes (project_id, email, purpose, created_at desc);

create index if not exists idx_project_email_codes_owner_created
  on project_email_codes (owner_user_id, created_at desc);

create table if not exists project_email_quota_daily (
  owner_user_id uuid not null references users(id) on delete cascade,
  day date not null,
  sent_count integer not null default 0,
  primary key (owner_user_id, day)
);
