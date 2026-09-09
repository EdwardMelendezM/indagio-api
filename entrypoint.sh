#!/bin/sh
set -e

echo "===> [ENTRYPOINT] Corriendo migraciones pendientes..."
migrate -path ./db/migrations -database "$DATABASE_URL" up

echo "===> [ENTRYPOINT] Migraciones completadas. Iniciando la API..."
exec ./main