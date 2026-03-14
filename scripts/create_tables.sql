-- Video upload pipeline schema (PostgreSQL)
-- Run this against your database to create the tables.

-- Videos: main catalog entity. No storage_path; playback paths are in video_renditions.
CREATE TABLE IF NOT EXISTS videos (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    title           VARCHAR(512) NOT NULL DEFAULT '',
    format          VARCHAR(64),
    status          VARCHAR(32) NOT NULL DEFAULT 'processing',
    duration_sec    NUMERIC(12, 2),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_videos_user_id ON videos (user_id);
CREATE INDEX IF NOT EXISTS idx_videos_status ON videos (status);
CREATE INDEX IF NOT EXISTS idx_videos_created_at ON videos (created_at);
CREATE INDEX IF NOT EXISTS idx_videos_deleted_at ON videos (deleted_at) WHERE deleted_at IS NULL;

-- Uploads: pipeline run per video. Holds path to the original uploaded file (source for workers and for the source-resolution rendition).
CREATE TABLE IF NOT EXISTS uploads (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id        UUID NOT NULL REFERENCES videos (id) ON DELETE RESTRICT,
    storage_path    VARCHAR(1024),
    status          VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_uploads_video_id ON uploads (video_id);
CREATE INDEX IF NOT EXISTS idx_uploads_status ON uploads (status);
CREATE INDEX IF NOT EXISTS idx_uploads_expires_at ON uploads (expires_at) WHERE expires_at IS NOT NULL;

-- Upload steps: one row per step per upload. Workers claim and update these; orchestrator checks when all are done.
CREATE TABLE IF NOT EXISTS upload_steps (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id       UUID NOT NULL REFERENCES uploads (id) ON DELETE CASCADE,
    step            VARCHAR(64) NOT NULL,
    status          VARCHAR(32) NOT NULL DEFAULT 'pending',
    error_message   TEXT,
    attempt         INT NOT NULL DEFAULT 1,
    locked_by       VARCHAR(256),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (upload_id, step)
);

CREATE INDEX IF NOT EXISTS idx_upload_steps_upload_id ON upload_steps (upload_id);
CREATE INDEX IF NOT EXISTS idx_upload_steps_status ON upload_steps (upload_id, status);

-- Video renditions: one row per resolution/quality. Includes source resolution (e.g. 1080p) pointing to same path as uploads.storage_path; others are transcoded outputs.
CREATE TABLE IF NOT EXISTS video_renditions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id        UUID NOT NULL REFERENCES videos (id) ON DELETE CASCADE,
    resolution      VARCHAR(32) NOT NULL,
    storage_path    VARCHAR(1024) NOT NULL,
    format          VARCHAR(16),
    width           INT,
    height          INT,
    bitrate_kbps    INT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (video_id, resolution)
);

CREATE INDEX IF NOT EXISTS idx_video_renditions_video_id ON video_renditions (video_id);
