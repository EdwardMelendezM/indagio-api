package postgres

import _ "embed"

//go:embed queries/find_by_email.sql
var queryFindUserByEmail string

//go:embed queries/find_by_id.sql
var queryFindUserByID string

//go:embed queries/create.sql
var queryCreateUser string

//go:embed queries/mark_verified.sql
var queryMarkUserVerified string

//go:embed queries/update_name.sql
var queryUpdateUserName string

//go:embed queries/update_password.sql
var queryUpdateUserPassword string

//go:embed queries/block_user.sql
var queryBlockUser string

//go:embed queries/list_available.sql
var queryListAvailableUsers string

//go:embed queries/count_available.sql
var queryCountAvailableUsers string

//go:embed queries/list_all.sql
var queryListAllUsers string

//go:embed queries/count_all.sql
var queryCountAllUsers string

//go:embed queries/update_avatar_url.sql
var queryUpdateUserAvatarURL string

//go:embed queries/clear_avatar.sql
var queryClearUserAvatar string
