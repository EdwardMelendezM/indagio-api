# 02 · Perfiles de usuario

## Propósito

Lectura y edición mínima del perfil de los investigadores: perfil público,
listado de usuarios disponibles (para armar equipos de proyecto) y cambio de
nombre. Módulo auxiliar: no participa del flujo de recolección de datos.

## Entidades y tablas

Opera sobre `users`, descrita en [01_auth](01_auth.md#entidades-y-tablas).

## Endpoints

| Método | Ruta | Auth | Descripción |
|---|---|---|---|
| `GET` | `/api/users/:id` | — | Perfil público |
| `GET` | `/api/users/available` | Bearer | Usuarios verificados, no bloqueados, no borrados |
| `PATCH` | `/api/users/:id/name` | Bearer | Cambia el nombre |

`user_handler.go:155-160`. `:id` es público por diseño; las otras dos llevan
`authMiddleware` a nivel de ruta.

⚠️ Gin resuelve `/users/available` contra el parámetro `:id` del grupo. El
orden de registro importa y hoy funciona, pero cualquier ruta literal nueva bajo
`/users` corre riesgo de colisión. Vale la pena mover `available` fuera del
grupo paramétrico.

## Reglas de negocio

- `ListAvailable(excludeUserID, search, limit, offset)` excluye al solicitante y
  filtra por `verified = true AND blocked = false AND deleted_at IS NULL`.
  Devuelve la página pedida más el total de coincidencias, para paginar en el
  cliente.
- `UpdateUserName` (`auth_usecase.go:202`) compara `requesterID` con
  `targetUserID`: solo el propio usuario puede cambiar su nombre. **Es el único
  chequeo de autoría correcto fuera del módulo `projects`.**
- `ListAll` existe en el repositorio para gestión administrativa (incluye
  bloqueados) y lo usa [03_admin](03_admin.md).

## Autorización

| | |
|---|---|
| **Implementada** | `/users/:id` público; `/available` y `/:id/name` exigen Bearer. El cambio de nombre verifica que el solicitante sea el dueño. |
| **Esperada** | Correcta. Revisar si el perfil público debe exponer el correo — hoy `GetPublicProfile` responde sin autenticación. |

## Dependencias

- **Requiere:** [01_auth](01_auth.md).
- **Lo consume:** [04_projects](04_projects.md) — `/users/available` es la forma
  de encontrar el `user_id` que `POST /projects/:id/members` necesita, dado que
  el flujo de invitaciones no se puede cerrar ([G-06](00_GAPS.md#g-06--el-flujo-de-invitaciones-no-se-puede-cerrar)).

## Gaps conocidos

- `GetPublicProfile` no requiere autenticación y devuelve datos del usuario.
  Revisar qué campos corresponde exponer públicamente.
- La gestión de avatares (`UpdateAvatarURL`, `ClearAvatar`, `AvatarVariantURL`)
  está completa en repositorio y storage, pero no hay endpoint que la use: es
  herencia del producto del que salió este módulo. Relacionado con
  [G-10](00_GAPS.md#g-10--el-módulo-media-es-de-otro-producto).
- `/users/available` convive con `:id` en el mismo grupo paramétrico.

## Requests de ejemplo

[`requests/users.http`](../../requests/users.http)
