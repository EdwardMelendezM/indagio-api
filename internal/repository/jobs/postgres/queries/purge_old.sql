DELETE FROM background_jobs
WHERE created_at < NOW() - ($1 || ' days')::INTERVAL
  AND status IN ('completed', 'failed')