-- =============================================================================
-- Maradhi — Complete Database Schema
-- Run this in Supabase SQL Editor (Dashboard → SQL Editor → New Query)
-- =============================================================================
--
-- Design principles:
--   1. All tables use uuid PRIMARY KEY (gen_random_uuid()) — no sequential IDs
--      that are guessable. Supabase has pgcrypto enabled by default.
--   2. Every table has user_id → auth.users(id) ON DELETE CASCADE.
--      Deleting a user in Supabase Auth automatically deletes all their data.
--   3. Row Level Security (RLS) is enabled on every table so users can NEVER
--      access each other's data, even if a backend bug sends the wrong user_id.
--   4. created_at / updated_at use TIMESTAMPTZ (with timezone) — always UTC.
--   5. Enums are Postgres CHECK constraints rather than ENUM types.
--      CHECK constraints can be altered without a table rewrite; ENUM types cannot.


-- =============================================================================
-- EXTENSIONS (Supabase enables these by default, but be explicit)
-- =============================================================================
CREATE EXTENSION IF NOT EXISTS "pgcrypto"; -- gen_random_uuid()


-- =============================================================================
-- PROFILES
-- Supabase handles authentication in auth.users.
-- We create a profiles table in the public schema for app-level user data.
-- A trigger automatically creates a profile when a user signs up.
-- =============================================================================
CREATE TABLE IF NOT EXISTS profiles (
    id          UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    full_name   TEXT,
    avatar_url  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Automatically create a profile row when a new user signs up
CREATE OR REPLACE FUNCTION public.handle_new_user()
RETURNS TRIGGER LANGUAGE plpgsql SECURITY DEFINER AS $$
BEGIN
    INSERT INTO public.profiles (id, full_name)
    VALUES (NEW.id, NEW.raw_user_meta_data->>'full_name')
    ON CONFLICT DO NOTHING;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS on_auth_user_created ON auth.users;
CREATE TRIGGER on_auth_user_created
    AFTER INSERT ON auth.users
    FOR EACH ROW EXECUTE FUNCTION public.handle_new_user();

ALTER TABLE profiles ENABLE ROW LEVEL SECURITY;
CREATE POLICY "users can view own profile"   ON profiles FOR SELECT USING (auth.uid() = id);
CREATE POLICY "users can update own profile" ON profiles FOR UPDATE USING (auth.uid() = id);


-- =============================================================================
-- TAGS
-- User-defined labels with a hex colour, reusable across tasks.
-- =============================================================================
CREATE TABLE IF NOT EXISTS tags (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    color       TEXT        NOT NULL DEFAULT '#888888',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- A user can't have two tags with the same name
    UNIQUE (user_id, name)
);

CREATE INDEX IF NOT EXISTS idx_tags_user_id ON tags(user_id);

ALTER TABLE tags ENABLE ROW LEVEL SECURITY;
CREATE POLICY "users manage own tags" ON tags USING (auth.uid() = user_id);


-- =============================================================================
-- TASKS
-- Core entity. Priority and status are enforced via CHECK constraints.
-- =============================================================================
CREATE TABLE IF NOT EXISTS tasks (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    title       TEXT        NOT NULL CHECK (length(trim(title)) > 0),
    notes       TEXT,
    priority    TEXT        NOT NULL DEFAULT 'medium'
                            CHECK (priority IN ('high', 'medium', 'low')),
    status      TEXT        NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending', 'done', 'draft')),
    due_date    TIMESTAMPTZ,          -- NULL = no deadline
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for the most common query patterns
CREATE INDEX IF NOT EXISTS idx_tasks_user_id     ON tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status      ON tasks(user_id, status);
CREATE INDEX IF NOT EXISTS idx_tasks_due_date    ON tasks(user_id, due_date);
CREATE INDEX IF NOT EXISTS idx_tasks_priority    ON tasks(user_id, priority);

-- Auto-update updated_at on every row modification
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$;

CREATE TRIGGER tasks_updated_at
    BEFORE UPDATE ON tasks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

ALTER TABLE tasks ENABLE ROW LEVEL SECURITY;
CREATE POLICY "users manage own tasks" ON tasks USING (auth.uid() = user_id);


-- =============================================================================
-- TASK_TAGS  (many-to-many join table)
-- A task can have many tags; a tag can apply to many tasks.
-- ON DELETE CASCADE means removing a task or tag cleans up automatically.
-- =============================================================================
CREATE TABLE IF NOT EXISTS task_tags (
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    tag_id  UUID NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
    PRIMARY KEY (task_id, tag_id)   -- composite PK prevents duplicates
);

CREATE INDEX IF NOT EXISTS idx_task_tags_tag_id ON task_tags(tag_id);

ALTER TABLE task_tags ENABLE ROW LEVEL SECURITY;
-- Users can only manage task_tags for their own tasks
CREATE POLICY "users manage own task_tags" ON task_tags
    USING (
        EXISTS (SELECT 1 FROM tasks WHERE tasks.id = task_id AND tasks.user_id = auth.uid())
    );


-- =============================================================================
-- BUCKET_LIST_ITEMS
-- Life goals. completed_at is set when is_done flips to true.
-- =============================================================================
CREATE TABLE IF NOT EXISTS bucket_list_items (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    title        TEXT        NOT NULL CHECK (length(trim(title)) > 0),
    description  TEXT,
    category     TEXT        NOT NULL DEFAULT 'general',
                             -- e.g. travel, skills, adventure, fitness, personal
    is_done      BOOLEAN     NOT NULL DEFAULT FALSE,
    completed_at TIMESTAMPTZ,         -- NULL until marked done
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bucket_user_id  ON bucket_list_items(user_id);
CREATE INDEX IF NOT EXISTS idx_bucket_category ON bucket_list_items(user_id, category);

ALTER TABLE bucket_list_items ENABLE ROW LEVEL SECURITY;
CREATE POLICY "users manage own bucket items" ON bucket_list_items USING (auth.uid() = user_id);


-- =============================================================================
-- NOTES
-- Freeform personal notes with a simple tag label (not FK, just text).
-- For full-text search we add a GIN index on a tsvector column.
-- =============================================================================
CREATE TABLE IF NOT EXISTS notes (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    title       TEXT        NOT NULL CHECK (length(trim(title)) > 0),
    content     TEXT        NOT NULL DEFAULT '',
    tag_label   TEXT        NOT NULL DEFAULT 'personal',
                             -- simple string label: work | ideas | personal | ...
    -- Full-text search vector, auto-updated by trigger below
    search_vec  TSVECTOR,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notes_user_id   ON notes(user_id);
CREATE INDEX IF NOT EXISTS idx_notes_tag_label ON notes(user_id, tag_label);
-- GIN index enables fast full-text search
CREATE INDEX IF NOT EXISTS idx_notes_search    ON notes USING GIN(search_vec);

-- Keep search_vec in sync with title + content
CREATE OR REPLACE FUNCTION notes_search_vec_update()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.search_vec := to_tsvector('english', COALESCE(NEW.title, '') || ' ' || COALESCE(NEW.content, ''));
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$;

CREATE TRIGGER notes_search_vec_trigger
    BEFORE INSERT OR UPDATE ON notes
    FOR EACH ROW EXECUTE FUNCTION notes_search_vec_update();

ALTER TABLE notes ENABLE ROW LEVEL SECURITY;
CREATE POLICY "users manage own notes" ON notes USING (auth.uid() = user_id);


-- =============================================================================
-- HABITS
-- Recurring daily habits the user wants to track.
-- The actual completion data lives in habit_logs.
-- =============================================================================
CREATE TABLE IF NOT EXISTS habits (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL CHECK (length(trim(name)) > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (user_id, name)  -- no duplicate habit names per user
);

CREATE INDEX IF NOT EXISTS idx_habits_user_id ON habits(user_id);

ALTER TABLE habits ENABLE ROW LEVEL SECURITY;
CREATE POLICY "users manage own habits" ON habits USING (auth.uid() = user_id);


-- =============================================================================
-- HABIT_LOGS
-- One row per (habit, user, date) = "I did this habit on this day".
-- The composite UNIQUE constraint prevents double-logging.
-- =============================================================================
CREATE TABLE IF NOT EXISTS habit_logs (
    id           UUID  PRIMARY KEY DEFAULT gen_random_uuid(),
    habit_id     UUID  NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    user_id      UUID  NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    logged_date  DATE  NOT NULL DEFAULT CURRENT_DATE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- One log per habit per day (upsert uses ON CONFLICT on this)
    UNIQUE (habit_id, user_id, logged_date)
);

CREATE INDEX IF NOT EXISTS idx_habit_logs_habit_id    ON habit_logs(habit_id);
CREATE INDEX IF NOT EXISTS idx_habit_logs_user_date   ON habit_logs(user_id, logged_date);

ALTER TABLE habit_logs ENABLE ROW LEVEL SECURITY;
CREATE POLICY "users manage own habit logs" ON habit_logs USING (auth.uid() = user_id);


-- =============================================================================
-- FOCUS_SESSIONS
-- Records completed Pomodoro/deep-work sessions.
-- =============================================================================
CREATE TABLE IF NOT EXISTS focus_sessions (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    duration_minutes INTEGER     NOT NULL CHECK (duration_minutes > 0),
    task_note        TEXT,                 -- what the user was working on
    started_at       TIMESTAMPTZ NOT NULL,
    ended_at         TIMESTAMPTZ NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Sanity check: session can't end before it starts
    CHECK (ended_at >= started_at)
);

CREATE INDEX IF NOT EXISTS idx_focus_user_id    ON focus_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_focus_started_at ON focus_sessions(user_id, started_at);

ALTER TABLE focus_sessions ENABLE ROW LEVEL SECURITY;
CREATE POLICY "users manage own focus sessions" ON focus_sessions USING (auth.uid() = user_id);


-- =============================================================================
-- MOOD_LOGS
-- Simple mood entry with an optional note.
-- =============================================================================
CREATE TABLE IF NOT EXISTS mood_logs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    mood        TEXT        NOT NULL
                            CHECK (mood IN ('great', 'good', 'okay', 'low')),
    note        TEXT,
    logged_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mood_user_id   ON mood_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_mood_logged_at ON mood_logs(user_id, logged_at);

ALTER TABLE mood_logs ENABLE ROW LEVEL SECURITY;
CREATE POLICY "users manage own mood logs" ON mood_logs USING (auth.uid() = user_id);


-- =============================================================================
-- DONE — your Maradhi schema is ready.
-- Next steps:
--   1. Paste this into Supabase SQL Editor and run it
--   2. Enable Email auth in Supabase Dashboard → Authentication → Providers
--   3. Copy your JWT secret from Dashboard → Settings → API
--   4. Update your .env file and run: go run ./cmd/api
-- =============================================================================
