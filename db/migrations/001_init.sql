create extension if not exists "pgcrypto";

create table if not exists users (
  id uuid primary key default gen_random_uuid(),
  email text not null unique,
  status text not null default 'active',
  created_at timestamptz not null default now()
);

create table if not exists projects (
  id uuid primary key default gen_random_uuid(),
  owner_user_id uuid not null references users(id),
  username text not null,
  slug text not null,
  name text not null,
  visibility text not null default 'unlisted',
  current_release_id uuid,
  public_key text not null,
  created_at timestamptz not null default now(),
  unique (username, slug)
);

create table if not exists project_releases (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references projects(id) on delete cascade,
  status text not null default 'processing',
  source_archive_path text not null,
  public_dir_path text not null,
  entry_file text not null default 'index.html',
  created_at timestamptz not null default now()
);

create table if not exists collections (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references projects(id) on delete cascade,
  name text not null,
  permissions jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  unique (project_id, name)
);

create table if not exists collection_fields (
  id uuid primary key default gen_random_uuid(),
  collection_id uuid not null references collections(id) on delete cascade,
  name text not null,
  type text not null,
  required boolean not null default false,
  is_list boolean not null default false,
  reference_collection text not null default '',
  sort_order integer not null default 0,
  unique (collection_id, name)
);

create table if not exists records (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references projects(id) on delete cascade,
  collection_id uuid not null references collections(id) on delete cascade,
  created_by_user_id uuid references users(id),
  data jsonb not null default '{}'::jsonb,
  status text not null default 'active',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists subscriptions (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references projects(id) on delete cascade,
  plan_code text not null,
  status text not null default 'pending',
  starts_at timestamptz,
  ends_at timestamptz,
  payment_note text not null default '',
  created_at timestamptz not null default now()
);

create table if not exists moderation_cases (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references projects(id) on delete cascade,
  reason text not null,
  source text not null default 'system',
  status text not null default 'open',
  created_at timestamptz not null default now()
);

create index if not exists idx_records_project_collection_created_at
  on records (project_id, collection_id, created_at desc);

create index if not exists idx_records_collection_user
  on records (collection_id, created_by_user_id);

create index if not exists idx_records_data_gin
  on records using gin (data);
