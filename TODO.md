# Video Upload Platform — Post-Workers TODO

Finish workers (metadata, transcode, thumbnail, explicit-content classification) first. Then implement the following.

---

## 1. Handle "client never sends completion" (pre-signed upload flow)

- [ ] **Pending upload lifecycle**
  - When issuing pre-signed URL, create a pending upload record (`upload_id`, `user_id`, `created_at`, `status = "pending"`).
  - Use short-lived pre-signed URLs (e.g. 15–60 min).

- [ ] **Timeout + cleanup job**
  - Background job (cron/scheduler) that finds pending uploads older than threshold (e.g. 1–2 hours) never finalized.
  - Mark them as `expired` / `abandoned`.
  - Optionally delete orphan object from S3 if it was uploaded but never finalized.

- [ ] **(Optional)** Use S3 Event Notifications to detect uploads to staging path and support auto-finalize or prompts.

---

## 2. Orchestration for multi-step pipeline

- [ ] **DB as state store**
  - Table for pipeline steps (e.g. `video_processing_jobs` or `video_pipeline_steps`): `video_id`, `step` (metadata, transcode, thumbnail, moderation), `status` (pending | running | done | failed), `updated_at`, optional `attempt`, `error_message`.

- [ ] **Upload service**
  - On finalize: insert one row per step (all `pending`) and publish "video ready for processing" (or one event per step).

- [ ] **Workers**
  - On finish: update DB (`status = done` or `failed`) and optionally publish "step X done for video Y".

- [ ] **Orchestrator**
  - Small service or logic in upload service that subscribes to "step done" (or polls DB).
  - If all required steps `done` → mark video `ready`, notify.
  - If any step `failed` → mark video `processing_failed`, trigger retries/alerts.

---

## 3. Idempotent workers (job status in DB)

- [ ] **Job identity**
  - Stable job ID per run: e.g. `(video_id, step)` or unique `job_id`.

- [ ] **DB schema**
  - Store: `job_id` (or `video_id` + `step`), `status` (pending | running | done | failed), `updated_at`, optional `attempt`, `error_message`, `locked_by` / `worker_id`.

- [ ] **Claim-before-process**
  - On message: `UPDATE ... SET status = 'running', locked_by = $worker_id WHERE ... AND status = 'pending' RETURNING *`.
  - If no row updated → ack message (duplicate or already taken).
  - If row updated → do work, then `UPDATE status = 'done'` (or `failed`), then ack.

- [ ] **Stuck-job handling**
  - If `status = 'running'` and `updated_at` very old, treat as stuck and allow retry (e.g. set back to `pending` or dedicated retry path).
