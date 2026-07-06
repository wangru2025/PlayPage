create table if not exists project_notification_events (
  project_id uuid not null references projects(id) on delete cascade,
  event_key text not null,
  created_at timestamptz not null default now(),
  primary key(project_id, event_key)
);

create index if not exists project_notification_events_created_idx on project_notification_events(created_at desc);
