alter table projects
  add column if not exists interactive boolean not null default false;
