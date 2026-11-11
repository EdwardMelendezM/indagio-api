# 04 · Proyectos, miembros e invitaciones

## Pantallas

Lista de proyectos · creación · detalle y edición · gestión del equipo ·
archivar y eliminar.

**Es el único módulo donde el backend sí verifica permisos**, y por eso la
fuente de verdad de los permisos de toda la interfaz.

## Tipos

```ts
// ── Requests ───────────────────────────────────────────────────────────────
interface CreateProjectRequest {
  name: string;                 // 2-160
  description?: string;
  // ⚠️ "status" se IGNORA si se envía. El proyecto siempre nace "draft".
}

interface UpdateProjectRequest {
  name?: string;                // los tres son opcionales: solo se actualiza lo enviado
  description?: string;
  status?: ProjectStatus;
}

interface AddMemberRequest    { user_id: string; role: ProjectMemberRole }
interface InviteMemberRequest { email: string; role: "admin" | "member"; expires_at: string }

// ── Responses ──────────────────────────────────────────────────────────────
interface ProjectResponse {
  id: string;
  owner_id: string;
  name: string;
  description: string;
  status: ProjectStatus;
  created_at: string;
  updated_at: string;
  // Sin archived_at ni deleted_at: no se exponen.
}

interface MemberUserResponse {
  id: string; email: string; name: string;
  role: UserRole;               // rol GLOBAL del usuario, no el del proyecto
  created_at: string; updated_at: string;
}

interface ProjectMemberResponse {
  id: string;
  project_id: string;
  user: MemberUserResponse;     // ⚠️ anidado, no plano
  role: ProjectMemberRole;      // rol DENTRO del proyecto ← el que importa
  status: string;               // "active"
  created_at: string;
  updated_at: string;
}

interface ProjectInvitationResponse {
  id: string; project_id: string; email: string;
  role: ProjectMemberRole; status: string;
  expires_at: string; created_at: string;
}

interface ArchiveResponse { status: "archived" }
```

⚠️ **Hay dos campos `role` en `ProjectMemberResponse`** y significan cosas
distintas: `member.role` es el rol en el proyecto (`owner`/`admin`/`member`), y
`member.user.role` es el rol global del usuario (`estudiante`/`moderador`). El
que gobierna los permisos es **el primero**.

## Endpoints

| Método | Ruta | Rol mínimo | Respuesta |
|---|---|---|---|
| `POST` | `/projects` | autenticado | `201 ProjectResponse` |
| `GET` | `/projects?status=` | autenticado | `200 ProjectResponse[]` |
| `GET` | `/projects/:id` | miembro | `200 ProjectResponse` |
| `PATCH` | `/projects/:id` | owner/admin | `200 ProjectResponse` |
| `DELETE` | `/projects/:id` | owner | `204` sin cuerpo |
| `POST` | `/projects/:id/archive` | owner/admin | `200 ArchiveResponse` |
| `POST` | `/projects/:id/members` | owner/admin | `201 ProjectMemberResponse` |
| `POST` | `/projects/:id/members/invite` | owner/admin | `201 ProjectInvitationResponse` |
| `GET` | `/projects/:id/members` | miembro | `200 ProjectMemberResponse[]` |

```ts
export const projectsApi = {
  create: (body: CreateProjectRequest) =>
    api.post<ProjectResponse>("/projects", body),

  list: (status?: ProjectStatus) =>
    api.get<ProjectResponse[]>(`/projects${status ? `?status=${status}` : ""}`),

  get: (id: string) => api.get<ProjectResponse>(`/projects/${id}`),

  update: (id: string, body: UpdateProjectRequest) =>
    api.patch<ProjectResponse>(`/projects/${id}`, body),

  archive: (id: string) => api.post<ArchiveResponse>(`/projects/${id}/archive`),
  remove:  (id: string) => api.del<void>(`/projects/${id}`),

  members: (id: string) =>
    api.get<ProjectMemberResponse[]>(`/projects/${id}/members`),

  addMember: (id: string, body: AddMemberRequest) =>
    api.post<ProjectMemberResponse>(`/projects/${id}/members`, body),

  invite: (id: string, body: InviteMemberRequest) =>
    api.post<ProjectInvitationResponse>(`/projects/${id}/members/invite`, body),
};
```

## Flujo de creación — son dos pasos, no uno

```ts
// ⚠️ El status del body se descarta: el backend fija "draft" sin leerlo.
const project = await projectsApi.create({ name, description });
console.log(project.status);  // → "draft", aunque se haya mandado "active"

// Para dejarlo activo hace falta un PATCH aparte:
const active = await projectsApi.update(project.id, { status: "active" });
```

Si la UI tiene un formulario de "crear proyecto activo", debe hacer las dos
llamadas y tratarlas como una sola operación de cara al usuario (G-14).

El creador queda automáticamente como miembro `owner`: no hace falta agregarlo.

## Permisos de la interfaz

`GET /projects/:id/members` es **la única fuente de permisos de la API**.
Conviene pedirlo al entrar al detalle del proyecto y derivar las capacidades:

```ts
type ProjectPermissions = {
  canView: boolean; canEdit: boolean; canManageTeam: boolean; canDelete: boolean;
};

function permissionsFor(
  members: ProjectMemberResponse[],
  currentUserId: string,
): ProjectPermissions {
  const me = members.find(m => m.user.id === currentUserId);
  const role = me?.role;
  return {
    canView:       Boolean(me),
    canEdit:       role === "owner" || role === "admin",
    canManageTeam: role === "owner" || role === "admin",
    canDelete:     role === "owner",
  };
}
```

Un `GET /projects/:id` sobre un proyecto ajeno devuelve `403`: ese sí es un
control real y se puede usar como guard de ruta.

⚠️ **Ese control existe solo aquí.** Participantes, instrumentos, respuestas y
exports **no** verifican pertenencia (G-02): un `project_id` equivocado en la URL
no va a dar `403`, va a devolver datos de otro proyecto. Conviene validar en el
cliente que el `project_id` de la ruta sea el del proyecto cargado antes de
hacer llamadas a esos módulos.

## Gestión del equipo

### Agregar un miembro (el camino que funciona)

```ts
// 1. Buscar al usuario — ver 02_users.md
const { data: candidates } = await usersApi.available({ q: search });
// 2. Agregarlo con su id
await projectsApi.addMember(projectId, { user_id: candidates[0].id, role: "member" });
```

`role` acepta `owner`, `admin` o `member` en este endpoint, pero asignar `owner`
a un segundo usuario deja el proyecto con dos owners y sin transferencia real:
`projects.owner_id` no cambia. Mejor restringir la UI a `admin` y `member`.

### Invitar por correo (el camino que no se puede completar)

```ts
await projectsApi.invite(projectId, {
  email: "colega@unsaac.edu.pe",
  role: "member",
  expires_at: new Date(Date.now() + 7 * 864e5).toISOString(),  // debe ser futuro
});
```

⚠️ Crea la fila de invitación y **ahí termina**:

- no se envía correo al invitado;
- no hay endpoint para aceptar la invitación;
- no hay endpoint para listar las invitaciones pendientes.

(G-06.) Si se expone el botón "Invitar", el usuario creerá que mandó algo que
nadie va a recibir. Recomendación: no mostrarlo hasta que el backend cierre el
flujo, y usar el buscador + `addMember` mientras tanto.

### Quitar un miembro

No hay endpoint. `RemoveMember` está implementado en el backend pero sin ruta.
No se puede ofrecer en la UI.

## Ciclo de vida y estados

```
draft ──> active ──> paused ──> closed ──> archived
```

⚠️ **El backend no valida transiciones**: acepta cualquier salto, incluido
`archived` → `active`. Si el flujo importa, imponerlo en el cliente (G-15).

```ts
const ALLOWED: Record<ProjectStatus, ProjectStatus[]> = {
  draft:    ["active"],
  active:   ["paused", "closed"],
  paused:   ["active", "closed"],
  closed:   ["archived", "active"],
  archived: [],
};
```

`archive` y `delete` son **borrados lógicos**: el proyecto desaparece de los
listados pero no se destruye. `delete` responde **`204 No Content`**, sin cuerpo:
no intentar parsear la respuesta.

Tras archivar o eliminar, invalidar la caché de `/projects`: el objeto
devuelto por `archive` es solo `{ status: "archived" }`, no el proyecto.

## Validación en el cliente

| Campo | Regla | Endpoint |
|---|---|---|
| `name` | requerido, 2-160 | create |
| `description` | opcional, sin límite declarado | create |
| `user_id` | requerido, UUID | addMember |
| `role` | requerido; `owner`\|`admin`\|`member` | addMember |
| `email` | requerido, formato correo | invite |
| `role` | requerido; **solo** `admin`\|`member` | invite |
| `expires_at` | requerido, RFC 3339, **debe ser futuro** | invite |

En `update` los tres campos son opcionales, pero mandar `{}` no falla: devuelve
el proyecto sin cambios.

`expires_at` tiene que ser una fecha futura o devuelve `422`. Enviarla como
`new Date(...).toISOString()`.

## Errores

| Código | Significado | Acción en UI |
|---|---|---|
| `400` | `:id` no es UUID | Error de navegación |
| `403` | No es miembro, o le falta rol | Ocultar la acción; no debería llegar aquí |
| `404` | El proyecto no existe o fue eliminado | Pantalla de no encontrado |
| `422` | `name` fuera de rango, `role` inválido, `expires_at` pasada | Resaltar el campo |

## Trampas

1. **`status` en la creación se ignora**: siempre `draft`. Crear + activar son
   dos llamadas (G-14).
2. **El usuario viene anidado**: `member.user.id`, no `member.user_id`.
3. **Dos campos `role`** con significados distintos; el del proyecto es
   `member.role`.
4. **Invitar por correo no llega a nadie** y no se puede aceptar (G-06).
5. **No se puede quitar a un miembro**: no hay endpoint.
6. **No se puede transferir la propiedad**, aunque el dominio lo prevea.
7. **Las transiciones de estado no se validan**: imponerlas en el cliente.
8. **`archive` devuelve `{status: "archived"}`**, no el proyecto actualizado.
9. **`GET /projects` no pagina** y resuelve por membresía. Sin problema con
   decenas de proyectos; a vigilar si crece.
10. **La verificación de permisos termina en este módulo.** No confiar en que el
    backend proteja los demás recursos del proyecto (G-02).
