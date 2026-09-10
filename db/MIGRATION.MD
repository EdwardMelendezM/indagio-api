# MIGRATIONS

## NEW MIGRATIONS
```bash
migrate create -ext sql -dir db/migrations {name}.sql
```

## Apply migrations and check

```bash
# Install migrate CLI
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run up
migrate -path db/migrations -database "postgres://postgres:postgres@localhost:5492/indagio_db?sslmode=disable" up

.\migrate -path db/migrations -database "postgresql://neondb_owner:npg_b0w9TXCJRmIO@ep-royal-scene-ad26zls9.c-2.us-east-1.aws.neon.tech/neondb?sslmode=require" up

# Verify
# Para verificar las tablas
docker exec -it foro_unsaac_postgres_local psql -U postgres -d foro_unsaac_db -c "\dt"
```

## ⚠️ Dirty database version — recovery

`golang-migrate` keeps state in the `schema_migrations(version BIGINT, dirty BOOLEAN)`
table. If a migration partially runs (server crash, manual `psql -f` bypass,
network blip) the row is left with `dirty = true` and the next `migrate up`
refuses to run with:

```
error: Dirty database version 20260721000001. Fix and force version.
```

**Recovery:**

1. Identify the latest applied version:

   ```sql
   SELECT version, dirty FROM schema_migrations;
   ```

2. If the schema actually matches that version, mark it clean and (optionally)
   bump to the highest applied version:

   ```sql
   -- example: every migration up to 20260721000003 was already applied via psql
   UPDATE schema_migrations
   SET    version = 20260721000003,
          dirty   = false
   WHERE  version = 20260721000001;
   ```

3. Re-run `migrate up` — should report `no change`.

> **Prevention:** never apply a migration file via `psql -f` directly unless
> you also `INSERT INTO schema_migrations (version, dirty) VALUES (X, false)`
> in the same statement. The Docker one-liners in the verify section above are
> fine for inspection; only use them to apply migrations if you also seed the
> tracking table.


## Windows
- New migration
```powershell
&$HOME\go\bin\migrate create -ext sql -dir db/migrations {name}.sql
```
- Apply migration
```powershell
&$HOME\go\bin\migrate -path db/migrations -database "DATABASE_URL" up
```

## Restore backup windows
```bash
& C:\Users\Edward\scoop\apps\postgresql\current\bin\pg_restore.exe --host=HOST --port=PORT --username=USERNAME --dbname=DBNAME --format=c --no-owner --no-privileges --verbose "C:\Users\Edward\Downloads\postgres-foro-unsaac-postgres-db-2026-06-13-06-00-06\backup\export"
```

## Seed
```bash
docker cp seeds/categories.sql foro_unsaac_postgres_local:/categories.sql
docker exec -it foro_unsaac_postgres_local psql -U postgres -d foro_unsaac_db -f /categories.sql
```

## APPLY SEEDs WITH SSH
From your local machine, you can apply the migration to the remote database using SSH. Make sure you have the necessary permissions and that the remote server is accessible.
```bash
ssh user@IP "dokku postgres:connect foro-unsaac-postgres-db" < db/seeds/categories.sql
```

## APPLY MIGRATION WITH SSH
```bash
ssh user@IP "dokku run foro-unsaac-api migrate -path ./db/migrations -database \$(dokku config:get foro-unsaac-api DATABASE_URL) up"
```