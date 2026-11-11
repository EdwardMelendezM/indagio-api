# Flujo completo de un proyecto nuevo

Secuencia real de llamadas para llevar un proyecto de investigación desde cero
hasta tener puntajes calculados y un export solicitado. Derivado del código, no
del diseño: todo lo que aparece aquí existe y está ruteado en `main.go`.

Base URL: `http://localhost:8080/api` · Auth: `Authorization: Bearer <access_token>`

## Mapa de dependencias

```
auth ──> projects ──┬──> participants ──┐
                    │                   │
                    └──> instruments ───┼──> scoring_catalog ──> administrations ──> answers
                                        │                              │
                                        │                              └──> instrument_scores
                                        └──> exports
```

La dependencia crítica y menos obvia: **`administrations` requiere que el
instrumento tenga al menos una versión de catálogo publicada**. Sin ese paso,
`StartAdministration` falla al resolver `GetLatestCatalog`
(`internal/usecase/instrument_administration/instrument_administration.go:29`).

## Los 13 pasos

### 1 · Registro

```http
POST /api/auth/register
Content-Type: application/json

{ "name": "Investigador", "email": "inv@unsaac.edu.pe", "password": "secret123" }
```

Crea el `users` row con `verified = false` y encola un job `send_otp` en
`background_jobs`. El `JobWorker` lo toma en ≤3 s y envía el correo.

### 2 · Verificación de OTP

```http
POST /api/auth/verify-otp
{ "email": "inv@unsaac.edu.pe", "code": "123456" }
```

Código de 6 dígitos (`binding:"len=6"`). Marca `verified = true` y devuelve
`access_token` + `refresh_token`.

### 3 · Login

```http
POST /api/auth/login
{ "email": "inv@unsaac.edu.pe", "password": "secret123" }
```

El `access_token` lleva `user_id` y `role` en los claims. Todos los pasos
siguientes lo requieren.

### 4 · Crear el proyecto

```http
POST /api/projects
{ "name": "Depresión, ansiedad y estrés 2026-2027", "description": "..." }
```

⚠️ El proyecto **siempre** nace en `draft`: el repositorio fija el estado y
descarta cualquier `status` del body (`project_repository.go:31`). El creador
queda automáticamente como `project_members` con rol `owner`
(`project_usecase.go:38`).

### 5 · Activar el proyecto

```http
PATCH /api/projects/{projectId}
{ "status": "active" }
```

Estados válidos: `draft` → `active` → `paused` → `closed` → `archived`.
No hay validación de transiciones; cualquier estado puede ir a cualquier otro.

### 6 · Agregar el equipo

```http
POST /api/projects/{projectId}/members
{ "user_id": "<uuid>", "role": "member" }
```

Roles dentro del proyecto: `owner`, `admin`, `member`. Solo `owner` y `admin`
pueden agregar o invitar.

> `POST /projects/{id}/members/invite` crea la fila en `project_invitations`
> pero **no existe endpoint para aceptarla** — ver `features/00_GAPS.md` → G-06.
> Para sumar gente al proyecto hoy hay que usar `/members` con el `user_id`.

### 7 · Crear participantes

```http
POST /api/projects/{projectId}/participants
{ "display_name": "Participante 001", "external_identifier": "EST-2026-001" }
```

El backend genera el `code` de 8 caracteres sobre el alfabeto
`ABCDEFGHJKLMNPQRSTUVWXYZ23456789` (sin `I`, `O`, `0`, `1` para evitar
ambigüedad al dictarlo o transcribirlo). Único por proyecto.

Estados del participante: `pending` → `active` → `completed` | `expired`.

### 8 · Crear el instrumento

```http
POST /api/projects/{projectId}/instruments
{
  "name": "PHQ-8",
  "kind": "single_choice",
  "config": {
    "description": "Patient Health Questionnaire - 8 items",
    "response_labels": ["Ningún día", "Varios días", "Más de la mitad de los días", "Casi todos los días"]
  }
}
```

`kind` ∈ `text`, `numeric`, `scale`, `single_choice`, `multi_choice`, `audio`,
`video`. `config` es JSONB libre: no se valida su forma contra el `kind`.

### 9 · Publicar el catálogo de puntuación ← **paso bisagra**

```http
POST /api/projects/{projectId}/instruments/{instrumentId}/versions
{
  "items": [
    { "question_key": "phq8_1", "order_index": 1, "item_type": "likert_0_3",
      "weight": 1, "is_scored": true, "value_map": { "0": 0, "1": 1, "2": 2, "3": 3 } }
    // ... phq8_2 .. phq8_8
  ],
  "subscales": [],
  "rules": [
    { "subscale_key": null, "algorithm": "sum", "min_items_required": 6,
      "missing_strategy": "prorate", "config": {} }
  ],
  "bands": [
    { "subscale_key": null, "min_score": 0,  "max_score": 4,  "label": "Mínima" },
    { "subscale_key": null, "min_score": 5,  "max_score": 9,  "label": "Leve" },
    { "subscale_key": null, "min_score": 10, "max_score": 14, "label": "Moderada" },
    { "subscale_key": null, "min_score": 15, "max_score": 19, "label": "Moderadamente severa" },
    { "subscale_key": null, "min_score": 20, "max_score": 24, "label": "Severa" }
  ]
}
```

Incrementa `instruments.version` transaccionalmente e inserta cada fila
etiquetada con la versión nueva. Nunca muta versiones anteriores: por eso una
aplicación vieja siempre se puede recalcular contra el catálogo con el que fue
puntuada originalmente.

El orden dentro del body importa para la validación: todo `subscale_key` que
aparezca en `items`, `rules` o `bands` debe existir en el array `subscales` del
mismo request. La regla con `subscale_key: null` es el puntaje total y es
obligatoria (al menos una regla).

### 10 · Iniciar la aplicación

```http
POST /api/projects/{projectId}/participants/{participantId}/instruments/{instrumentId}/administrations
{ "client_generated_id": "local-admin-uuid-001" }
```

Resuelve `GetLatestCatalog` y **congela** ese número en
`instrument_administrations.instrument_version`. El `client_generated_id` da
idempotencia por `UNIQUE (project_id, participant_id, client_generated_id)`.

Guardar el `id` devuelto: es el `administrationId` de los pasos 11-13.

### 11 · Renderizar el formulario de la versión congelada

```http
GET /api/projects/{projectId}/instruments/{instrumentId}/versions?version=2
```

Usar siempre el `instrument_version` que devolvió el paso 10, **no** la última.
Si entre el inicio y el fin de la entrevista alguien publica una versión nueva,
pedir "la última" mostraría un formulario distinto al que se está puntuando.

### 12 · Enviar respuestas

```http
POST /api/projects/{projectId}/administrations/{administrationId}/answers
{
  "participant_id": "<uuid>",
  "instrument_id": "<uuid>",
  "question_key": "phq8_1",
  "answer_type": "scale",
  "value": { "value": 2 },
  "client_generated_id": "client-answer-phq8-1"
}
```

Una llamada por `question_key`. `answer_type` ∈ `single_choice`,
`multiple_choice`, `text`, `scale`.

⚠️ **El envoltorio de `value` importa.** El motor de puntuación lee
`{"value": X}` (`internal/domain/scoring.go:18-20`). `requests/answers.http` manda
`{"raw": 2}`, que el motor descarta en silencio como "sin respuesta" — ver
`features/00_GAPS.md` → **G-01**. Con el payload equivocado, el paso 13 devuelve
`raw_score: null` sin ningún error.

`value` puede llevar un número (se usa tal cual) o una etiqueta string (se
traduce con el `value_map` del item; si la etiqueta no está en el mapa, el
cálculo falla con error).

### 13 · Completar y puntuar

```http
PATCH /api/administrations/{administrationId}/complete
```

Encadena: catálogo de la versión congelada → `ListAnswersByAdministration` →
`ScoringEngine.Compute` → `instrument_scores`. Responde con la aplicación y los
puntajes juntos. Rechaza con `409` si la aplicación ya estaba `abandoned`.

Alternativa si el participante no terminó: `PATCH .../abandon` (no calcula nada).

### Después

```http
GET  /api/administrations/{administrationId}/scores      # lee lo guardado, no recalcula
POST /api/administrations/{administrationId}/scores/recompute
PATCH /api/projects/{projectId}/participants/{participantId}/status   # → "completed"
POST /api/projects/{projectId}/exports                   # { "format": "csv" }
GET  /api/projects/{projectId}/exports
```

`recompute` hoy no puede cambiar nada: no existe forma de editar una respuesta
ya guardada (`UpsertAnswer` devuelve `not implemented`). El export crea la fila
y se queda en `pending` para siempre: no hay job que lo procese. Ver G-07 y G-05.

## Forma del puntaje resultante

```json
[
  {
    "subscale_key": null,
    "raw_score": 11,
    "scaled_score": null,
    "items_answered": 8,
    "items_expected": 8,
    "band_label": "Moderada",
    "computed_at": "2026-11-10T14:23:01Z"
  }
]
```

`subscale_key: null` es el puntaje total. Un instrumento con subescalas devuelve
una fila por subescala más la del total.

`raw_score: null` significa que `min_items_required` no se cumplió — o que las
respuestas llegaron con el envoltorio equivocado (G-01).

## Verificación rápida del flujo

```bash
make run              # levanta la API con swagger regenerado
curl localhost:8080/healthz
open http://localhost:8080/swagger/index.html
```

Los archivos en `requests/` cubren cada paso en orden; `requests/bootstrap.http`
es el punto de partida. Están escritos para el HTTP client de JetBrains.
