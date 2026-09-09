UPDATE background_jobs SET
    status     = CASE WHEN attempts >= max_attempts THEN 'failed' ELSE 'pending' END,
    run_at     = CASE WHEN attempts >= max_attempts
                     THEN run_at
                     ELSE NOW() + (INTERVAL '5 minutes' * power(2, attempts - 1))
                 END,
    error_log  = $2,
    updated_at = NOW()
WHERE id = $1