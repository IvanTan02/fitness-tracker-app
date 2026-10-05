-- Compact manual tracking: retain only metrics needed to monitor fat loss
-- while preserving skeletal muscle.
create table if not exists profiles (
    user_id uuid primary key references auth.users(id) on delete cascade,
    height_cm numeric not null,
    gender text not null,
    age integer not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

alter table profiles enable row level security;
create policy "profiles_select_own" on profiles for select to authenticated using ((select auth.uid()) = user_id);
create policy "profiles_insert_own" on profiles for insert to authenticated with check ((select auth.uid()) = user_id);
create policy "profiles_update_own" on profiles for update to authenticated using ((select auth.uid()) = user_id) with check ((select auth.uid()) = user_id);
create policy "profiles_delete_own" on profiles for delete to authenticated using ((select auth.uid()) = user_id);

alter table scans
    drop column if exists lean_body_mass,
    drop column if exists body_fat_mass,
    drop column if exists visceral_fat,
    drop column if exists bmr,
    drop column if exists tee,
    drop column if exists lean_left_arm,
    drop column if exists lean_right_arm,
    drop column if exists lean_trunk,
    drop column if exists lean_left_leg,
    drop column if exists lean_right_leg,
    drop column if exists fat_left_arm,
    drop column if exists fat_right_arm,
    drop column if exists fat_trunk,
    drop column if exists fat_left_leg,
    drop column if exists fat_right_leg,
    drop column if exists report_photo_path,
    drop column if exists progress_photo_path;
