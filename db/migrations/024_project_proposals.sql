create table if not exists project_proposals (
  id uuid primary key default gen_random_uuid(),
  source_project_id uuid not null references projects(id) on delete cascade,
  target_project_id uuid not null references projects(id) on delete cascade,
  author_user_id uuid not null references users(id) on delete cascade,
  target_owner_user_id uuid not null references users(id) on delete cascade,
  title text not null,
  body text not null,
  status text not null default 'open',
  source_release_id uuid not null references project_releases(id) on delete restrict,
  merged_release_id uuid null references project_releases(id) on delete set null,
  review_note text not null default '',
  reviewed_by_user_id uuid null references users(id) on delete set null,
  reviewed_at timestamptz null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  check (status in ('open', 'accepted', 'rejected', 'closed'))
);

create index if not exists project_proposals_source_idx on project_proposals(source_project_id, created_at desc);
create index if not exists project_proposals_target_idx on project_proposals(target_project_id, status, created_at desc);
create index if not exists project_proposals_author_idx on project_proposals(author_user_id, created_at desc);
create index if not exists project_proposals_target_owner_idx on project_proposals(target_owner_user_id, created_at desc);
