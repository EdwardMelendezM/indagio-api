# 04 · Proyectos, miembros e invitaciones

## Propósito

El proyecto es el contenedor de todo: participantes, instrumentos, respuestas y
exports cuelgan de él, y la membresía en el proyecto es (debería ser) la unidad
de autorización de la plataforma. Pasos 4-6 del [flujo](../2_PROJECT_FLOW.md).

**Este es el único módulo con autorización completa.** El patrón de
`GetProject` es el que los demás deberían replicar — ver
[G-02](00_GAPS.md#g-02--sin-verificación-de-pertenencia-al-proyecto-fuera-del-módulo-projects).

## Entidades y tablas

### `projects`

| Columna | Tipo | Nota |
|---|---|---|
| `id` | `UUID` PK | |
| `owner_id` | `UUID` → `users` | `ON DELETE CASCADE` |
| `name` | `VARCHAR(160)` | 2-160 caracteres |
| `description` | `TEXT` NULL | |
| `status` | `VARCHAR(32)` | default `draft` |
| `archived_at` / `deleted_at` | `TIMESTAMPTZ` NULL | borrado lógico |

### `project_members`

| Columna | Tipo | Nota |
|---|---|---|
| `project_id`, `user_id` | `UUID` | `UNIQUE(project_id, user_id)` |
| `role` | `VARCHAR(32)` | `owner` \| `admin` \| `member` |
| `status` | `VARCHAR(32)` | default `active` |
| `invited_by` | `UUID` NULL | `ON DELETE SET NULL` |

### `project_invitations`

| Columna | Tipo | Nota |
|---|---|---|
| `email` | `TEXT` | el invitado puede no tener cuenta aún |
| `role` | `VARCHAR(32)` | `admin` \| `member` (nunca `owner`) |
| `status` | `VARCHAR(32)` | default `pending` |
| `expires_at` | `TIMESTAMPTZ` | obligatorio, debe ser futuro |

## Ciclo de vida

```
draft ──> active ──> paused ──> closed ──> archived
```

Enum en `domain/project.go:14-20`. **Las transiciones no se validan**: cualquier
estado puede ir a cualquier otro, incluido `archived` → `active`. Ver
[G-15](00_GAPS.md#g-15--sin-transiciones-de-estado-validadas).

`archived_at` lo fija `POST /:id/archive`; `deleted_at`, el `DELETE`. Ambos son
borrado lógico: `GetByID` filtra por `deleted_at IS NULL`.

## Endpoints

| Método | Ruta | Rol mínimo | Descripción |
|---|---|---|---|
| `POST` | `/api/projects` | autenticado | Crear |
| `GET` | `/api/projects` | autenticado | Listar los propios, filtro `?status=` |
| `GET` | `/api/projects/:id` | miembro | Detalle |
| `PATCH` | `/api/projects/:id` | owner/admin | Actualizar nombre, descripción, estado |
| `DELETE` | `/api/projects/:id` | owner | Borrado lógico |
| `POST` | `/api/projects/:id/archive` | owner/admin | Archivar |
| `POST` | `/api/projects/:id/members` | owner/admin | Agregar miembro por `user_id` |
| `POST` | `/api/projects/:id/members/invite` | owner/admin | Invitar por correo |
| `GET` | `/api/projects/:id/members` | miembro | Listar miembros |

`project_handler.go:438-449`.

## Contratos

```jsonc
// POST /api/projects
{ "name": "Depresión, ansiedad y estrés 2026-2027", "description": "..." }
// status en el body se IGNORA — ver G-14

// POST /api/projects/:id/members
{ "user_id": "<uuid>", "role": "member" }

// POST /api/projects/:id/members/invite
{ "email": "inv@unsaac.edu.pe", "role": "member", "expires_at": "2026-12-31T23:59:59Z" }
```

`ProjectMember` en las respuestas viene hidratado con datos del usuario
(`UserEmail`, `UserName`, `UserRole`, `UserCreatedAt`) por un join en el
repositorio: el cliente no necesita una segunda llamada para pintar la lista.

## Reglas de negocio

- **Creación** (`project_usecase.go:22`): valida nombre 2-160, crea el proyecto
  en `draft`, y a continuación inserta la membresía `owner` del creador
  (`:38`). Son **dos operaciones sin transacción**: si `AddMember` falla, queda
  un proyecto huérfano sin miembros, invisible para su propio creador (porque
  `ListProjects` y `GetProject` resuelven por membresía).
- **Lectura** (`:45`): trae el proyecto, lista miembros, y busca al solicitante.
  Si no está → `403 ErrForbidden`. Patrón correcto.
- **Invitación** (`:199`): valida correo no vacío, rol ∈ {`admin`, `member`} —
  no se puede invitar como `owner` —, `expires_at` futuro, y que el actor sea
  `owner` o `admin` del proyecto (`:214-221`).
- El `owner` es único y vive en `projects.owner_id` además de en
  `project_members`. `domain/project.go:36` lo describe como transferible pero
  no hay endpoint de transferencia.

## Autorización

| | |
|---|---|
| **Implementada** | Completa. `GetProject` exige membresía; `UpdateProject`, `ArchiveProject`, `AddMember` e `InviteMember` exigen `owner` o `admin`; `DeleteProject` exige `owner`. |
| **Esperada** | Lo mismo. Nada que corregir aquí. |

## Dependencias

- **Requiere:** [01_auth](01_auth.md); [02_users](02_users.md) en la práctica,
  para obtener el `user_id` que `/members` necesita.
- **Lo consume:** todo. `project_id` es el ancla de participantes, instrumentos,
  respuestas, sync y exports.

## Gaps conocidos

- **[G-06](00_GAPS.md#g-06--el-flujo-de-invitaciones-no-se-puede-cerrar)** —
  `AcceptInvitation` (`project_repository.go:322`), `ListInvitations` (`:296`) y
  `RemoveMember` (`:222`) están implementados en el repositorio pero no llegan al
  usecase ni al router. La invitación crea la fila y muere ahí; tampoco se envía
  correo al invitado.
- **[G-14](00_GAPS.md#g-14--status-se-ignora-al-crear-un-proyecto)** — el
  `status` del body del `POST` se descarta en silencio; el proyecto siempre nace
  `draft`.
- **[G-15](00_GAPS.md#g-15--sin-transiciones-de-estado-validadas)** — sin
  máquina de estados: `archived` puede volver a `active`.
- `CreateProject` no es transaccional (proyecto + membresía del owner).
- No hay transferencia de propiedad pese a estar prevista en el dominio.
- `GET /api/projects` resuelve por membresía vía `ListByUser`, sin paginación:
  devuelve todo de una vez.

## Requests de ejemplo

[`requests/projects.http`](../../requests/projects.http)
