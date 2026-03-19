-- Rollback for 0001_video_upload_schema
-- Since this migration is an initial schema, the rollback drops the whole set.

DROP TABLE IF EXISTS video_renditions;
DROP TABLE IF EXISTS upload_steps;
DROP TABLE IF EXISTS uploads;
DROP TABLE IF EXISTS videos;

