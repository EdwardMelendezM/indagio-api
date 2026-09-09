package postgres

import _ "embed"

//go:embed queries/enqueue.sql
var queryEnqueue string

//go:embed queries/dequeue.sql
var queryDequeue string

//go:embed queries/mark_completed.sql
var queryMarkCompleted string

//go:embed queries/mark_failed.sql
var queryMarkFailed string

//go:embed queries/purge_old.sql
var queryPurgeOld string
