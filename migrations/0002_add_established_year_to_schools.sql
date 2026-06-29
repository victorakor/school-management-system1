-- Migration: 0002_add_established_year_to_schools
-- Adds an EstablishedYear column to the schools table so the About page
-- can display a live "Years of Excellence" counter derived from the DB.

ALTER TABLE schools
  ADD COLUMN IF NOT EXISTS established_year INTEGER NOT NULL DEFAULT 0;

COMMENT ON COLUMN schools.established_year IS
  'Calendar year the school was founded (e.g. 2010). 0 = not set; frontend falls back to a default.';
