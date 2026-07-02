
2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:48
[0m[33m[0.060ms] [34;1m[rows:-][0m SELECT count(*) FROM sqlite_master WHERE type='table' AND name="users"

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:48
[0m[33m[0.126ms] [34;1m[rows:2][0m SELECT sql FROM sqlite_master WHERE type IN ("table","index") AND tbl_name = "users" AND sql IS NOT NULL order by type = "table" desc

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:48
[0m[33m[0.043ms] [34;1m[rows:-][0m SELECT * FROM `users` LIMIT 1

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:48
[0m[33m[0.019ms] [34;1m[rows:-][0m SELECT count(*) FROM sqlite_master WHERE type = "index" AND tbl_name = "users" AND name = "idx_users_deleted_at"

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:48
[0m[33m[0.010ms] [34;1m[rows:-][0m SELECT count(*) FROM sqlite_master WHERE type='table' AND name="auths"

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:48
[0m[33m[0.049ms] [34;1m[rows:2][0m SELECT sql FROM sqlite_master WHERE type IN ("table","index") AND tbl_name = "auths" AND sql IS NOT NULL order by type = "table" desc

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:48
[0m[33m[0.012ms] [34;1m[rows:-][0m SELECT * FROM `auths` LIMIT 1

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:48
[0m[33m[0.011ms] [34;1m[rows:-][0m SELECT count(*) FROM sqlite_master WHERE type = "index" AND tbl_name = "auths" AND name = "idx_auths_deleted_at"

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:48
[0m[33m[0.010ms] [34;1m[rows:-][0m SELECT count(*) FROM sqlite_master WHERE type='table' AND name="user_auths"

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:48
[0m[33m[0.036ms] [34;1m[rows:1][0m SELECT sql FROM sqlite_master WHERE type IN ("table","index") AND tbl_name = "user_auths" AND sql IS NOT NULL order by type = "table" desc

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:48
[0m[33m[0.012ms] [34;1m[rows:-][0m SELECT * FROM `user_auths` LIMIT 1

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:48
[0m[33m[0.044ms] [34;1m[rows:-][0m SELECT count(*) FROM sqlite_master WHERE type = "table" AND tbl_name = "user_auths" AND (sql LIKE "%CONSTRAINT \"fk_user_auths_user\" %" OR sql LIKE "%CONSTRAINT fk_user_auths_user %" OR sql LIKE "%CONSTRAINT `fk_user_auths_user`%" OR sql LIKE "%CONSTRAINT [fk_user_auths_user]%" OR sql LIKE "%CONSTRAINT 	fk_user_auths_user	%")

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:48
[0m[33m[0.014ms] [34;1m[rows:-][0m SELECT count(*) FROM sqlite_master WHERE type = "table" AND tbl_name = "user_auths" AND (sql LIKE "%CONSTRAINT \"fk_user_auths_auth\" %" OR sql LIKE "%CONSTRAINT fk_user_auths_auth %" OR sql LIKE "%CONSTRAINT `fk_user_auths_auth`%" OR sql LIKE "%CONSTRAINT [fk_user_auths_auth]%" OR sql LIKE "%CONSTRAINT 	fk_user_auths_auth	%")

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:49
[0m[33m[0.009ms] [34;1m[rows:-][0m SELECT count(*) FROM sqlite_master WHERE type='table' AND name="auths"

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:49
[0m[33m[0.036ms] [34;1m[rows:2][0m SELECT sql FROM sqlite_master WHERE type IN ("table","index") AND tbl_name = "auths" AND sql IS NOT NULL order by type = "table" desc

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:49
[0m[33m[0.010ms] [34;1m[rows:-][0m SELECT * FROM `auths` LIMIT 1

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:49
[0m[33m[0.010ms] [34;1m[rows:-][0m SELECT count(*) FROM sqlite_master WHERE type = "index" AND tbl_name = "auths" AND name = "idx_auths_deleted_at"

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:49
[0m[33m[0.008ms] [34;1m[rows:-][0m SELECT count(*) FROM sqlite_master WHERE type='table' AND name="users"

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:49
[0m[33m[0.036ms] [34;1m[rows:2][0m SELECT sql FROM sqlite_master WHERE type IN ("table","index") AND tbl_name = "users" AND sql IS NOT NULL order by type = "table" desc

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:49
[0m[33m[0.009ms] [34;1m[rows:-][0m SELECT * FROM `users` LIMIT 1

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:49
[0m[33m[0.009ms] [34;1m[rows:-][0m SELECT count(*) FROM sqlite_master WHERE type = "index" AND tbl_name = "users" AND name = "idx_users_deleted_at"

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:49
[0m[33m[0.008ms] [34;1m[rows:-][0m SELECT count(*) FROM sqlite_master WHERE type='table' AND name="user_auth"

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:49
[0m[33m[0.029ms] [34;1m[rows:1][0m SELECT sql FROM sqlite_master WHERE type IN ("table","index") AND tbl_name = "user_auth" AND sql IS NOT NULL order by type = "table" desc

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:49
[0m[33m[0.008ms] [34;1m[rows:-][0m SELECT * FROM `user_auth` LIMIT 1

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:49
[0m[33m[0.014ms] [34;1m[rows:-][0m SELECT count(*) FROM sqlite_master WHERE type = "table" AND tbl_name = "user_auth" AND (sql LIKE "%CONSTRAINT \"fk_user_auth_auth\" %" OR sql LIKE "%CONSTRAINT fk_user_auth_auth %" OR sql LIKE "%CONSTRAINT `fk_user_auth_auth`%" OR sql LIKE "%CONSTRAINT [fk_user_auth_auth]%" OR sql LIKE "%CONSTRAINT 	fk_user_auth_auth	%")

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:49
[0m[33m[0.013ms] [34;1m[rows:-][0m SELECT count(*) FROM sqlite_master WHERE type = "table" AND tbl_name = "user_auth" AND (sql LIKE "%CONSTRAINT \"fk_user_auth_user\" %" OR sql LIKE "%CONSTRAINT fk_user_auth_user %" OR sql LIKE "%CONSTRAINT `fk_user_auth_user`%" OR sql LIKE "%CONSTRAINT [fk_user_auth_user]%" OR sql LIKE "%CONSTRAINT 	fk_user_auth_user	%")

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:50
[0m[33m[0.008ms] [34;1m[rows:-][0m SELECT count(*) FROM sqlite_master WHERE type='table' AND name="users"

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:50
[0m[33m[0.030ms] [34;1m[rows:2][0m SELECT sql FROM sqlite_master WHERE type IN ("table","index") AND tbl_name = "users" AND sql IS NOT NULL order by type = "table" desc

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:50
[0m[33m[0.009ms] [34;1m[rows:-][0m SELECT * FROM `users` LIMIT 1

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:50
[0m[33m[0.009ms] [34;1m[rows:-][0m SELECT count(*) FROM sqlite_master WHERE type = "index" AND tbl_name = "users" AND name = "idx_users_deleted_at"

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:50
[0m[33m[0.008ms] [34;1m[rows:-][0m SELECT count(*) FROM sqlite_master WHERE type='table' AND name="auths"

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:50
[0m[33m[0.029ms] [34;1m[rows:2][0m SELECT sql FROM sqlite_master WHERE type IN ("table","index") AND tbl_name = "auths" AND sql IS NOT NULL order by type = "table" desc

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:50
[0m[33m[0.008ms] [34;1m[rows:-][0m SELECT * FROM `auths` LIMIT 1

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:50
[0m[33m[0.009ms] [34;1m[rows:-][0m SELECT count(*) FROM sqlite_master WHERE type = "index" AND tbl_name = "auths" AND name = "idx_auths_deleted_at"

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:50
[0m[33m[0.007ms] [34;1m[rows:-][0m SELECT count(*) FROM sqlite_master WHERE type='table' AND name="user_auths"

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:50
[0m[33m[0.028ms] [34;1m[rows:1][0m SELECT sql FROM sqlite_master WHERE type IN ("table","index") AND tbl_name = "user_auths" AND sql IS NOT NULL order by type = "table" desc

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:50
[0m[33m[0.008ms] [34;1m[rows:-][0m SELECT * FROM `user_auths` LIMIT 1

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:50
[0m[33m[0.013ms] [34;1m[rows:-][0m SELECT count(*) FROM sqlite_master WHERE type = "table" AND tbl_name = "user_auths" AND (sql LIKE "%CONSTRAINT \"fk_user_auths_user\" %" OR sql LIKE "%CONSTRAINT fk_user_auths_user %" OR sql LIKE "%CONSTRAINT `fk_user_auths_user`%" OR sql LIKE "%CONSTRAINT [fk_user_auths_user]%" OR sql LIKE "%CONSTRAINT 	fk_user_auths_user	%")

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:50
[0m[33m[0.012ms] [34;1m[rows:-][0m SELECT count(*) FROM sqlite_master WHERE type = "table" AND tbl_name = "user_auths" AND (sql LIKE "%CONSTRAINT \"fk_user_auths_auth\" %" OR sql LIKE "%CONSTRAINT fk_user_auths_auth %" OR sql LIKE "%CONSTRAINT `fk_user_auths_auth`%" OR sql LIKE "%CONSTRAINT [fk_user_auths_auth]%" OR sql LIKE "%CONSTRAINT 	fk_user_auths_auth	%")

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:55
[0m[33m[0.247ms] [34;1m[rows:0][0m SELECT * FROM `user_auth` WHERE `user_auth`.`auth_id` IN (1,2)

2023/02/18 22:42:43 [32m/Users/hanjian/work/go/gostudy/examples/gorm/main.go:55
[0m[33m[0.745ms] [34;1m[rows:2][0m SELECT * FROM `auths` WHERE `auths`.`deleted_at` IS NULL
[{{1 2023-02-18 22:28:13.108385 +0800 +0800 2023-02-18 22:28:13.108385 +0800 +0800 {0001-01-01 00:00:00 +0000 UTC false}} admin []} {{2 2023-02-18 22:28:13.108385 +0800 +0800 2023-02-18 22:28:13.108385 +0800 +0800 {0001-01-01 00:00:00 +0000 UTC false}} edit []}]
