alter table users add column if not exists username text;
alter table users add constraint users_username_key unique (username);

create table if not exists auth_codes (
  id uuid primary key default gen_random_uuid(),
  email text not null,
  username text not null default '',
  code text not null,
  purpose text not null default 'login',
  expires_at timestamptz not null,
  consumed_at timestamptz,
  created_at timestamptz not null default now()
);

create index if not exists idx_auth_codes_email_created_at
  on auth_codes (email, created_at desc);

create table if not exists sessions (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  token text not null unique,
  expires_at timestamptz not null,
  created_at timestamptz not null default now()
);

create index if not exists idx_sessions_token
  on sessions (token);
