# 09 · Respuestas y media records

## Propósito

Guarda cada respuesta de un participante como una fila, y registra la metadata
de los archivos de audio o video asociados. Es la tabla que más crece y el
destino final de todo el flujo de recolección. Paso 12 del
[flujo](../2_PROJECT_FLOW.md).

## Entidades y tablas

### `answer_records`

| Columna | Tipo | Nota |
|---|---|---|
| `project_id`, `participant_id` | `UUID` | `ON DELETE CASCADE` |
| `instrument_id` | `UUID` NULL | `ON DELETE SET NULL` |
| `administration_id` | `UUID` NULL | `ON DELETE SET NULL`; agregada en `20260923234729` |
| `question_key` | `VARCHAR(120)` | debe coincidir con un `instrument_items.question_key` |
| `answer_type` | `VARCHAR(32)` | `single_choice` \| `multiple_choice` \| `text` \| `scale` |
| `value` | `JSONB` | **ver el envoltorio abajo** |
| `status` | `VARCHAR(32)` | default `draft` — nunca se mueve |
| `sync_status` | `VARCHAR(32)` | default `pending` — nunca se mueve |
| `client_generated_id` | `VARCHAR(128)` NULL | para idempotencia; **sin constraint `UNIQUE`** |

La columna `questionnaire_id` existió y fue eliminada en la migración
`20260912211033`, junto con la tabla `questionnaires`.

`client_generated_id` tiene índice (`idx_answer_records_client_generated_id`)
pero **no es único**: el mismo id reenviado crea una fila duplicada. El índice
sirve para buscar, no para proteger.

### `media_files`

| Columna | Tipo | Nota |
|---|---|---|
| `project_id`, `participant_id` | `UUID` | |
| `answer_id` | `UUID` NULL | `ON DELETE SET NULL` |
| `file_key` | `VARCHAR(255)` | ruta en R2, **la provee el cliente** |
| `mime_type` | `VARCHAR(80)` | |
| `size_bytes` | `BIGINT` | |
| `duration_seconds` | `INTEGER` NULL | |
| `checksum` | `VARCHAR(128)` NULL | |
| `status` | `VARCHAR(32)` | fijado a `pending`, nunca avanza |

## El envoltorio de `value` — leer antes de integrar

El motor de puntuación lee `{"value": X}` (`internal/domain/scoring.go:18`):

```jsonc
{ "value": 2 }                      // numérico: se usa directo
{ "value": "Varios días" }          // etiqueta: se traduce con el value_map del ítem
{ "value": "texto libre..." }       // ítems con is_scored = false
```

**`requests/answers.http` manda `{"raw": 2}`, que no funciona.** El motor no
encuentra la clave `value`, cuenta el ítem como no respondido y no emite ningún
error. Con el PHQ-8 y su `min_items_required: 6`, el resultado es
`raw_score: null` para las 8 respuestas enviadas.

Es [G-01](00_GAPS.md#g-01--el-envoltorio-de-value-no-coincide-con-el-motor-de-puntuación),
el gap más caro de depurar porque falla en silencio tres pasos más adelante.

## Endpoints

| Método | Ruta | Descripción |
|---|---|---|
| `POST` | `/api/projects/:id/administrations/:administrationId/answers` | Crear una respuesta |
| `GET` | `/api/projects/:id/participants/:participantID/answers` | Listar por participante; `?instrumentId=` |
| `POST` | `/api/projects/:id/participants/:participantID/media` | Registrar metadata de un archivo |

`answer_handler.go:218-226`.

El `GET` filtra **por participante en todo el proyecto**, opcionalmente por
instrumento, nunca por aplicación. `ListAnswersByAdministration` existe en el
repositorio (lo usa el motor de puntuación) pero no está expuesta —
[G-09](00_GAPS.md#g-09--falta-listar-respuestas-por-aplicación-en-http).

## Contratos

```jsonc
// POST .../answers
{
  "participant_id": "<uuid>",
  "instrument_id": "<uuid>",
  "question_key": "phq8_1",
  "answer_type": "scale",
  "value": { "value": 2 },
  "client_generated_id": "client-answer-phq8-1"
}

// POST .../media
{
  "answer_id": "<uuid>",
  "file_key": "uploads/participant-123/audio.mp3",
  "mime_type": "audio/mpeg",
  "size_bytes": 245760,
  "duration_seconds": 15,
  "checksum": "abc123def456"
}
```

`participant_id` va en el body aunque la ruta ya tiene `administrationId`, del
que se podría derivar. Redundante y sin verificación cruzada.

## Reglas de negocio

`CreateAnswer` (`answer_usecase.go:22`) valida:

- ids de actor, proyecto y participante no nulos — nótese que
  **`administration_id` no se valida**, pese a que el mensaje de error lo
  menciona (`:24`);
- `question_key` no vacío tras recortar;
- `value` es JSON bien formado (`json.Valid`);
- `answer_type` pertenece al enum.

Lo que **no** valida, y debería ([G-08](00_GAPS.md#g-08--las-respuestas-no-se-validan-contra-el-catálogo-congelado)):

- que `question_key` exista en el catálogo de la versión congelada — una clave
  equivocada se guarda y luego el motor la ignora al puntuar, así que el dato se
  pierde sin aviso;
- que la aplicación exista, esté `in_progress`, y no `completed`/`abandoned`;
- que la aplicación corresponda al `participant_id` y al `project_id` de la ruta;
- que `instrument_id` coincida con el de la aplicación;
- que no exista ya una respuesta para ese `question_key` en esa aplicación — se
  pueden crear duplicados y el motor se queda con el último que gane en el mapa
  por clave (`scoring.go:25`).

`CreateMedia` valida que `file_key` y `mime_type` no estén vacíos y que
`size_bytes >= 0`. No verifica que el objeto exista en R2 ni que el `file_key`
caiga bajo un prefijo del proyecto.

## Autorización

| | |
|---|---|
| **Implementada** | Solo `actorID != uuid.Nil`. |
| **Esperada** | Miembro del proyecto. Es la tabla con los datos más sensibles del sistema. |

[G-02](00_GAPS.md#g-02--sin-verificación-de-pertenencia-al-proyecto-fuera-del-módulo-projects)
pega más fuerte aquí: `GET /projects/:id/participants/:pid/answers` con ids
ajenos devuelve las respuestas de investigación de otro estudio.

## Dependencias

- **Requiere:** [08_administrations](08_administrations.md) (una aplicación
  abierta), [05_participants](05_participants.md),
  [07_scoring_catalog](07_scoring_catalog.md) (de donde salen los
  `question_key`).
- **Lo consume:** [08_administrations](08_administrations.md) al puntuar,
  [12_exports](12_exports.md) cuando exista.

## Gaps conocidos

- **[G-01](00_GAPS.md#g-01--el-envoltorio-de-value-no-coincide-con-el-motor-de-puntuación)** — envoltorio de `value`. **P0.**
- **[G-02](00_GAPS.md#g-02--sin-verificación-de-pertenencia-al-proyecto-fuera-del-módulo-projects)** — sin verificación de proyecto. **P0.**
- **[G-07](00_GAPS.md#g-07--no-se-puede-editar-una-respuesta-así-que-recompute-no-sirve)** — `UpsertAnswer` devuelve `not implemented` (`answer_repository.go:182`): no hay corrección de respuestas.
- **[G-08](00_GAPS.md#g-08--las-respuestas-no-se-validan-contra-el-catálogo-congelado)** — nada se valida contra el catálogo ni contra la aplicación.
- **[G-09](00_GAPS.md#g-09--falta-listar-respuestas-por-aplicación-en-http)** — falta exponer el listado por aplicación; sin él no se puede reanudar un formulario a medio llenar.
- `client_generated_id` sin `UNIQUE`: la idempotencia que promete no existe.
- `status` y `sync_status` nunca cambian de su valor por defecto.
- `media_files.status` se queda en `pending`: no hay proceso que confirme la
  subida, ni thumbnails, ni transcripción (`docs/1_MAIN_FEATURES.md` §6).
- `ListAnswersByParticipant` no pagina — es la tabla que más crece.

## Requests de ejemplo

[`requests/answers.http`](../../requests/answers.http) — ⚠️ el `value` de los
ejemplos usa `{"raw": N}` y no puntúa; cambiar a `{"value": N}`.
