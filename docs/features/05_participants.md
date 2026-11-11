# 05 · Participantes y códigos

## Propósito

Cada participante es un sujeto de investigación dentro de un proyecto,
identificado por un **código corto** en lugar de datos personales. El código es
la pieza de seudonimización: permite referirse al sujeto en campo sin exponer su
identidad. Paso 7 del [flujo](../2_PROJECT_FLOW.md).

## Entidades y tablas

### `participants`

| Columna | Tipo | Nota |
|---|---|---|
| `id` | `UUID` PK | |
| `project_id` | `UUID` → `projects` | `ON DELETE CASCADE` |
| `code` | `VARCHAR(32)` | `UNIQUE(project_id, code)` |
| `display_name` | `VARCHAR(160)` | 2-160 caracteres |
| `external_identifier` | `VARCHAR(160)` NULL | `UNIQUE(project_id, external_identifier)` |
| `status` | `VARCHAR(32)` | default `pending` |
| `last_access_at` | `TIMESTAMPTZ` NULL | columna presente, **nunca se escribe** |

`external_identifier` es el puente con el sistema de origen (código de
matrícula, historia clínica). Único por proyecto, así que la misma persona no
se puede cargar dos veces por error.

## Generación del código

`participant_usecase.go:86` — 8 caracteres sobre el alfabeto
`ABCDEFGHJKLMNPQRSTUVWXYZ23456789`, usando `crypto/rand`.

Sin `I`, `O`, `0` ni `1`: el código se dicta en voz alta y se transcribe a mano
en campo, donde esos caracteres se confunden. 32⁸ ≈ 1.1 × 10¹² combinaciones.

Si `crypto/rand` falla, cae a `CODE-<nanos % 1000000>` (`:93`) — formato distinto
y con riesgo real de colisión, aunque se apoya en el `UNIQUE` de la tabla.
**No hay reintento ante colisión**: un choque devuelve el error de constraint
tal cual.

## Ciclo de vida

```
pending ──> active ──> completed
                   └─> expired
```

`domain/participant.go:13-18`. Solo se valida que el valor pertenezca al enum;
cualquier transición es aceptada, incluido `completed` → `pending`.

Ningún otro módulo actualiza este estado: iniciar o completar una aplicación no
mueve al participante de `pending`. Hay que hacerlo explícitamente con el `PATCH`.

## Endpoints

| Método | Ruta | Descripción |
|---|---|---|
| `POST` | `/api/projects/:id/participants` | Crear, con código autogenerado |
| `GET` | `/api/projects/:id/participants` | Listar los del proyecto |
| `GET` | `/api/projects/:id/participants/code/:code` | Buscar por código |
| `PATCH` | `/api/projects/:id/participants/:participantID/status` | Cambiar estado |

`participant_handler.go:211-217`. Todas con `authMiddleware`.

## Contratos

```jsonc
// POST
{ "display_name": "Participante 001", "external_identifier": "EST-2026-001" }

// Respuesta
{ "id": "...", "project_id": "...", "code": "MXSK2EBN", "display_name": "...",
  "external_identifier": "EST-2026-001", "status": "pending",
  "created_at": "...", "last_access_at": null }

// PATCH .../status
{ "status": "active" }
```

## Reglas de negocio

- `display_name` entre 2 y 160 caracteres; `external_identifier` opcional, hasta
  160.
- `GetParticipantByCode` normaliza a mayúsculas y recorta espacios
  (`participant_usecase.go:60`), así que el código se puede teclear en minúsculas.
- La carga masiva por CSV que pide `docs/1_MAIN_FEATURES.md` §3 no existe: hay
  que crear uno por llamada.

## Autorización

| | |
|---|---|
| **Implementada** | Solo `actorID != uuid.Nil`. **No se verifica membresía en el proyecto.** |
| **Esperada** | Miembro del proyecto para leer; `owner`/`admin` para crear y cambiar estado. |

Es el caso más sensible de [G-02](00_GAPS.md#g-02--sin-verificación-de-pertenencia-al-proyecto-fuera-del-módulo-projects):
`GET /projects/:id/participants` con un `project_id` ajeno devuelve el padrón
completo de otro estudio — códigos, nombres e identificadores externos.

`UpdateParticipantStatus` es peor aún: recibe solo `participantID`, sin
`project_id`, así que ni siquiera podría verificar el proyecto sin una consulta
extra.

## Dependencias

- **Requiere:** [04_projects](04_projects.md).
- **Lo consume:** [08_administrations](08_administrations.md),
  [09_answers](09_answers.md), [11_offline_sync](11_offline_sync.md).

## Gaps conocidos

- **[G-02](00_GAPS.md#g-02--sin-verificación-de-pertenencia-al-proyecto-fuera-del-módulo-projects)** — sin verificación de proyecto. Crítico aquí.
- **[G-15](00_GAPS.md#g-15--sin-transiciones-de-estado-validadas)** — sin máquina de estados.
- **[G-12](00_GAPS.md#g-12--los-roles-de-usuario-no-son-los-del-dominio)** — el
  código existe pero no habilita nada: no hay token de acceso para el
  participante ni portal donde se autoadministre un instrumento. `GetByCode` es
  una búsqueda para el investigador.
- `last_access_at` nunca se escribe. Quedaría natural actualizarla al iniciar una
  aplicación.
- Sin reintento ante colisión de código; el fallback de `crypto/rand` genera un
  formato distinto.
- Sin importación masiva CSV.
- `ListByProject` no pagina.

## Requests de ejemplo

[`requests/participants.http`](../../requests/participants.http)
