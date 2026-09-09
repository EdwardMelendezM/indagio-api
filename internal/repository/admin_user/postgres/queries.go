package admin_users_postgres

import _ "embed"

//go:embed queries/find_by_email.sql
var queryFindByEmail string

//go:embed queries/find_by_id.sql
var queryFindById string

//go:embed queries/mark_verified.sql
var queryMarkVerified string

//go:embed queries/create.sql
var queryCreate string

//go:embed queries/find_first.sql
var queryFindFirst string
