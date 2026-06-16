create table if not exists upgrade_requests (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  current_plan text not null,
  target_plan text not null,
  payment_method text not null,
  payer_note text not null default '',
  status text not null default 'pending',
  admin_note text not null default '',
  reviewed_by_user_id uuid references users(id),
  reviewed_at timestamptz,
  created_at timestamptz not null default now()
);

create index if not exists idx_upgrade_requests_status_created_at
  on upgrade_requests (status, created_at desc);
