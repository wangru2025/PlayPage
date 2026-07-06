create table if not exists project_discussions (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references projects(id) on delete cascade,
  author_user_id uuid not null references users(id) on delete cascade,
  title text not null,
  body text not null,
  status text not null default 'open',
  closed_by_user_id uuid null references users(id) on delete set null,
  closed_at timestamptz null,
  last_commented_at timestamptz not null default now(),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  check (status in ('open', 'closed'))
);

create table if not exists project_discussion_comments (
  id uuid primary key default gen_random_uuid(),
  discussion_id uuid not null references project_discussions(id) on delete cascade,
  author_user_id uuid not null references users(id) on delete cascade,
  body text not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists project_discussions_project_idx on project_discussions(project_id, status, last_commented_at desc);
create index if not exists project_discussion_comments_discussion_idx on project_discussion_comments(discussion_id, created_at asc);
