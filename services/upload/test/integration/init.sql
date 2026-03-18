-- Video upload pipeline schema (matches scripts/create_tables.sql)
CREATE TABLE IF NOT EXISTS videos (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    title           VARCHAR(512) NOT NULL DEFAULT '',
    format          VARCHAR(64),
    thumbnail_path  VARCHAR(1024),
    status          VARCHAR(32) NOT NULL DEFAULT 'processing',
    duration_sec    NUMERIC(12, 2),
    hls_master_path VARCHAR(1024),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_videos_user_id ON videos (user_id);
CREATE INDEX IF NOT EXISTS idx_videos_status ON videos (status);
CREATE INDEX IF NOT EXISTS idx_videos_created_at ON videos (created_at);
CREATE INDEX IF NOT EXISTS idx_videos_deleted_at ON videos (deleted_at) WHERE deleted_at IS NULL;

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

CREATE TABLE IF NOT EXISTS video_renditions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id        UUID NOT NULL REFERENCES videos (id) ON DELETE CASCADE,
    resolution      VARCHAR(32) NOT NULL,
    storage_path    VARCHAR(1024),
    format          VARCHAR(16),
    width           INT,
    height          INT,
    bitrate_kbps    INT,
    status          VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (video_id, resolution)
);

CREATE INDEX IF NOT EXISTS idx_video_renditions_video_id ON video_renditions (video_id);
CREATE INDEX IF NOT EXISTS idx_video_renditions_status ON video_renditions (video_id, status) WHERE status = 'pending';
