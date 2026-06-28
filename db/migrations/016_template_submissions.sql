create table if not exists template_submissions (
  id uuid primary key default gen_random_uuid(),
  author_user_id uuid not null references users(id) on delete cascade,
  slug text not null unique,
  name text not null,
  category text not null default 'community',
  category_label text not null default '用户投稿',
  summary text not null default '',
  description text not null default '',
  tags jsonb not null default '[]'::jsonb,
  interactive_required boolean not null default false,
  analytics_recommended boolean not null default true,
  config_fields jsonb not null default '[]'::jsonb,
  collections jsonb not null default '[]'::jsonb,
  html_source text not null,
  source_type text not null default 'text',
  status text not null default 'pending',
  admin_note text not null default '',
  reviewed_by_user_id uuid references users(id),
  reviewed_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists idx_template_submissions_status_created
  on template_submissions (status, created_at desc);

create index if not exists idx_template_submissions_author_created
  on template_submissions (author_user_id, created_at desc);
