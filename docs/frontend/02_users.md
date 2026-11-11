# 02 · Perfiles y buscador de usuarios

## Pantallas

Perfil público · selector de usuarios para agregar al equipo de un proyecto ·
edición del propio nombre.

## Tipos

```ts
interface PublicUserResponse {
  id: string;
  name: string;
  role: UserRole;
  avatar_url: string | null;
  created_at: string;
  // Sin email ni blocked: ver nota abajo.
}

// GET /users/available — el ÚNICO endpoint paginado de la API
interface ListAvailableUsersResponse {
  data: UserResponse[];   // la misma UserResponse de /auth/me (con email)
  total: number;
  page: number;
  limit: number;
}

interface UpdateUserNameRequest  { name: string }
interface UpdateUserNameResponse { message: string }
```

`PublicUserResponse` omite a propósito `email` (dato personal, solo el dueño lo
ve vía `/auth/me`) y `blocked` (estado interno de moderación). El DTO lo
documenta explícitamente en `user_dto.go`.

## Endpoints

| Método | Ruta | Auth | Respuesta |
|---|---|---|---|
| `GET` | `/users/:id` | — **público** | `200 PublicUserResponse` |
| `GET` | `/users/available?page&limit&q` | Bearer | `200 ListAvailableUsersResponse` |
| `PATCH` | `/users/:id/name` | Bearer | `200 UpdateUserNameResponse` |

```ts
export const usersApi = {
  publicProfile: (id: string) =>
    api.get<PublicUserResponse>(`/users/${id}`),

  available: (params: { page?: number; limit?: number; q?: string } = {}) => {
    const qs = new URLSearchParams();
    if (params.page)  qs.set("page",  String(params.page));
    if (params.limit) qs.set("limit", String(params.limit));
    if (params.q)     qs.set("q",     params.q);
    const suffix = qs.toString() ? `?${qs}` : "";
    return api.get<ListAvailableUsersResponse>(`/users/available${suffix}`);
  },

  updateName: (id: string, name: string) =>
    api.patch<UpdateUserNameResponse>(`/users/${id}/name`, { name }),
};
```

## Query de `/users/available`

| Parámetro | Default | Límites |
|---|---|---|
| `page` | `1` | mínimo 1 |
| `limit` | `20` | 1-100 |
| `q` | `""` | máximo 80 caracteres |

Devuelve usuarios **verificados, no bloqueados y no eliminados**, excluyendo
siempre al solicitante. Con `q` vacío devuelve todos (paginados).

Como es el único endpoint con paginación de la API, conviene no generalizar el
patrón: el resto de los listados son arrays pelados.

```ts
// Selector de miembros con búsqueda
const { data: users, total } = await usersApi.available({ q: search, page, limit: 20 });
const hasMore = page * 20 < total;
```

## Flujo: agregar un miembro al proyecto

Este módulo existe sobre todo para resolver un problema del flujo de proyectos:
`POST /projects/:id/members` pide un `user_id`, y no hay forma de obtenerlo por
correo. El camino viable es:

```
1. El usuario escribe un nombre o correo en el buscador
2. GET /users/available?q=<texto>       → lista de candidatos con su id
3. El usuario elige uno
4. POST /projects/:id/members { user_id, role }
```

El camino "invitar por correo" existe (`POST /projects/:id/members/invite`) pero
**no se puede completar**: no hay endpoint para aceptar la invitación ni se
manda correo al invitado (G-06). Ver [04_projects.md](04_projects.md).

Para filtrar a los que ya son miembros, cruzar con `GET /projects/:id/members`
en el cliente: el backend no lo hace.

```ts
const memberIds = new Set(members.map(m => m.user.id));
const candidates = users.filter(u => !memberIds.has(u.id));
```

## Editar el propio nombre

```ts
await usersApi.updateName(currentUser.id, "Nombre Nuevo");
```

El backend verifica que el solicitante sea el dueño del perfil: pasar el `id` de
otro usuario devuelve error. **Es la única verificación de autoría correcta
fuera del módulo de proyectos.** No existe endpoint para que un admin cambie el
nombre de otro.

La respuesta es `{ message: "name updated" }`, **no el usuario actualizado**:
hay que actualizar el estado local a mano o volver a pedir `/auth/me`.

## Validación en el cliente

| Campo | Regla |
|---|---|
| `name` | requerido, 2-100 caracteres |
| `q` | máximo 80 caracteres |
| `page` | entero ≥ 1 |
| `limit` | entero 1-100 |

⚠️ `name` admite 2-100 aquí, pero el registro admite 2-**80**. Son dos límites
distintos para el mismo campo: conviene usar 80 en toda la UI para no permitir
un nombre que después no se puede volver a enviar desde otro formulario.

Un `limit` fuera de rango o un `page` en 0 devuelve `422`, no se recorta en
silencio.

## Errores

| Código | Endpoint | Significado |
|---|---|---|
| `400` | todos | `:id` no es un UUID válido |
| `404` | `/users/:id` | El usuario no existe |
| `403` | `/users/:id/name` | Intentar cambiar el nombre de otro |
| `422` | `/users/available` | `page`, `limit` o `q` fuera de rango |

## Trampas

1. **`/users/available` es el único endpoint paginado** y el único que envuelve
   en `{data, total, page, limit}`. Todo el resto devuelve arrays pelados.
2. **`GET /users/:id` es público**: funciona sin token. No usarlo como prueba de
   que hay sesión.
3. **`updateName` devuelve un mensaje, no el usuario.** Refrescar el estado local.
4. **`available` excluye al propio usuario**: no aparecerá en su propio buscador.
5. **El límite de `name` difiere entre registro (80) y edición (100).**
6. **`/users/available` convive con `/users/:id` en el mismo grupo de rutas.**
   Hoy funciona por el orden de registro en Gin, pero una ruta literal nueva bajo
   `/users` puede colisionar con el parámetro. Si un día `/users/available`
   empieza a responder `400 invalid user id`, esta es la causa.
7. **La gestión de avatares no tiene endpoint.** `avatar_url` llega en las
   respuestas y el backend tiene toda la maquinaria (variantes `thumb`/`medium`/
   `full`, versionado para cache-busting), pero no hay ruta para subir ni borrar
   un avatar. Hoy ese campo siempre es `null` (G-10).
