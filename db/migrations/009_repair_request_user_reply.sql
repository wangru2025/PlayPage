alter table repair_requests
  add column if not exists user_reply text not null default '',
  add column if not exists user_replied_at timestamptz;
