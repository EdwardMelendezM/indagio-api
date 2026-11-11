# Gaps conocidos — consolidado

Todo lo que falta o está roto para que el flujo de
[2_PROJECT_FLOW.md](../2_PROJECT_FLOW.md) funcione de punta a punta, ordenado
por severidad. Cada entrada apunta al `archivo:línea` donde se verifica.

Estado al 2026-11-10, commit `99dc794`. Documento vivo: al cerrar un gap,
marcarlo y dejar la referencia al commit que lo resolvió.

## Severidad

| | Significado |
|---|---|
| **P0** | Rompe el flujo o expone datos. No se puede ir a producción así. |
| **P1** | Deja un camino sin salida: la feature existe a medias. |
| **P2** | Herencia de otro producto o deuda de consistencia. |

G-16 y G-17 se descubrieron al documentar la integración con el frontend
(`docs/frontend/`) y están clasificados como P2: no rompen el backend, pero
trasladan al cliente trabajo que le corresponde a la API.

---

## P0 — Rompen el flujo

### G-01 · El envoltorio de `value` no coincide con el motor de puntuación

El `ScoringEngine` lee `answer_records.value` esperando `{"value": <número|etiqueta>}`
(`internal/domain/scoring.go:18`, `scoring.go:196`). Los ejemplos del repo mandan
`{"raw": 2}` (`requests/answers.http`).

`extractItemScore` devuelve `ok=false` cuando no encuentra la clave `value`, y
`computeOne` trata ese caso como ítem no respondido — **sin error**. Con la regla
del PHQ-8 (`min_items_required: 6`), las 8 respuestas cuentan como 0 respondidas,
no se alcanza el mínimo y `raw_score` queda `null`.

Impacto: una corrida completa del PHQ-8 con el payload documentado en el repo no
produce puntaje, y no hay nada en la respuesta que indique por qué.

Decisión pendiente: fijar `{"value": X}` como contrato y corregir los `.http`, o
aceptar ambas formas en `extractItemScore`. El comentario en `scoring.go:15`
("This is the one place to change if the actual client payload differs") sugiere
que la segunda opción era la prevista.

### G-02 · Sin verificación de pertenencia al proyecto fuera del módulo `projects`

Solo `usecase/projects` comprueba membresía: `GetProject` recorre `ListMembers` y
devuelve `ErrForbidden` si el actor no está (`project_usecase.go:55-65`).

Los demás usecases solo validan que el UUID no sea nulo:

| Usecase | Chequeo |
|---|---|
| `participants` | `actorID == uuid.Nil` |
| `instruments` | `actorID == uuid.Nil` |
| `instrument_scoring` | `actorID == uuid.Nil` |
| `instrument_administration` | `actorID == uuid.Nil` |
| `answers` | `actorID == uuid.Nil` |
| `exports` | `actorID == uuid.Nil` |

Impacto: cualquier usuario autenticado que conozca (o adivine) un `project_id`
puede listar participantes, leer respuestas, publicar catálogos, iniciar
aplicaciones y pedir exports de un proyecto ajeno. En datos de investigación con
sujetos humanos esto es el hallazgo más grave de la lista.

Arreglo esperado: un helper compartido —`ensureProjectAccess(ctx, actorID, projectID, minRole)`—
inyectado en cada usecase, o un middleware que resuelva la membresía una vez y la
deje en el contexto de Gin.

### G-03 · `/api/sync/batches` está sin autenticación

`RegisterSyncRoutes(api, syncHandler)` se llama sin middleware (`main.go:181`),
a diferencia de todos los demás módulos. El handler intenta leer el actor de un
header `X-Actor-ID` y, si no está, **inventa uno**:

```go
actorID, err := uuid.Parse(c.GetHeader("X-Actor-ID"))
if err != nil || actorID == uuid.Nil {
    actorID = uuid.New() // temporary compatibility for unscoped local validation
}
```

`sync_handler.go:68` y `sync_handler.go:91`.

Impacto: endpoint público de escritura. Cualquiera puede insertar `sync_batches`
contra cualquier `project_id` / `participant_id`, sin credenciales.

---

## P1 — Caminos sin salida

### G-04 · Los lotes de sync nunca se aplican

`SubmitBatch` persiste el `payload` JSONB y responde `201`. No existe consumidor:
nada convierte un lote en `answer_records` ni en `instrument_administrations`.
`MarkBatchProcessed`, `CreateEvent` y `ListEventsByBatch` están implementados en
el repositorio pero no los llama nadie, y la tabla `sync_events` queda siempre
vacía.

Impacto: el soporte offline es de solo escritura. Un dispositivo puede subir su
trabajo de campo y los datos no aparecen en el proyecto.

Falta además definir el esquema del `payload` — hoy es JSONB arbitrario, solo se
valida `json.Valid`.

### G-05 · Los exports nunca se procesan

`RequestExport` crea la fila en `project_exports` con `status = pending`. No hay
`JobTypeProcessExport` en `domain/job.go`, el `JobWorker` no conoce ese tipo, y
`ExportRepository.UpdateStatus` no se llama desde ningún lado.

Impacto: todo export queda `pending` indefinidamente. `file_key` nunca se llena,
no hay nada que descargar.

Además `ExportUsecase.GetExport` está implementado pero no ruteado: no existe
`GET /projects/:id/exports/:exportId`, así que no se puede consultar un export
puntual.

### G-06 · El flujo de invitaciones no se puede cerrar

`ProjectRepository.AcceptInvitation` y `ListInvitations` existen
(`project_repository.go:322` y `:296`) pero no están en `ProjectUsecase` ni
expuestos en HTTP.

Impacto: `POST /projects/:id/members/invite` crea la fila en
`project_invitations` y ahí muere. El invitado no tiene endpoint que llamar. La
única vía real para sumar a alguien es `POST /projects/:id/members` con un
`user_id` ya conocido, lo que obliga a averiguarlo por fuera.

`RemoveMember` tiene el mismo problema: implementado en el repositorio, ausente
del usecase y del router.

### G-07 · No se puede editar una respuesta, así que `recompute` no sirve

`UpsertAnswer` devuelve `not implemented`
(`internal/repository/answers/postgres/answer_repository.go:182`).

Impacto: `POST /administrations/:id/scores/recompute` recalcula sobre exactamente
las mismas respuestas, con el mismo catálogo congelado, así que siempre da el
mismo resultado. Solo tendrá efecto cuando exista corrección de respuestas.

### G-08 · Las respuestas no se validan contra el catálogo congelado

`CreateAnswer` valida que `question_key` no esté vacío y que `answer_type` sea de
la lista (`answer_usecase.go:22-38`). No valida que:

- el `question_key` exista como `instrument_items` de la versión congelada;
- la `administration` exista, esté `in_progress` y no `completed`/`abandoned`;
- la `administration` pertenezca al `participant_id` y al `project_id` de la ruta;
- `instrument_id` coincida con el de la aplicación.

Impacto: se aceptan respuestas con claves inexistentes (el motor las ignora al
puntuar, así que se pierden en silencio), respuestas sobre aplicaciones ya
cerradas, y respuestas cruzadas entre participantes.

### G-09 · Falta listar respuestas por aplicación en HTTP

`AnswerRepository.ListAnswersByAdministration` existe y la usa el motor de
puntuación, pero no hay ruta que la exponga. `GET /projects/:id/participants/:pid/answers`
filtra por participante en todo el proyecto, opcionalmente por `instrumentId`,
nunca por aplicación.

Impacto: el cliente no puede recuperar las respuestas de un intento concreto —
necesario para reanudar un formulario a medio llenar.

---

## P2 — Herencia y consistencia

### G-10 · El módulo `media` es de otro producto

`PresignUploadRequest.TargetType` acepta `oneof=avatar thread comment chat`
(`media_dto.go:4`) y el DTO incluye stickers con cuotas por usuario. Nada de eso
corresponde a Indagio, y falta lo que sí hace falta: audio
(`InstrumentKindAudio` existe en el dominio) y una ruta de presign para media de
respuesta ligada a `project_id` / `participant_id`.

`POST /projects/:id/participants/:pid/media` registra metadata en `media_files`
pero el `file_key` lo provee el cliente: nada verifica que ese objeto exista en
R2 ni que esté bajo un prefijo del proyecto.

### G-11 · Los tipos de job son de otro producto

`internal/delivery/worker/job_worker.go:29-37` registra
`send_push_reaction`, `send_push_comment`, `send_push_new_thread`,
`send_push_chat_message`, `send_push_scholarship_deadline`. De Indagio solo
aplican `send_otp` y `send_password_reset`.

Faltan los que el flujo necesita: procesar exports (G-05), aplicar lotes de sync
(G-04) y enviar invitaciones por correo (G-06).

### G-12 · Los roles de usuario no son los del dominio

`domain/user.go:40-41` define `estudiante` y `moderador`, y la migración los fija
como `ENUM user_role`. `docs/1_MAIN_FEATURES.md` §1 describe `platform_admin`,
`project_owner`, `project_member` e `interviewee`.

Los roles *de proyecto* (`owner`/`admin`/`member` en `project_members`) sí son
correctos y son los que de hecho gobiernan el acceso. El rol global viaja en el
JWT y no se usa para autorizar nada del flujo de investigación.

No existe acceso de participante/entrevistado: no hay token de un solo uso ni
portal propio. Todo pasa por un investigador autenticado, y `GetByCode` es un
endpoint para el investigador, no para el sujeto.

### G-13 · `1_MAIN_FEATURES.md` está desactualizado

- Describe `questionnaires` como pieza central (§5), tabla **eliminada** en la
  migración `20260912211033_delete_questionnaries`. `requests/instruments.http`
  todavía documenta `POST /projects/:id/questionnaires` y
  `.../questionnaires/:id/publish`, rutas que no existen.
- No menciona el catálogo de puntuación versionado
  (`instrument_items` / `instrument_subscales` / `instrument_scoring_rules` /
  `instrument_score_bands`) ni las aplicaciones, que son hoy el corazón del flujo.
- Propone `chi` o `Fiber` (§0, §15); lo implementado es **Gin**.
- Propone `asynq` sobre Redis (§9); lo implementado es una cola en Postgres
  (`background_jobs` + polling cada 3 s).

### G-14 · `status` se ignora al crear un proyecto

`requests/projects.http` manda `"status": "active"` en el `POST /projects`. El
repositorio fija `ProjectStatusDraft` sin leer el body
(`project_repository.go:31`), y `CreateProject` no recibe el parámetro. Hace
falta el `PATCH` del paso 5 para activarlo.

Hay que decidir si el campo se acepta al crear o se quita del DTO; hoy el cliente
cree que funciona.

### G-15 · Sin transiciones de estado validadas

Ni `projects.status`, ni `participants.status`, ni `instruments.status` validan la
transición: solo que el valor pertenezca al enum. Un proyecto `archived` puede
volver a `active`; un participante `completed` puede volver a `pending`.

`instrument_administrations` es la excepción parcial: `CompleteAdministration`
rechaza con `409` si ya estaba `abandoned`
(`instrument_administration.go:44`), pero `AbandonAdministration` no verifica
nada y puede abandonar una aplicación ya completada, dejando los
`instrument_scores` huérfanos de un estado coherente.

### G-16 · Las violaciones de unicidad no se traducen a `409`

Los repositorios envuelven el error de constraint de Postgres sin inspeccionarlo
(`fmt.Errorf("insert participant: %w", err)`), así que sale como **`500`**.

En toda la API solo hay **tres** fuentes reales de `409`:

| Endpoint | Origen |
|---|---|
| `POST /auth/register` | `ErrUserAlreadyExists` (`auth_usecase.go:70`) |
| `POST /projects/:id/members` | `ErrConflict` (`project_usecase.go:192`) |
| `PATCH /administrations/:id/complete` | `ErrConflict` (`instrument_administration.go:44`) |

Quedan sin traducir, devolviendo `500`:

- `participants.external_identifier` duplicado en el proyecto;
- `instrument_administrations.client_generated_id` duplicado;
- `sync_batches.client_generated_batch_id` duplicado;
- `instrument_items.question_key` duplicado al publicar un catálogo.

Impacto: los tres `client_generated_id` existen precisamente para que el cliente
pueda **reintentar** tras un corte de red. Ese reintento es el camino feliz, y
hoy devuelve un error indistinguible de una caída del servidor. El cliente tiene
que hacer una llamada extra de listado para saber si su operación ya se había
aplicado — ver `docs/frontend/08_administrations.md`.

Arreglo esperado: inspeccionar el `*pq.Error` con código `23505` en los
repositorios y devolver `domain.ErrConflict`.

### G-17 · El detalle de los errores se descarta antes de llegar al cliente

Los usecases construyen mensajes precisos y los envuelven sobre un centinela:

```go
return nil, fmt.Errorf("item %q references unknown subscale %q: %w",
    it.QuestionKey, *it.SubscaleKey, domain.ErrValidation)
```

Pero `utils.HandleError` responde con `appErr.Message` —el mensaje del
centinela— y adjunta `Detail` solo si no está vacío. **`AppError.Detail` no se
rellena en ningún punto del código.** El cliente siempre recibe:

```json
{ "error": "validation failed" }
```

Impacto: ningún `422` indica qué campo falló, ni siquiera cuando el backend lo
sabe con exactitud. Para el constructor de catálogos de puntuación —el body más
complejo de la API, con cuatro arrays anidados— significa que un error en
cualquier parte devuelve el mismo mensaje opaco. Toda la validación útil tiene
que duplicarse en el cliente.

Arreglo esperado: rellenar `Detail` con el mensaje envuelto, o devolver errores
de validación por campo.

---

## Resumen por feature

| Feature | Gaps |
|---|---|
| [01_auth](01_auth.md) | G-12 |
| [04_projects](04_projects.md) | G-06, G-14, G-15, G-17 |
| [05_participants](05_participants.md) | G-02, G-12, G-15, G-16 |
| [06_instruments](06_instruments.md) | G-02, G-13, G-15 |
| [07_scoring_catalog](07_scoring_catalog.md) | G-02, G-16, G-17 |
| [08_administrations](08_administrations.md) | G-01, G-02, G-07, G-15, G-16 |
| [09_answers](09_answers.md) | G-01, G-02, G-07, G-08, G-09 |
| [10_media_storage](10_media_storage.md) | G-10 |
| [11_offline_sync](11_offline_sync.md) | G-03, G-04, G-16 |
| [12_exports](12_exports.md) | G-02, G-05 |
| [13_background_jobs](13_background_jobs.md) | G-11 |
