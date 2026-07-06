create table if not exists author_follows (
  id uuid primary key default gen_random_uuid(),
  follower_user_id uuid not null references users(id) on delete cascade,
  target_user_id uuid not null references users(id) on delete cascade,
  created_at timestamptz not null default now(),
  unique(follower_user_id, target_user_id),
  check (follower_user_id <> target_user_id)
);

create index if not exists author_follows_follower_idx on author_follows(follower_user_id, created_at desc);
create index if not exists author_follows_target_idx on author_follows(target_user_id, created_at desc);
