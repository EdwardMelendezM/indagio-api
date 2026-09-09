FROM golang:1.26-alpine AS builder

# gcc y musl-dev son necesarios para que CGO_ENABLED=1 pueda usar el linker
# externo (musl-gcc) y producir un binario 100% estático con
# -extldflags=-static. Sin gcc, CGO_ENABLED=1 falla al compilar.
RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev

WORKDIR /app

# Instalamos la herramienta de migraciones. Igual que con el binario
# principal, hay que forzar linkado estático (-extldflags=-static) para
# que funcione en el runner con glibc; si no, linkea contra musl y el
# binario falla con "migrate: not found".
RUN CGO_ENABLED=1 go install -ldflags '-s -w -extldflags "-static"' -tags 'postgres,netgo,osusergo' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.0

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compilamos desde la raíz ya que ahí está tu main.go
#RUN cat main.go | grep media
#RUN ls -la media/
# CGO_ENABLED=1 + -extldflags=-static produce un binario 100% estático
# (sin PT_INTERP, sin NEEDED contra ld-musl/libc). Esto evita que el
# binario linkee contra la musl de Alpine, que es incompatible con la
# glibc del runner (debian:bookworm-slim). Si solo ponemos CGO_ENABLED=0
# el binario sigue incluyendo /lib/ld-musl-*.so.1 como interpreter y
# falla con "exec: ./main: not found" en Debian.
RUN CGO_ENABLED=1 go build -ldflags '-s -w -extldflags "-static"' -tags 'netgo,osusergo' -o main .

# ─── Stage 2 (Imagen ligera de ejecución) ───
FROM debian:bookworm-slim AS runner
WORKDIR /app

# ffmpeg es requerido por el video worker (HLS transcoding + thumbnails + ffprobe).
# ca-certificates y tzdata son utilidades estándar.
# curl lo usa el HEALTHCHECK contra /healthz.
#RUN apt-get update && apt-get install -y --no-install-recommends \
#    ffmpeg ca-certificates tzdata curl \
# && rm -rf /var/lib/apt/lists/*

# Binarios
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate
COPY --from=builder /app/main .

# Copiamos las migraciones respetando tu estructura db/migrations
COPY --from=builder /app/db/migrations ./db/migrations

COPY entrypoint.sh .
RUN chmod +x entrypoint.sh

EXPOSE 8080

# Liveness probe para Dokku / Docker: si /healthz deja de responder 200,
# el contenedor se marca como unhealthy y Dokku lo reinicia en lugar de
# quedarse en crash-loop silencioso.
HEALTHCHECK --interval=30s --timeout=3s --start-period=15s --retries=3 \
  CMD curl -fsS http://localhost:8080/healthz || exit 1

# El entrypoint se encarga de todo el flujo de inicio de forma segura
ENTRYPOINT ["./entrypoint.sh"]
