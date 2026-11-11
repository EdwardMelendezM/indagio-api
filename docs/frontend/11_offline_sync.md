# 11 · Sincronización offline

## Pantallas

Cliente móvil o PWA que recoge datos en campo sin conexión y los sube al
recuperarla · indicador de estado de sincronización.

## ⚠️ Antes de integrar: el módulo no funciona

Dos problemas que cambian por completo cómo encarar esto:

1. **Los lotes nunca se aplican.** El endpoint los recibe, los guarda y responde
   `201`. **Nada los convierte en respuestas ni en aplicaciones.** Los datos
   entran a una tabla y se quedan ahí: no aparecen en el proyecto, no se
   puntúan, no salen en los exports. El soporte offline es, hoy, de solo
   escritura (G-04).
2. **El endpoint no tiene autenticación** y fabrica un actor cuando no recibe el
   header `X-Actor-ID` (G-03). Es una ruta pública de escritura.

Y una consecuencia técnica que lo vuelve inutilizable desde el navegador:

3. **CORS no permite el header `X-Actor-ID`.** `AllowHeaders` solo incluye
   `Authorization` y `Content-Type`, así que **una app web no puede identificar
   al actor de ningún modo**. El backend le asignará un UUID aleatorio a cada
   lote.

**Recomendación: no construir el flujo offline sobre este endpoint todavía.** Ver
la alternativa al final de este documento.

## Tipos

```ts
interface SubmitBatchRequest {
  project_id: string;                  // requerido, UUID
  participant_id: string;              // requerido, UUID
  client_generated_batch_id: string;   // requerido — da idempotencia por lote
  payload: unknown;                    // ⚠️ SIN ESQUEMA DEFINIDO
}

interface SyncBatchResponse {
  id: string;
  project_id: string;
  participant_id: string;
  client_generated_batch_id: string;
  status: SyncBatchStatus;             // siempre "pending": nada lo mueve
  created_at: string;
  updated_at: string;
  // El payload NO se devuelve.
}
```

### `payload` no tiene contrato

El backend solo verifica que sea JSON válido. No hay esquema: ni qué operaciones
admite, ni cómo se representa una respuesta o una aplicación offline.

**No se puede integrar contra un contrato que no existe.** Lo que falta acordar
con backend, como mínimo:

```ts
// PROPUESTA — no implementada, sujeta a acuerdo con backend
interface SyncPayload {
  operations: SyncOperation[];
}

type SyncOperation =
  | { type: "administration.start";
      client_generated_id: string; instrument_id: string; started_at: string }
  | { type: "answer.create";
      client_generated_id: string; administration_client_id: string;
      question_key: string; answer_type: AnswerType; value: { value: unknown } }
  | { type: "administration.complete";
      administration_client_id: string; completed_at: string };
```

Y la respuesta tendría que informar qué se aplicó y qué falló —lo que
`docs/1_MAIN_FEATURES.md` §7 pide explícitamente y hoy no existe:

```ts
// PROPUESTA
interface SyncResultResponse {
  batch_id: string;
  status: "processed" | "partial" | "failed";
  applied: { client_generated_id: string; server_id: string }[];
  failed:  { client_generated_id: string; error: string }[];
}
```

Hoy el `201` no dice nada de eso: solo confirma que el lote se guardó.

## Endpoints

| Método | Ruta | Auth | Respuesta |
|---|---|---|---|
| `POST` | `/sync/batches` | **ninguna** | `201 SyncBatchResponse` |
| `GET` | `/sync/batches?participant_id=` | **ninguna** | `200 SyncBatchResponse[]` |

```ts
export const syncApi = {
  submitBatch: (body: SubmitBatchRequest) =>
    api.post<SyncBatchResponse>("/sync/batches", body),

  listBatches: (participantId: string) =>
    api.get<SyncBatchResponse[]>(`/sync/batches?participant_id=${participantId}`),
};
```

`participant_id` en el `GET` es **obligatorio** y va como query string: sin él
devuelve `400`. No hay forma de listar los lotes de un proyecto.

No existe `GET /sync/batches/:id`: `GetBatch` está implementado en el backend
pero sin ruta. No se puede consultar un lote puntual ni su resultado.

## Idempotencia: lo único que sí funciona

`client_generated_batch_id` tiene un `UNIQUE (project_id, participant_id,
client_generated_batch_id)` real. Reenviar el mismo lote tras un corte **no
duplica**: choca con el constraint.

```ts
// Generar el id una vez, al crear el lote local — no al enviarlo
const batch = {
  id: crypto.randomUUID(),      // se mantiene entre reintentos
  projectId, participantId,
  operations: [],
};
```

Es la única garantía que el módulo cumple hoy, y vale aprovecharla: permite
reintentar sin duplicar.

⚠️ **Pero el choque sale como `500`, no como `409`.** La violación de unicidad no
se traduce a un error de dominio: el repositorio envuelve el error de constraint
tal cual. Un reenvío legítimo es indistinguible de una caída del servidor.

```ts
try {
  await syncApi.submitBatch(body);
  markAsSent(body.client_generated_batch_id);
} catch (e) {
  if (e instanceof ApiError && e.status === 500) {
    // Puede ser el UNIQUE (ya subido) o un error real. Hay que comprobarlo.
    const batches = await syncApi.listBatches(body.participant_id);
    const already = batches.some(
      b => b.client_generated_batch_id === body.client_generated_batch_id,
    );
    if (already) { markAsSent(body.client_generated_batch_id); return; }
  }
  throw e;
}
```

El `GET` sirve para desambiguar, que es lo único bueno de que exista.

## El problema del actor

```go
actorID, err := uuid.Parse(c.GetHeader("X-Actor-ID"))
if err != nil || actorID == uuid.Nil {
    actorID = uuid.New()   // "temporary compatibility for unscoped local validation"
}
```

El backend espera un header `X-Actor-ID`, y si no lo recibe **inventa un UUID
aleatorio**. Como CORS no permite ese header, desde el navegador es inalcanzable:
cada lote queda atribuido a un actor distinto e inexistente.

Un cliente nativo (React Native, Flutter) **sí** puede mandarlo, porque no está
sujeto a CORS. Pero no hay forma de que el servidor verifique que ese valor sea
cierto: no es autenticación, es una declaración del cliente.

```ts
// Solo en cliente nativo; desde el navegador este header se descarta.
headers: { "X-Actor-ID": currentUserId }
```

## Alternativa recomendada mientras tanto

Construir la cola offline **en el cliente**, sobre los endpoints normales que sí
funcionan. Más trabajo, pero los datos llegan de verdad al proyecto.

```ts
type QueuedCall =
  | { kind: "start";    projectId: string; participantId: string; instrumentId: string; localId: string }
  | { kind: "answer";   projectId: string; administrationId: string; body: CreateAnswerRequest }
  | { kind: "complete"; administrationId: string };

// Persistir en IndexedDB: localStorage no alcanza para trabajo de campo.
async function flushQueue(queue: QueuedCall[]) {
  for (const call of queue) {
    try {
      switch (call.kind) {
        case "start":
          // startOrResume maneja el UNIQUE, que sale como 500 —
          // ver 08_administrations.md
          await startOrResume(
            call.projectId, call.participantId, call.instrumentId, call.localId,
          );
          break;
        case "answer":
          await answersApi.create(call.projectId, call.administrationId, call.body);
          break;
        case "complete":
          await administrationsApi.complete(call.administrationId);
          break;
      }
      dequeue(call);
    } catch {
      break;    // cortar el drenaje: el orden importa y lo que sigue depende de esto
    }
  }
}
```

Dos cuidados en ese enfoque:

- **El orden es obligatorio**: `start` → `answer`s → `complete`. Si falla un
  `start`, no se puede seguir con sus respuestas, porque hace falta el
  `administration_id` que devuelve el servidor. De ahí el `break` en vez de
  `continue`.
- **Los reintentos de `start` necesitan `startOrResume`**, porque el choque de
  `client_generated_id` llega como `500` y hay que distinguirlo de un fallo real
  consultando el listado.
- **Las respuestas no tienen idempotencia real** (`client_generated_id` sin
  `UNIQUE`, ver [09_answers.md](09_answers.md)): hay que marcar localmente las ya
  enviadas para no duplicar en un reintento.

## Validación en el cliente

| Campo | Regla |
|---|---|
| `project_id` | requerido, UUID |
| `participant_id` | requerido, UUID |
| `client_generated_batch_id` | requerido, no vacío tras recortar, máx. 128 |
| `payload` | requerido, JSON válido |

## Errores

| Código | Significado | Acción |
|---|---|---|
| `400` | `project_id` o `participant_id` no son UUID válidos | Error de programación |
| `500` | ⚠️ Lote ya subido (violación de `UNIQUE`) **o** error real — indistinguibles | Comprobar con el `GET` antes de decidir |
| `422` | Falta un campo o `payload` no es JSON | Error de programación |

## Trampas

1. **Los lotes nunca se aplican** (G-04): se guardan y nada más. Los datos no
   llegan al proyecto.
2. **El endpoint es público** y fabrica el actor (G-03).
3. **CORS no permite `X-Actor-ID`**: desde el navegador no hay forma de
   identificarse.
4. **`payload` no tiene esquema**: no hay contra qué integrar.
5. **`status` siempre es `pending`**: no sirve como indicador de progreso en la
   UI.
6. **No hay respuesta de aplicación parcial**: el `201` no dice qué se aceptó.
7. **No hay `GET /sync/batches/:id`**: no se puede consultar un lote.
8. **El `GET` exige `participant_id`**: no hay vista por proyecto.
9. **Un reenvío da `500`, no `409`**: hay que consultar el `GET` para saber si
   el lote ya estaba subido o si el servidor falló de verdad.
10. **`sync_status` en respuestas y aplicaciones nunca cambia de `pending`**:
    esos campos existen para que este módulo los mueva, y no lo hace.
