# 09 · Respuestas y registro de media

## Pantallas

Formulario de ítems del instrumento (el corazón del runner) · revisión de
respuestas de un participante · asociación de archivos de audio/video.

⚠️ **Leer primero la sección del envoltorio de `value`.** Es el error más caro
del sistema: se acepta con `201` y rompe el puntaje tres pasos después, sin
ningún mensaje.

## El envoltorio de `value` — la trampa principal

El motor de puntuación lee `answer_records.value` esperando **exactamente** esta
forma:

```jsonc
{ "value": 2 }                    // ✅ numérico: se usa directo
{ "value": "Varios días" }        // ✅ etiqueta: se traduce con el value_map del ítem
{ "value": "texto libre" }        // ✅ para ítems con is_scored: false
```

Cualquier otra clave de primer nivel se guarda sin error y el motor la **ignora
en silencio**:

```jsonc
{ "raw": 2 }                      // ❌ se acepta con 201 y el ítem cuenta como NO respondido
{ "answer": 2 }                   // ❌ igual
2                                 // ❌ igual (no es objeto)
```

El efecto: con el PHQ-8 y su `min_items_required: 6`, las 8 respuestas cuentan
como 0 respondidas, no se alcanza el mínimo, y `raw_score` sale `null`. Sin
excepción, sin log, sin pista.

⚠️ **`requests/answers.http` en el repositorio usa `{"raw": 2}` y está mal.** Es
el gap G-01; no tomar esos ejemplos como referencia para el payload.

Blindarlo en el cliente con un constructor único:

```ts
/** Única forma permitida de construir el value de una respuesta. */
export function answerValue(v: number | string | boolean): { value: number | string | boolean } {
  return { value: v };
}

// Uso — nunca armar el objeto a mano
await answersApi.create(projectId, administrationId, {
  participant_id: participantId,
  instrument_id: instrumentId,
  question_key: "phq8_1",
  answer_type: "scale",
  value: answerValue(2),
  client_generated_id: `${administrationId}:phq8_1`,
});
```

### Numérico o etiqueta: cuál mandar

| Caso | Qué enviar |
|---|---|
| El ítem tiene `value_map` y la UI conoce el número | `answerValue(2)` — directo, sin ambigüedad |
| La UI solo tiene la etiqueta visible | `answerValue("Varios días")` — **debe existir como clave en el `value_map`** |
| Ítem de texto, `is_scored: false` | `answerValue("respuesta abierta")` |

⚠️ Si se manda una etiqueta que **no** está en el `value_map`, el cálculo falla
con `500` al completar la aplicación — no al crear la respuesta. Mandar el número
es más seguro:

```ts
// Traducir la etiqueta a número en el cliente, usando el value_map del catálogo
function numericFor(item: ItemResponse, selectedLabel: string): number {
  const n = item.value_map?.[selectedLabel];
  if (n === undefined) throw new Error(`"${selectedLabel}" no está en el value_map de ${item.question_key}`);
  return n;
}
```

## Tipos

```ts
interface CreateAnswerRequest {
  participant_id: string;          // requerido, aunque la ruta ya tenga administrationId
  instrument_id?: string | null;   // opcional pero recomendado
  question_key: string;            // requerido — debe existir en el catálogo congelado
  answer_type: AnswerType;         // requerido
  value: { value: number | string | boolean };   // requerido — ver arriba
  client_generated_id?: string | null;
}

interface CreateMediaRequest {
  answer_id?: string | null;
  file_key: string;                // requerido — la clave en R2
  mime_type: string;               // requerido
  size_bytes: number;              // requerido y > 0 (ver trampas)
  duration_seconds?: number | null;
  checksum?: string | null;
}

interface InstrumentData { id: string; name?: string; kind?: InstrumentKind }

interface AnswerResponse {
  id: string;
  project_id: string;
  participant_id: string;
  instrument_id?: string;          // ausente si es null
  instrument?: InstrumentData;     // hidratado solo al listar
  question_key: string;
  answer_type: AnswerType;
  value: unknown;                  // tal cual se guardó
  status: string;                  // siempre "draft"
  sync_status: string;             // siempre "pending"
  client_generated_id?: string;    // ausente si es null
  created_at: string;
  updated_at: string;
  // Sin administration_id: NO se devuelve, aunque se guarde.
}

interface MediaResponse {
  id: string; project_id: string; participant_id: string;
  answer_id?: string; file_key: string; mime_type: string;
  size_bytes: number; duration_seconds?: number; checksum?: string;
  status: string;                  // siempre "pending"
  created_at: string; updated_at: string;
}
```

⚠️ **`AnswerResponse` no devuelve el `administration_id`**, aunque la respuesta
quede ligada a la aplicación. El cliente tiene que recordar a qué intento
pertenece cada respuesta por su cuenta.

⚠️ `status` y `sync_status` **nunca cambian** de `draft` y `pending`. No
construir UI sobre ellos.

## Endpoints

| Método | Ruta | Respuesta |
|---|---|---|
| `POST` | `/projects/:id/administrations/:administrationId/answers` | `201 AnswerResponse` |
| `GET` | `/projects/:id/participants/:participantID/answers[?instrumentId=]` | `200 AnswerResponse[]` |
| `POST` | `/projects/:id/participants/:participantID/media` | `201 MediaResponse` |

```ts
export const answersApi = {
  create: (projectId: string, administrationId: string, body: CreateAnswerRequest) =>
    api.post<AnswerResponse>(
      `/projects/${projectId}/administrations/${administrationId}/answers`, body,
    ),

  listByParticipant: (projectId: string, participantId: string, instrumentId?: string) =>
    api.get<AnswerResponse[]>(
      `/projects/${projectId}/participants/${participantId}/answers${
        instrumentId ? `?instrumentId=${instrumentId}` : ""
      }`,
    ),

  createMedia: (projectId: string, participantId: string, body: CreateMediaRequest) =>
    api.post<MediaResponse>(
      `/projects/${projectId}/participants/${participantId}/media`, body,
    ),
};
```

⚠️ La ruta de creación lleva **el `project_id` y el `administrationId`**, y el
body repite el `participant_id`. Es redundante y el backend **no verifica que
coincidan**: un `participant_id` equivocado en el body guarda la respuesta
contra otro sujeto sin protestar. Validarlo en el cliente.

## Flujo del formulario

Una llamada por ítem. No hay envío en lote (para eso estaría el sync, que no
aplica los lotes — ver [11_offline_sync.md](11_offline_sync.md)).

```ts
async function submitAnswers(
  projectId: string, administrationId: string,
  participantId: string, instrumentId: string,
  responses: Record<string, number | string>,   // question_key → valor
  catalog: Catalog,
) {
  const byKey = new Map(catalog.items.map(i => [i.question_key, i]));
  const failed: string[] = [];

  for (const [questionKey, raw] of Object.entries(responses)) {
    const item = byKey.get(questionKey);
    if (!item) { failed.push(questionKey); continue; }   // clave fuera del catálogo

    try {
      await answersApi.create(projectId, administrationId, {
        participant_id: participantId,
        instrument_id: instrumentId,
        question_key: questionKey,
        answer_type: answerTypeFor(item),
        value: answerValue(raw),
        client_generated_id: `${administrationId}:${questionKey}`,  // estable e idempotente
      });
    } catch {
      failed.push(questionKey);
    }
  }
  return { failed };
}
```

### Guardar ítem por ítem o todo al final

| Estrategia | A favor | En contra |
|---|---|---|
| Al avanzar cada ítem | No se pierde trabajo si se cae la conexión | N llamadas durante la entrevista; no hay forma de corregir (ver abajo) |
| Todo al completar | Una ráfaga controlada, se puede revisar antes | Si falla a mitad, quedan respuestas parciales imposibles de corregir |

**La decisión está condicionada por un límite importante: no se puede editar una
respuesta ya enviada.** `UpsertAnswer` no está implementado (G-07): no hay `PATCH`
ni `DELETE` de respuestas. Una vez enviada, queda.

Por eso conviene **recoger todo en el estado local, permitir corregir libremente,
y enviar al final**, justo antes de `complete`. Guardar el borrador en
`localStorage` para sobrevivir recargas:

```ts
const draftKey = `indagio:draft:${administrationId}`;
useEffect(() => localStorage.setItem(draftKey, JSON.stringify(responses)), [responses]);
```

Si se opta por guardar al avanzar, hay que dejar claro en la UI que la respuesta
no se puede cambiar.

### `client_generated_id`: ponerlo, aunque no proteja

Usar una clave estable como `${administrationId}:${questionKey}` deja huella del
intento al que pertenece cada respuesta — útil, porque el `administration_id` no
viene en la respuesta.

⚠️ Pero **la columna no tiene `UNIQUE`**: reenviar el mismo
`client_generated_id` crea una fila duplicada, no un `409`. La idempotencia que
el nombre promete **no existe**. Hay que evitar el reenvío desde el cliente:

```ts
const sent = useRef(new Set<string>());
if (sent.current.has(questionKey)) return;     // ya enviada en esta sesión
sent.current.add(questionKey);
```

Si se envía dos veces el mismo `question_key`, el motor de puntuación se queda
con **la última** fila al armar su mapa por clave: no suma dos veces, pero la
respuesta duplicada queda en los datos y aparecerá en el export.

## `answer_type` debe coincidir con el ítem

```ts
function answerTypeFor(item: ItemResponse): AnswerType {
  // Derivarlo del item_type del catálogo o del kind del instrumento
  if (item.item_type.startsWith("likert")) return "scale";
  if (item.value_map && Object.keys(item.value_map).length) return "single_choice";
  return "text";
}
```

⚠️ Los enums **no coinciden** entre instrumento y respuesta:

| `InstrumentKind` | `AnswerType` |
|---|---|
| `multi_choice` | `multiple_choice` |
| `single_choice` | `single_choice` |
| `scale` | `scale` |
| `text`, `numeric`, `audio`, `video` | `text` (no hay tipos propios) |

Mandar `multi_choice` como `answer_type` devuelve `422`. Y para audio o video no
hay un `answer_type` propio: se usa `text` y el archivo se asocia por
`media_files`.

El backend valida que `answer_type` esté en el enum, pero **no** que tenga
sentido para ese ítem.

## Lo que el backend no valida

Nada de esto se verifica (G-08). Todo es responsabilidad del cliente:

| Sin validar | Consecuencia |
|---|---|
| Que `question_key` exista en el catálogo congelado | Se guarda y el motor la ignora: **dato perdido en silencio** |
| Que la aplicación exista y esté `in_progress` | Se aceptan respuestas sobre intentos ya cerrados |
| Que la aplicación sea del `participant_id` y del `project_id` enviados | Respuestas cruzadas entre sujetos |
| Que `instrument_id` coincida con el de la aplicación | Datos inconsistentes |
| Que no haya ya una respuesta para ese `question_key` | Duplicados |

Guarda recomendada antes de cada envío:

```ts
function assertAnswerIsValid(
  item: ItemResponse | undefined, admin: AdministrationResponse,
  participantId: string, projectId: string,
) {
  if (!item) throw new Error("question_key fuera del catálogo de esta versión");
  if (admin.status !== "in_progress") throw new Error("la aplicación ya está cerrada");
  if (admin.participant_id !== participantId) throw new Error("participante no coincide");
  if (admin.project_id !== projectId) throw new Error("proyecto no coincide");
}
```

## Listar respuestas

```ts
// Todas las del participante en el proyecto
const all = await answersApi.listByParticipant(projectId, participantId);

// Filtradas por instrumento
const phq8 = await answersApi.listByParticipant(projectId, participantId, instrumentId);
```

⚠️ **No se puede filtrar por aplicación** (G-09). Con pretest y postest del mismo
instrumento, las respuestas de ambos intentos vienen mezcladas y **no hay forma
de separarlas**: ni el filtro ni la respuesta incluyen el `administration_id`.

Si el estudio contempla medidas repetidas, es una limitación de diseño a
levantar con backend antes de construir la pantalla de revisión.

Al listar, `AnswerResponse.instrument` viene hidratado con `{id, name, kind}`:
se puede agrupar sin pedir los instrumentos aparte.

```ts
const byInstrument = Object.groupBy(all, a => a.instrument?.name ?? "Sin instrumento");
```

Sin paginación: trae todo. En participantes con muchos instrumentos, filtrar y
paginar en el cliente.

## Registro de media

El archivo se sube **directo a R2** con una URL prefirmada
([10_media.md](10_media.md)); este endpoint solo guarda la metadata.

```ts
// 1. Presign → 2. PUT a R2 → 3. registrar aquí
await answersApi.createMedia(projectId, participantId, {
  answer_id: answer.id,                       // opcional, pero recomendado
  file_key: "videos/raw/abc-123.mp4",         // la clave que devolvió el presign
  mime_type: "video/mp4",
  size_bytes: file.size,                      // debe ser > 0
  duration_seconds: Math.round(durationSec),
  checksum: undefined,
});
```

⚠️ `size_bytes` tiene `binding:"required"` en Go, y en Go `required` **rechaza el
valor cero**: un archivo de 0 bytes devuelve `422`. En la práctica no molesta,
pero explica un `422` desconcertante si se manda `0` por un cálculo fallido.

⚠️ El backend **no verifica que el objeto exista en R2** ni que la clave
corresponda al proyecto. Un `file_key` mal copiado se guarda igual y la metadata
apunta a un archivo inexistente. `status` queda en `pending` para siempre: no hay
confirmación de subida.

## Validación en el cliente

| Campo | Regla del backend |
|---|---|
| `participant_id` | requerido, UUID |
| `question_key` | requerido, no vacío tras recortar |
| `answer_type` | requerido, del enum de respuestas |
| `value` | requerido, JSON válido — **y con la clave `value` dentro** |
| `client_generated_id` | opcional, máx. 128 |
| `file_key` | requerido, no vacío |
| `mime_type` | requerido, no vacío |
| `size_bytes` | requerido, **> 0** |

Más las comprobaciones de la sección anterior, que el backend no hace.

## Errores

| Código | Significado | Acción en UI |
|---|---|---|
| `400` | UUID de ruta o de body malformado | Error de navegación |
| `422` | Falta un campo, `answer_type` inválido, `value` no es JSON, `size_bytes` en 0 | Resaltar el campo |
| `500` | Error interno | Reintentar; la respuesta puede haberse guardado igual |

⚠️ **No existe un error para "respuesta inválida para este instrumento".** Un
`question_key` inexistente, una aplicación cerrada o un participante que no
corresponde: todo devuelve `201`. La ausencia de error no significa que el dato
sea correcto.

## Trampas

1. **`value` va envuelto en `{ "value": X }`.** Cualquier otra forma se acepta y
   rompe el puntaje sin aviso (G-01). Usar un constructor único.
2. **Mandar el número, no la etiqueta**, cuando se pueda: una etiqueta ausente
   del `value_map` hace fallar `complete` con `500`.
3. **No se puede editar ni borrar una respuesta** (G-07). Recoger en el estado
   local y enviar al final.
4. **`client_generated_id` no da idempotencia**: la columna no es única. Evitar
   el reenvío en el cliente.
5. **`AnswerResponse` no trae `administration_id`.** Hay que llevar la cuenta.
6. **No se puede listar por aplicación** (G-09): pretest y postest vienen
   mezclados, sin forma de separarlos.
7. **`multi_choice` (instrumento) vs `multiple_choice` (respuesta)**: no son el
   mismo string.
8. **El backend no valida nada contra el catálogo ni contra la aplicación**
   (G-08). La integridad de los datos de investigación depende del cliente.
9. **`status` y `sync_status` nunca cambian.**
10. **`size_bytes: 0` devuelve `422`** por la semántica de `required` en Go.
11. **La metadata de media no se verifica contra R2**, y su `status` queda
    `pending` para siempre.
12. **Sin paginación** en el listado.
