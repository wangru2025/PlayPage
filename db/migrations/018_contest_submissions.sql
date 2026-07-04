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
  admin_note text not null default '',
  reviewed_by uuid null references users(id) on delete set null,
  reviewed_at timestamptz null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique(user_id, project_id)
);

create index if not exists contest_submissions_status_created_idx on contest_submissions(status, created_at desc);
create index if not exists contest_submissions_project_idx on contest_submissions(project_id);

alter table contest_submissions add column if not exists admin_note text not null default '';
alter table contest_submissions add column if not exists reviewed_by uuid null references users(id) on delete set null;
alter table contest_submissions add column if not exists reviewed_at timestamptz null;
