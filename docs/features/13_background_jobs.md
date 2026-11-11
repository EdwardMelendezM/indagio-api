# 13 · Worker de jobs en segundo plano

## Propósito

Sacar del camino de la request HTTP todo lo que puede esperar o fallar: envío de
correos hoy, y —cuando se implementen— procesamiento de exports y aplicación de
lotes de sync.

Es una **cola sobre Postgres**, no Redis. `docs/1_MAIN_FEATURES.md` §9 proponía
`asynq` sobre Redis; lo implementado evita esa dependencia operativa a cambio de
polling. Para el volumen de un proyecto de investigación es la decisión correcta
y vale la pena dejarla asentada.

## Entidades y tablas

### `background_jobs`

| Columna | Tipo | Nota |
|---|---|---|
| `type` | `VARCHAR(80)` | discriminador; ver tipos abajo |
| `payload` | `JSONB` | tipado por `type` |
| `status` | `VARCHAR(32)` | `pending` \| `processing` \| `completed` \| `failed` |
| `attempts` / `max_attempts` | `INTEGER` | default 0 / 5 |
| `run_at` | `TIMESTAMPTZ` | default `NOW()`; el backoff lo empuja al futuro |
| `error_log` | `TEXT` NULL | último error |

Índice `idx_background_jobs_status_run_at (status, run_at)`: exactamente las dos
columnas del `WHERE` del dequeue.

## Cómo corre

`main.go:192-195` — una goroutine arrancada al inicio, con un `context`
cancelable atado al apagado del proceso:

```go
jobWorker := worker.NewJobWorker(jobRepo, emailSvc, logger)
jobCtx, stopJobs := context.WithCancel(context.Background())
defer stopJobs()
go jobWorker.Run(jobCtx)
```

El bucle (`job_worker.go:41`) hace `Dequeue` y, si no hay nada, duerme 3 s
(`pollInterval`). Un solo job por iteración, procesado de forma secuencial: sin
concurrencia dentro del worker.

### El dequeue es seguro entre réplicas

`queries/dequeue.sql`:

```sql
UPDATE background_jobs
SET status = 'processing', attempts = attempts + 1, updated_at = NOW()
WHERE id = (
    SELECT id FROM background_jobs
    WHERE status = 'pending' AND run_at <= NOW() AND type = ANY($1)
    ORDER BY run_at ASC
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING ...
```

`FOR UPDATE SKIP LOCKED` es la pieza clave: varias instancias de la API pueden
correr su worker contra la misma base sin tomar el mismo job dos veces. El
diseño ya soporta escalar horizontalmente.

### Reintentos con backoff exponencial

`queries/mark_failed.sql` — al fallar con `scheduleRetry = true`:

```
run_at = NOW() + INTERVAL '5 minutes' * 2^(attempts - 1)
```

Es decir 5 min, 10, 20, 40, 80… hasta `max_attempts` (5 por defecto), donde el
job pasa a `failed` definitivo.

Un `payload` malformado se marca con `scheduleRetry = false`: no se reintenta,
porque volver a intentarlo daría el mismo error. Distinción correcta entre fallo
transitorio y permanente.

Un `type` desconocido también cae en `failed` inmediato, con un `Warn` en el log
(`job_worker.go:91`).

## Tipos de job

Declarados en `domain/job.go:13-19`:

| Tipo | ¿Lo maneja el worker? | ¿Corresponde a Indagio? |
|---|---|---|
| `send_otp` | ✅ | ✅ |
| `send_password_reset` | ✅ | ✅ |
| `send_push_reaction` | ❌ | ❌ herencia |
| `send_push_comment` | ❌ | ❌ herencia |
| `send_push_new_thread` | ❌ | ❌ herencia |
| `send_push_chat_message` | ❌ | ❌ herencia |
| `send_push_scholarship_deadline` | ❌ | ❌ herencia |

Los cinco de notificaciones push están **registrados en `jobTypes`**
(`job_worker.go:29-37`), así que el `Dequeue` los reclama, pero el `switch` de
`process` no tiene `case` para ellos: caen en el `default` y se marcan `failed`
con `"unknown type"`. Hoy es inocuo porque nadie los encola. Ver
[G-11](00_GAPS.md#g-11--los-tipos-de-job-son-de-otro-producto).

### Los que faltan

| Tipo propuesto | Para | Gap |
|---|---|---|
| `process_export` | generar el archivo y subirlo a R2 | [G-05](00_GAPS.md#g-05--los-exports-nunca-se-procesan) |
| `process_sync_batch` | aplicar un lote offline | [G-04](00_GAPS.md#g-04--los-lotes-de-sync-nunca-se-aplican) |
| `send_project_invitation` | avisar al invitado por correo | [G-06](00_GAPS.md#g-06--el-flujo-de-invitaciones-no-se-puede-cerrar) |

## Interfaz

```go
type JobRepository interface {
    Enqueue(ctx, jobType string, payload any) error
    Dequeue(ctx, jobTypes []string) (*Job, error)
    MarkCompleted(ctx, id uuid.UUID) error
    MarkFailed(ctx, id uuid.UUID, errMsg string, scheduleRetry bool) error
    PurgeOld(ctx, olderThanDays int) (int64, error)
}
```

`PurgeOld` borra los `completed` y `failed` más viejos que N días. Está
implementado y **nadie lo llama**: `background_jobs` crece sin límite. Haría
falta un tick periódico en el worker.

## Autorización

No aplica: proceso interno, sin superficie HTTP. No hay endpoint para consultar
ni reintentar jobs.

## Dependencias

- **Requiere:** Postgres; `utils.EmailService` (SMTP configurado).
- **Lo consume:** [01_auth](01_auth.md) (OTP y reset de contraseña),
  [03_admin](03_admin.md) (OTP de admin). Debería consumirlo
  [12_exports](12_exports.md) y [11_offline_sync](11_offline_sync.md).

## Gaps conocidos

- **[G-11](00_GAPS.md#g-11--los-tipos-de-job-son-de-otro-producto)** — cinco
  tipos de push de otro producto registrados en `jobTypes` sin handler; faltan
  los tres que el flujo necesita.
- `PurgeOld` implementado y nunca invocado: la tabla crece indefinidamente.
- Sin observabilidad: no hay métricas de profundidad de cola, latencia ni tasa de
  fallo. `docs/1_MAIN_FEATURES.md` §13 lo pide explícitamente, y es donde más
  importa, porque un job que falla en silencio no tiene quien lo note.
- Un solo job por iteración, sin concurrencia interna. Suficiente para correos;
  habrá que revisarlo cuando un export grande ocupe el worker varios minutos y
  bloquee la cola de OTPs detrás.
- `pollInterval` de 3 s está fijo en el código (`job_worker.go:38`), no en
  configuración.
- El worker no espera a terminar el job en curso durante el apagado: el
  `context` se cancela y el job queda en `processing`, sin nadie que lo
  rescate — no hay recuperación de jobs trabados en ese estado.
