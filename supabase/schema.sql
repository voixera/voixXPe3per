-- PeeperPhone cloud schema
-- Run once in Supabase Dashboard -> SQL Editor.

create table if not exists public.pairing_sessions (
  code         text primary key,
  host_user_id uuid not null,
  host_name    text not null default '',
  status       text not null default 'waiting',
  device       jsonb,
  approved_at  timestamptz,
  created_at   timestamptz not null default now()
);

alter table public.pairing_sessions enable row level security;

drop policy if exists "session insert by owner" on public.pairing_sessions;
create policy "session insert by owner"
  on public.pairing_sessions for insert to authenticated
  with check (host_user_id = auth.uid());

-- Anyone authenticated who knows the exact room code can read/approve it.
drop policy if exists "session read authenticated" on public.pairing_sessions;
create policy "session read authenticated"
  on public.pairing_sessions for select to authenticated
  using (true);

drop policy if exists "session update authenticated" on public.pairing_sessions;
create policy "session update authenticated"
  on public.pairing_sessions for update to authenticated
  using (true)
  with check (true);
