# 11 · Sincronización offline por lotes

## Propósito

Permitir que un cliente móvil trabaje sin conexión en campo y suba después todo
lo recolectado en un solo envío. En investigación de campo —zonas sin cobertura,
entrevistas largas— es la pieza que decide si los datos se pierden o no.

⚠️ **El módulo está a medias y es el más riesgoso del repositorio.** Recibe
lotes, los guarda y responde `201`, pero **nada los aplica nunca**; y el endpoint
**no tiene autenticación**. Ver
[G-03](00_GAPS.md#g-03--apisyncbatches-está-sin-autenticación) y
[G-04](00_GAPS.md#g-04--los-lotes-de-sync-nunca-se-aplican).

## Entidades y tablas

### `sync_batches`

| Columna | Tipo | Nota |
|---|---|---|
| `project_id`, `participant_id` | `UUID` | `ON DELETE CASCADE` |
| `client_generated_batch_id` | `VARCHAR(128)` | `UNIQUE(project_id, participant_id, client_generated_batch_id)` |
| `payload` | `JSONB` | **esquema no definido** |
| `status` | `VARCHAR(32)` | `pending` \| `processed` \| `failed` — siempre `pending` |

El `UNIQUE` sí da idempotencia real: reenviar el mismo lote tras un corte choca
con el constraint en lugar de duplicar. Es la única garantía que el módulo cumple
hoy.

### `sync_events`

`batch_id`, `event_type`, `entity_id`, `payload`. Pensada para registrar qué se
aplicó de cada lote. **Siempre vacía**: nada inserta filas.

## Endpoints

| Método | Ruta | Auth | Descripción |
|---|---|---|---|
| `POST` | `/api/sync/batches` | **ninguna** | Recibe un lote |
| `GET` | `/api/sync/batches?participant_id=` | **ninguna** | Lista lotes del participante |

`sync_handler.go:112-115`. Registradas en `main.go:181` como
`RegisterSyncRoutes(api, syncHandler)` — **sin `authMiddleware`**, a diferencia
de todos los demás módulos.

El handler intenta leer el actor de un header y, si no está, lo inventa:

```go
actorID, err := uuid.Parse(c.GetHeader("X-Actor-ID"))
if err != nil || actorID == uuid.Nil {
    actorID = uuid.New() // temporary compatibility for unscoped local validation
}
```

`sync_handler.go:68` y `:91`. El comentario reconoce que es provisional. El
efecto es un endpoint público de escritura: cualquiera puede insertar lotes
contra cualquier `project_id` / `participant_id` sin credenciales.

## Contratos

```jsonc
// POST /api/sync/batches
{
  "project_id": "<uuid>",
  "participant_id": "<uuid>",
  "client_generated_batch_id": "batch-001",
  "payload": { /* forma libre: no hay esquema definido */ }
}
→ 201 { "id", "project_id", "participant_id", "client_generated_batch_id",
        "status": "pending", "created_at", "updated_at" }
```

El `payload` solo se valida con `json.Valid` (`sync usecase:30`). No hay
contrato: ni qué eventos admite, ni cómo se representa una respuesta o una
aplicación offline.

Falta definir, como mínimo:

- lista de operaciones por lote, cada una con su tipo
  (`administration.start`, `answer.create`, `administration.complete`);
- el `client_generated_id` de cada entidad, para idempotencia por elemento y no
  solo por lote;
- la respuesta de aplicación parcial: qué se aplicó y qué falló, que es lo que
  `docs/1_MAIN_FEATURES.md` §7 pide explícitamente ("confirmación explícita de
  qué se sincronizó y qué falló"). Hoy el `201` no informa nada de eso.

## Reglas de negocio

`SubmitBatch` (`sync usecase:22`) valida ids no nulos,
`client_generated_batch_id` no vacío tras recortar, y `payload` como JSON válido.
Luego inserta y responde.

`ListBatches` lista por participante. `GetBatch` está implementado en el usecase
pero **no ruteado**.

`MarkBatchProcessed`, `CreateEvent` y `ListEventsByBatch` están implementados en
el repositorio y **no los llama nadie**.

## Autorización

| | |
|---|---|
| **Implementada** | Ninguna. El actor se fabrica. |
| **Esperada** | Autenticación obligatoria, y pertenencia al proyecto del lote. Si el día que exista un acceso de participante los lotes los envía el propio sujeto, hará falta un token acotado a su participación. |

## Dependencias

- **Requiere:** [04_projects](04_projects.md), [05_participants](05_participants.md).
- **Debería alimentar:** [08_administrations](08_administrations.md) y
  [09_answers](09_answers.md). Hoy no alimenta nada.

## Gaps conocidos

- **[G-03](00_GAPS.md#g-03--apisyncbatches-está-sin-autenticación)** — endpoint
  público de escritura con actor fabricado. **P0.**
- **[G-04](00_GAPS.md#g-04--los-lotes-de-sync-nunca-se-aplican)** — nada aplica
  los lotes: el soporte offline es de solo escritura. Un dispositivo sube su
  trabajo de campo y los datos no aparecen en el proyecto.
- Esquema del `payload` sin definir.
- `sync_events` siempre vacía; `MarkBatchProcessed` y `CreateEvent` sin usar.
- `GetBatch` implementado y sin ruta.
- Sin respuesta de aplicación parcial.
- `instrument_administrations.sync_status` y `answer_records.sync_status` existen
  para que este módulo las mueva; nunca salen de `pending`.

## Orden de trabajo sugerido

1. Poner `authMiddleware` en las rutas y tomar el actor del token (cierra G-03).
2. Definir el esquema del `payload` y documentarlo aquí.
3. Agregar un `JobTypeProcessSyncBatch` al worker que aplique cada operación en
   transacción, escriba `sync_events` y mueva `status` a `processed`/`failed`
   (cierra G-04).
4. Devolver el detalle de aplicación parcial en `GET /sync/batches/:id`.

## Requests de ejemplo

[`requests/sync.http`](../../requests/sync.http)
