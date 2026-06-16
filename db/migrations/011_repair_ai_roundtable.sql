create extension if not exists pgcrypto;

create table if not exists repair_ai_jobs (
  id uuid primary key default gen_random_uuid(),
  repair_request_id uuid not null references repair_requests(id) on delete cascade,
  project_id uuid not null references projects(id) on delete cascade,
  owner_user_id uuid not null references users(id) on delete cascade,
  status text not null default 'running',
  round integer not null default 1,
  feedback text not null default '',
  generated_html text not null default '',
  preview_url text not null default '',
  error_message text not null default '',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  finished_at timestamptz
);

create index if not exists idx_repair_ai_jobs_request_created on repair_ai_jobs(repair_request_id, created_at desc);
create index if not exists idx_repair_ai_jobs_user_status on repair_ai_jobs(owner_user_id, status);
create index if not exists idx_repair_ai_jobs_user_created on repair_ai_jobs(owner_user_id, created_at desc);

create table if not exists repair_ai_messages (
  id uuid primary key default gen_random_uuid(),
  job_id uuid not null references repair_ai_jobs(id) on delete cascade,
  repair_request_id uuid not null references repair_requests(id) on delete cascade,
  project_id uuid not null references projects(id) on delete cascade,
  owner_user_id uuid not null references users(id) on delete cascade,
  agent_key text not null default '',
  agent_name text not null default '',
  role text not null default 'assistant',
  visibility text not null default 'user',
  message_type text not null default 'text',
  content text not null default '',
  metadata_json text not null default '',
  message_seq integer not null,
  created_at timestamptz not null default now()
);

create index if not exists idx_repair_ai_messages_job_seq on repair_ai_messages(job_id, message_seq);
