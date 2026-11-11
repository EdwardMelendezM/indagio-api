# 00 · Convenciones transversales

Reglas que aplican a todos los endpoints. Leer antes que cualquier doc de feature.

## Base URL

| Entorno | API | Frontend permitido por CORS |
|---|---|---|
| Local | `http://localhost:8080/api` | `http://localhost:5173` |
| Producción | según despliegue | `https://indagio.unsaac.com` |

El prefijo `/api` es parte de la base; los docs de feature citan rutas
relativas a él. Fuera del prefijo solo hay dos rutas:

```
GET /healthz                  → { "status": "ok" }
GET /swagger/index.html       → Swagger UI
```

### CORS

`middleware/cors.go`:

| Config | Valor |
|---|---|
| `AllowOrigins` | `http://localhost:5173`, `https://indagio.unsaac.com` |
| `AllowMethods` | `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS` |
| `AllowHeaders` | **solo** `Authorization` y `Content-Type` |
| `AllowCredentials` | `true` |
| `MaxAge` | 12 h |

Dos consecuencias prácticas:

- Un origen nuevo (preview de Vercel, `127.0.0.1` en vez de `localhost`, otro
  puerto de Vite) **falla** hasta agregarlo a la lista. Es código, no
  configuración: requiere un deploy.
- No se puede mandar ningún header propio además de esos dos. En particular
  `X-Actor-ID`, que el endpoint de sync lee, **es inalcanzable desde el
  navegador** — ver [11_offline_sync.md](11_offline_sync.md).

## Autenticación

Casi todo pide `Authorization: Bearer <access_token>`.

### Rutas públicas

```
POST /auth/register        POST /auth/login          POST /auth/verify-otp
POST /auth/refresh         POST /auth/logout         POST /auth/forgot-password
POST /auth/reset-password  GET  /users/:id
POST /admin/auth/login     POST /admin/auth/verify-otp
POST /sync/batches         GET  /sync/batches        ← ⚠️ sin auth, ver G-03
```

Todo lo demás requiere el header. Sin él: `401 { "error": "Token requerido" }`.
Con formato incorrecto: `401 { "error": "Formato de token inválido" }`.
Expirado o inválido: `401 { "error": "Token inválido o expirado" }`.

### Vigencia de los tokens

| Token | Vigencia | Dónde se guarda |
|---|---|---|
| `access_token` | **3 días** | el cliente decide (memoria, `localStorage`) |
| `refresh_token` | **7 días** | cookie `httpOnly` + también en el body |

Ambas están fijas en el código (`internal/utils/token_service.go:31` y `:47`).
La variable de entorno `JWT_EXPIRATION_HOURS` existe en la configuración pero
**no la usa nadie**: cambiarla no tiene efecto.

Tres días de access token es mucho: no hace falta refrescar de forma preventiva,
pero sí manejar el `401` cuando el usuario vuelve después de un fin de semana
largo.

### La cookie de refresh — leer con atención

`login` y `verify-otp` hacen dos cosas con el refresh token: lo devuelven en el
JSON **y** lo fijan como cookie.

```go
c.SetCookie("refresh_token", refresh, 7*24*3600, "/api/auth", "", false, true)
//                                               ^^^^^^^^^^^  ^^^^^  ^^^^  ^^^^
//                                               Path         Domain Secure HttpOnly
```

De ahí salen cuatro consecuencias:

1. **`Path=/api/auth`** — el navegador solo manda la cookie a rutas bajo ese
   prefijo. Es intencional y correcto: no viaja en cada request de datos.
2. **`HttpOnly`** — el JavaScript no la puede leer. Correcto.
3. **`Secure=false`** y `SameSite` por defecto (`Lax`) — ⚠️ **en producción, si
   el frontend y la API están en dominios distintos, el navegador no enviará la
   cookie.** Para cross-site hace falta `SameSite=None; Secure=true`. Mientras
   no se corrija, el refresh por cookie solo funciona en local o con API y
   frontend bajo el mismo sitio.
4. `POST /auth/refresh` **solo lee la cookie**, nunca el body. Un cliente que no
   sea navegador (móvil nativo, script, Postman sin gestión de cookies) no puede
   refrescar aunque tenga el refresh token del login.

Por eso el login lo devuelve también en el body: para que un cliente nativo lo
guarde. Pero no hay forma de usarlo contra el endpoint. Es un faltante real a
tener en cuenta al planificar la app móvil.

```ts
// El refresh necesita credenciales incluidas, siempre.
await fetch(`${API}/auth/refresh`, { method: "POST", credentials: "include" });
```

Si se usa axios: `withCredentials: true`. Conviene ponerlo global, porque
`login`, `verify-otp` y `logout` también dependen de que la cookie se
establezca o se borre.

### `refresh` no rota el refresh token

```
POST /auth/refresh  →  200 { "access_token": "..." }
```

Devuelve **solo** el access token, y no renueva la cookie. A los 7 días del
login, el usuario vuelve a loguearse pase lo que pase. No hay rotación ni
revocación: `logout` solo borra la cookie, así que un access token ya emitido
sigue siendo válido hasta que expire.

### Cliente HTTP mínimo

```ts
const API = import.meta.env.VITE_API_URL ?? "http://localhost:8080/api";

export class ApiError extends Error {
  constructor(readonly status: number, message: string, readonly detail?: string) {
    super(message);
  }
}

let accessToken: string | null = null;
export const setAccessToken = (t: string | null) => { accessToken = t; };

async function request<T>(path: string, init: RequestInit = {}, retry = true): Promise<T> {
  const res = await fetch(`${API}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      ...(init.body ? { "Content-Type": "application/json" } : {}),
      ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
      ...init.headers,
    },
  });

  // Un 401 puede ser token vencido: intentar refresh una sola vez.
  if (res.status === 401 && retry && accessToken) {
    const refreshed = await fetch(`${API}/auth/refresh`, {
      method: "POST", credentials: "include",
    });
    if (refreshed.ok) {
      const { access_token } = await refreshed.json();
      setAccessToken(access_token);
      return request<T>(path, init, false);
    }
    setAccessToken(null);          // sesión terminada: redirigir a login
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new ApiError(res.status, body.error ?? "error desconocido", body.detail);
  }

  // 204 y respuestas sin cuerpo
  return res.status === 204 ? (undefined as T) : res.json();
}

export const api = {
  get:   <T>(p: string) => request<T>(p),
  post:  <T>(p: string, body?: unknown) =>
           request<T>(p, { method: "POST", body: body ? JSON.stringify(body) : undefined }),
  patch: <T>(p: string, body?: unknown) =>
           request<T>(p, { method: "PATCH", body: body ? JSON.stringify(body) : undefined }),
  del:   <T>(p: string) => request<T>(p, { method: "DELETE" }),
};
```

## Forma de los errores

Siempre un objeto con `error`. Los errores de cliente (`< 500`) pueden traer
además `detail`:

```ts
interface ApiErrorBody {
  error: string;     // mensaje corto; a veces en español, a veces en inglés
  detail?: string;   // ⚠️ el contrato lo contempla, pero NUNCA se envía: ver abajo
}
```

| Código | Cuándo | Qué mostrar |
|---|---|---|
| `400` | UUID malformado en la ruta o query | Error de navegación; no culpar al usuario |
| `401` | Falta el token, es inválido o expiró | Intentar refresh; si falla, al login |
| `403` | Sin permiso, o usuario no verificado | Mensaje explícito según el caso |
| `404` | El recurso no existe | Pantalla de "no encontrado" |
| `409` | Conflicto — **solo en 3 endpoints**, ver abajo | Mensaje accionable |
| `422` | Falló la validación del body | Resaltar los campos del formulario |
| `500` | Error interno | Mensaje genérico + reintento |

⚠️ **Los mensajes de error no son estables ni traducibles.** Mezclan idiomas
(`"Token requerido"`, `"validation failed"`, `"credenciales invalidas"`) y a
veces son genéricos (`"invalid request"` para cualquier fallo de binding, sin
decir qué campo). No se debe parsear el texto: usar el código HTTP para decidir,
y mapear a copys propios en el cliente.

### El campo `detail` nunca llega

El manejador de errores lo contempla, pero **ningún punto del backend lo
rellena**: `AppError.Detail` está vacío en todos los casos. En la práctica, la
respuesta de error es siempre un único campo:

```json
{ "error": "validation failed" }
```

Peor aún, **el contexto que el backend construye internamente se descarta**. El
usecase genera mensajes precisos como
`item "phq8_1" references unknown subscale "x"`, pero al envolverlos sobre un
error centinela, el manejador solo devuelve el mensaje del centinela
(`"validation failed"`). Toda esa información queda en los logs del servidor y
**no llega al cliente**.

Consecuencia: **no se puede construir ningún mensaje de error específico a partir
de la respuesta de la API.** Toda la precisión tiene que venir de la validación
en el cliente, antes de enviar. Los docs de feature listan las reglas a replicar
exactamente por esto.

### `409` existe solo en tres lugares

| Endpoint | Causa |
|---|---|
| `POST /auth/register` | El correo ya está registrado |
| `POST /projects/:id/members` | El usuario ya es miembro del proyecto |
| `PATCH /administrations/:id/complete` | La aplicación está `abandoned` |

⚠️ **Las violaciones de unicidad de la base de datos NO se mapean a `409`.** El
repositorio devuelve el error de constraint envuelto sin traducir, y sale como
**`500`**. Afecta a:

- `external_identifier` repetido al crear un participante;
- `client_generated_id` repetido al iniciar una aplicación;
- `client_generated_batch_id` repetido al subir un lote de sync;
- `question_key` duplicado al publicar un catálogo.

Son casos **esperables** en uso normal —sobre todo al reintentar tras un corte de
red— y el cliente los recibe como un error de servidor indistinguible de una
caída real. Dos implicancias:

1. **No tratar `500` como "algo se rompió" sin más.** En operaciones con
   identificador generado por el cliente, un `500` probablemente significa "ya
   estaba hecho".
2. **Prevenir los duplicados en el cliente**, comprobando contra los datos ya
   cargados antes de enviar. Es la única forma de dar un mensaje útil.

### `422` no dice qué campo falló

El backend responde `422 { "error": "invalid request" }` ante cualquier fallo de
binding. No hay errores por campo. Por eso **la validación del formulario tiene
que vivir en el cliente**: cada doc de feature lista las reglas de `binding` a
replicar, para que el usuario vea el error en el campo correcto antes de enviar.

### `403` tiene dos significados en login

En `POST /auth/login`, un `403` significa "usuario no verificado" y debe llevar
a la pantalla de OTP, no a un "sin permiso". En cualquier otro endpoint es falta
de permiso.

## Convenciones de datos

### Identificadores y fechas

- Todos los ids son **UUID v4 en string**. Nunca numéricos.
- Las fechas son **RFC 3339 / ISO 8601 en UTC**: `"2026-11-10T14:23:01Z"`.
  Directamente consumibles con `new Date(...)`.
- Los campos nullable llegan como `null` explícito, salvo los marcados
  `omitempty` en Go, que **desaparecen del JSON**. En TypeScript eso es
  `campo?: T` (opcional), no `campo: T | null`. Cada doc lo marca.

### Listados: array pelado, sin paginar

Todos los listados devuelven un **array JSON directo**, no un objeto envolvente:

```ts
const projects = await api.get<ProjectResponse[]>("/projects");  // [...]
```

Cuando están vacíos devuelven `[]`, no `null` — los handlers inicializan el
slice con `make(..., 0, n)`. Se puede iterar sin guardas.

**Dos excepciones:**

1. **`GET /users/available`** es el único endpoint paginado, y sí viene
   envuelto:
   ```ts
   interface ListAvailableUsersResponse {
     data: UserResponse[]; total: number; page: number; limit: number;
   }
   ```
2. **Los arrays internos de `CatalogResponse`** (`items`, `subscales`, `rules`,
   `bands`) se construyen con `append` sobre un slice nil, así que **llegan como
   `null` cuando están vacíos**. Un PHQ-8 publicado con `subscales: []` se lee
   de vuelta como `"subscales": null`. Normalizar al recibir:
   ```ts
   const subscales = catalog.subscales ?? [];
   ```

Ningún otro listado pagina. `GET /projects/:id/participants` y
`GET /projects/:id/participants/:pid/answers` traen **todo** de una vez: hay que
paginar y filtrar en el cliente, y asumir respuestas grandes en proyectos con
muchos sujetos. Es el límite más probable de encontrar en producción.

### Enums

Siempre strings. Conviene tiparlos como uniones literales, no como `string`:

```ts
export type ProjectStatus      = "draft" | "active" | "paused" | "closed" | "archived";
export type ProjectMemberRole  = "owner" | "admin" | "member";
export type ParticipantStatus  = "pending" | "active" | "completed" | "expired";
export type InstrumentKind     = "text" | "numeric" | "scale"
                               | "single_choice" | "multi_choice" | "audio" | "video";
export type InstrumentStatus   = "draft" | "active" | "archived";
export type AnswerType         = "single_choice" | "multiple_choice" | "text" | "scale";
export type AdministrationStatus = "in_progress" | "completed" | "abandoned";
export type ScoringAlgorithm   = "sum" | "average" | "weighted_sum" | "lookup_table" | "custom";
export type MissingStrategy    = "prorate" | "zero" | "null_if_missing";
export type ExportFormat       = "csv" | "json" | "zip";
export type ExportStatus       = "pending" | "processing" | "completed" | "failed";
export type SyncBatchStatus    = "pending" | "processed" | "failed";
export type UserRole           = "estudiante" | "moderador";
```

⚠️ `AnswerType` usa `multiple_choice`, mientras `InstrumentKind` usa
`multi_choice`. No son intercambiables: es fácil mandar el valor equivocado y
recibir un `422`.

El backend valida que el valor pertenezca al enum, pero **no valida
transiciones**: puede llevar un proyecto `archived` de vuelta a `active` o un
participante `completed` a `pending`. Si el flujo de trabajo importa, el cliente
tiene que imponerlo (G-15).

### Timeouts del servidor

Cada handler envuelve su usecase en un `context` de **5 segundos**, y el
servidor usa `ReadTimeout` 10 s / `WriteTimeout` 30 s. Un timeout del lado del
servidor sale como `500`. Conviene un timeout de cliente algo mayor (15-20 s)
para distinguir "el servidor tardó" de "la red se cayó".

## Autorización: lo que el cliente no puede dar por hecho

⚠️ **Solo el módulo de proyectos verifica la pertenencia al proyecto.**
Participantes, instrumentos, catálogos, aplicaciones, respuestas y exports
aceptan cualquier `project_id` de cualquier usuario autenticado (G-02).

Para el frontend significa dos cosas:

- **No esperar `403` como mecanismo de protección.** Un bug de navegación que
  mande el `project_id` equivocado no va a fallar: va a devolver —o peor,
  modificar— datos de otro proyecto. Conviene validar en el cliente que el
  `project_id` de la ruta coincida con el proyecto activo.
- **El único permiso que el backend sí informa es el de proyecto.**
  `GET /projects/:id/members` devuelve el `role` de cada miembro: de ahí sale
  qué puede hacer el usuario actual en la UI (ocultar "Eliminar proyecto" si no
  es `owner`, etc.). Es la fuente de verdad para los permisos de la interfaz.

```ts
// Permisos de UI derivados de la membresía
const me = members.find(m => m.user.id === currentUserId);
const canEdit   = me?.role === "owner" || me?.role === "admin";
const canDelete = me?.role === "owner";
```

## Convenciones de cada doc de feature

1. **Pantallas** — qué vistas cubre.
2. **Tipos** — interfaces TypeScript, derivadas de los DTOs de Go.
3. **Endpoints** — método, ruta y función cliente de ejemplo.
4. **Flujo** — orden de llamadas y qué guardar entre ellas.
5. **Validación en el cliente** — las reglas de `binding` a replicar.
6. **Errores** — qué códigos devuelve y qué mostrar.
7. **Trampas** — lo que cuesta tiempo descubrir.
