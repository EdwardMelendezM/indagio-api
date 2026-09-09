UPDATE background_jobs
SET status='completed', updated_at=NOW()
WHERE id=$1