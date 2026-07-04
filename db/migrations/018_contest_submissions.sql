create table if not exists contest_submissions (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  user_email text not null default '',
  username text not null default '',
  project_id uuid not null references projects(id) on delete cascade,
  project_name text not null default '',
  project_url text not null default '',
  track text not null,
  intro text not null default '',
  story text not null default '',
  allow_showcase boolean not null default true,
  status text not null default 'pending',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique(user_id, project_id)
);

create index if not exists contest_submissions_status_created_idx on contest_submissions(status, created_at desc);
create index if not exists contest_submissions_project_idx on contest_submissions(project_id);
