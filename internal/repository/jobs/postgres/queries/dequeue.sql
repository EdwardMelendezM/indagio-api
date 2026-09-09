UPDATE background_jobs
SET
    status     = 'processing',
    attempts   = attempts + 1,
    updated_at = NOW()
WHERE id = (
    SELECT id FROM background_jobs
    WHERE status  = 'pending'
      AND run_at <= NOW()
      AND type   = ANY($1)
    ORDER BY run_at ASC
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING id, type, payload, attempts, max_attempts, run_at, created_at