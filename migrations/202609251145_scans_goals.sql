-- Scans feature: scans and goals tables, scoped by user_id with RLS
-- enforcing that a user can only touch their own rows.

create table if not exists scans (
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references auth.users(id) on delete cascade,
    date date not null,
    weight numeric,
    body_fat numeric,
    smm numeric,
    visceral_fat numeric,
    lean_left_arm numeric,
    lean_right_arm numeric,
    lean_trunk numeric,
    lean_left_leg numeric,
    lean_right_leg numeric,
    notes text,
    report_photo_path text,
    progress_photo_path text,
    created_at timestamptz not null default now()
);

create index if not exists scans_user_id_date_idx on scans (user_id, date desc);

alter table scans enable row level security;

create policy "scans_select_own" on scans
    for select using (user_id = auth.uid());

create policy "scans_insert_own" on scans
    for insert with check (user_id = auth.uid());

create policy "scans_update_own" on scans
    for update using (user_id = auth.uid()) with check (user_id = auth.uid());

create policy "scans_delete_own" on scans
    for delete using (user_id = auth.uid());

create table if not exists goals (
    user_id uuid primary key references auth.users(id) on delete cascade,
    weight numeric,
    body_fat numeric,
    smm numeric
);

alter table goals enable row level security;

create policy "goals_select_own" on goals
    for select using (user_id = auth.uid());

create policy "goals_insert_own" on goals
    for insert with check (user_id = auth.uid());

create policy "goals_update_own" on goals
    for update using (user_id = auth.uid()) with check (user_id = auth.uid());

create policy "goals_delete_own" on goals
    for delete using (user_id = auth.uid());
