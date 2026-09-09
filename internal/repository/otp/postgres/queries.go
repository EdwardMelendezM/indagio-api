package postgres

import _ "embed"

//go:embed queries/create.sql
var queryCreateOTP string

//go:embed queries/find_active.sql
var queryFindActiveOTP string

//go:embed queries/consume.sql
var queryConsumeOTP string
