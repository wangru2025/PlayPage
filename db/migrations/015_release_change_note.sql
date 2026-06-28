alter table project_releases
  add column if not exists change_note text not null default '';
