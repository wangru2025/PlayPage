alter table upgrade_requests
  add column if not exists system_note text not null default '';
