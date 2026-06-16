create table if not exists reserved_subdomains (
  id uuid primary key default gen_random_uuid(),
  subdomain text not null unique,
  reason text not null default '',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists project_domains (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references projects(id) on delete cascade,
  owner_user_id uuid not null references users(id) on delete cascade,
  subdomain text not null,
  domain text not null unique,
  type text not null default 'platform_subdomain',
  status text not null default 'pending',
  reject_reason text not null default '',
  admin_note text not null default '',
  reviewed_by_user_id uuid references users(id),
  reviewed_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists idx_project_domains_project_id on project_domains(project_id);
create index if not exists idx_project_domains_status on project_domains(status, created_at desc);

create unique index if not exists idx_project_domains_one_open_per_project
  on project_domains(project_id)
  where status in ('pending', 'active');

create unique index if not exists idx_project_domains_one_open_per_subdomain
  on project_domains(subdomain)
  where status in ('pending', 'active');

create unique index if not exists idx_upgrade_requests_one_pending_per_user
  on upgrade_requests(user_id)
  where status = 'pending';

insert into reserved_subdomains (subdomain, reason)
values
('www', '平台保留'),
('web', '主站'),
('game', '已有站点'),
('a11y', '已有站点'),
('save', '已有站点'),
('fengsheng', '已有站点'),
('fenshon', '已有站点'),
('ts', '已有或预留站点'),
('demo', '已有或预留站点'),
('api', '平台接口预留'),
('admin', '平台管理预留'),
('mail', '邮件服务预留'),
('smtp', '邮件服务预留'),
('imap', '邮件服务预留'),
('pop', '邮件服务预留'),
('cdn', '静态资源预留'),
('static', '静态资源预留'),
('assets', '静态资源预留'),
('upload', '上传服务预留'),
('uploads', '上传服务预留'),
('download', '下载服务预留'),
('downloads', '下载服务预留'),
('auth', '登录服务预留'),
('login', '登录服务预留'),
('pay', '支付服务预留'),
('payment', '支付服务预留'),
('billing', '账单服务预留'),
('support', '支持服务预留'),
('help', '帮助服务预留'),
('docs', '文档服务预留'),
('status', '状态页预留'),
('monitor', '监控预留'),
('dev', '开发环境预留'),
('test', '测试环境预留'),
('staging', '测试环境预留'),
('prod', '生产环境预留'),
('production', '生产环境预留'),
('localhost', '本地地址预留')
on conflict (subdomain) do nothing;
