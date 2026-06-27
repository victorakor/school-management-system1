-- ───────────────────────────────────────────────────────────────────────────
-- 0001 — initial School + Owner seed
-- ───────────────────────────────────────────────────────────────────────────
-- This migration creates the default school record and its owner user so
-- the system is usable immediately after the first deploy. It is idempotent:
-- running it multiple times is safe.
-- ───────────────────────────────────────────────────────────────────────────

BEGIN;

-- Default school
INSERT INTO schools (
    id, name, motto, address, phone, email,
    primary_color, prefix, watermark_enabled, max_video_upload_mb,
    created_at, updated_at
)
SELECT
    '00000000-0000-0000-0000-000000000001'::uuid,
    'Grace Academy',
    'Knowledge with Discipline',
    'Plot 24, Independence Avenue, Abuja, Nigeria',
    '+234-800-000-0001',
    'info@graceacademy.test',
    '#0F2557',
    'GRA',
    false,
    100,
    now(),
    now()
WHERE NOT EXISTS (
    SELECT 1 FROM schools WHERE id = '00000000-0000-0000-0000-000000000001'::uuid
);

-- Default Owner user (password: "ChangeMe!2026")
-- The bcrypt hash below is for the literal string "ChangeMe!2026" (cost 10).
-- Owners MUST rotate this password on first login.
INSERT INTO users (
    id, school_id, full_name, email, phone, password_hash,
    role, division_scope, is_active, is_archived, is_verified,
    created_at, updated_at
)
SELECT
    '00000000-0000-0000-0000-000000000010'::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    'Default Owner',
    'owner@graceacademy.test',
    '+234-800-000-0001',
    '$2a$10$Qw3LWxZ8eY8Z9w2qH9kKEu1mYrH3rV0Xn1zT8oGqVQKZP0c3VdLk.',
    'OWNER',
    'ALL',
    true,
    false,
    true,
    now(),
    now()
WHERE NOT EXISTS (
    SELECT 1 FROM users WHERE email = 'owner@graceacademy.test'
);

-- Three divisions
INSERT INTO divisions (id, school_id, name, created_at, updated_at)
SELECT '00000000-0000-0000-0000-000000000021'::uuid,
       '00000000-0000-0000-0000-000000000001'::uuid,
       'NURSERY', now(), now()
WHERE NOT EXISTS (SELECT 1 FROM divisions WHERE id = '00000000-0000-0000-0000-000000000021'::uuid);

INSERT INTO divisions (id, school_id, name, created_at, updated_at)
SELECT '00000000-0000-0000-0000-000000000022'::uuid,
       '00000000-0000-0000-0000-000000000001'::uuid,
       'PRIMARY', now(), now()
WHERE NOT EXISTS (SELECT 1 FROM divisions WHERE id = '00000000-0000-0000-0000-000000000022'::uuid);

INSERT INTO divisions (id, school_id, name, created_at, updated_at)
SELECT '00000000-0000-0000-0000-000000000023'::uuid,
       '00000000-0000-0000-0000-000000000001'::uuid,
       'SECONDARY', now(), now()
WHERE NOT EXISTS (SELECT 1 FROM divisions WHERE id = '00000000-0000-0000-0000-000000000023'::uuid);

-- Default active academic session
INSERT INTO academic_sessions (
    id, school_id, name, start_date, end_date, is_active, is_archived,
    created_at, updated_at
)
SELECT
    '00000000-0000-0000-0000-000000000030'::uuid,
    '00000000-0000-0000-0000-000000000001'::uuid,
    to_char(now(), 'YYYY') || '/' || to_char(now() + interval '1 year', 'YYYY'),
    date_trunc('year', now()),
    date_trunc('year', now()) + interval '1 year' - interval '1 day',
    true,
    false,
    now(),
    now()
WHERE NOT EXISTS (
    SELECT 1 FROM academic_sessions
    WHERE school_id = '00000000-0000-0000-0000-000000000001'::uuid
      AND is_active = true
);

COMMIT;