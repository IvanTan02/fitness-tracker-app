-- Isolate development and production application data while continuing to
-- share one Supabase project and Auth user store.

begin;

alter table profiles
    add column if not exists environment text not null default 'production';

alter table goals
    add column if not exists environment text not null default 'production';

alter table scans
    add column if not exists environment text not null default 'production';

do $$
begin
    if not exists (
        select 1 from pg_constraint
        where conname = 'profiles_environment_check'
          and conrelid = 'public.profiles'::regclass
    ) then
        alter table profiles
            add constraint profiles_environment_check
            check (environment in ('development', 'production'));
    end if;

    if not exists (
        select 1 from pg_constraint
        where conname = 'goals_environment_check'
          and conrelid = 'public.goals'::regclass
    ) then
        alter table goals
            add constraint goals_environment_check
            check (environment in ('development', 'production'));
    end if;

    if not exists (
        select 1 from pg_constraint
        where conname = 'scans_environment_check'
          and conrelid = 'public.scans'::regclass
    ) then
        alter table scans
            add constraint scans_environment_check
            check (environment in ('development', 'production'));
    end if;
end $$;

alter table profiles drop constraint if exists profiles_pkey;
alter table profiles add primary key (user_id, environment);

alter table goals drop constraint if exists goals_pkey;
alter table goals add primary key (user_id, environment);

drop index if exists scans_user_id_date_idx;
create index if not exists scans_user_id_environment_date_idx
    on scans (user_id, environment, date desc);

commit;
