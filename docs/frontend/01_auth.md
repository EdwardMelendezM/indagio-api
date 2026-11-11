# 01 · Autenticación

## Pantallas

Registro · verificación de OTP · login · recuperación de contraseña · guard de
sesión.

## Tipos

```ts
// ── Requests ───────────────────────────────────────────────────────────────
interface RegisterRequest       { name: string; email: string; password: string }
interface VerifyOTPRequest      { email: string; code: string }
interface LoginRequest          { email: string; password: string }
interface ForgotPasswordRequest { email: string }
interface ResetPasswordRequest  { email: string; code: string; new_password: string }

// ── Responses ──────────────────────────────────────────────────────────────
interface UserResponse {
  id: string;
  name: string;
  email: string;
  role: UserRole;              // "estudiante" | "moderador"
  blocked: boolean;
  avatar_url: string | null;
  created_at: string;          // RFC 3339
}

interface AuthResponse {
  access_token: string;
  refresh_token: string;       // también se fija como cookie httpOnly
  user: UserResponse;
}

interface MessageResponse  { message: string }
interface RefreshResponse  { access_token: string }   // sin refresh_token
```

## Endpoints

| Método | Ruta | Respuesta |
|---|---|---|
| `POST` | `/auth/register` | `201 MessageResponse` |
| `POST` | `/auth/verify-otp` | `200 AuthResponse` |
| `POST` | `/auth/login` | `200 AuthResponse` |
| `POST` | `/auth/refresh` | `200 RefreshResponse` |
| `POST` | `/auth/logout` | `200 MessageResponse` |
| `POST` | `/auth/forgot-password` | `200 MessageResponse` |
| `POST` | `/auth/reset-password` | `200 MessageResponse` |
| `GET` | `/auth/me` | `200 UserResponse` — requiere Bearer |

```ts
export const authApi = {
  register: (body: RegisterRequest) =>
    api.post<MessageResponse>("/auth/register", body),

  verifyOtp: (body: VerifyOTPRequest) =>
    api.post<AuthResponse>("/auth/verify-otp", body),

  login: (body: LoginRequest) =>
    api.post<AuthResponse>("/auth/login", body),

  logout: () => api.post<MessageResponse>("/auth/logout"),

  me: () => api.get<UserResponse>("/auth/me"),

  forgotPassword: (email: string) =>
    api.post<MessageResponse>("/auth/forgot-password", { email }),

  resetPassword: (body: ResetPasswordRequest) =>
    api.post<MessageResponse>("/auth/reset-password", body),
};
```

## Flujo de registro

```
┌─ Formulario de registro
│  POST /auth/register  → 201 { message: "Verifica tu codigo en tu correo" }
│  ⚠️ NO devuelve tokens. El usuario todavía no está autenticado.
│
├─ Pantalla de OTP (guardar el email en el estado de la navegación)
│  POST /auth/verify-otp { email, code }  → 200 AuthResponse
│  Ahora sí hay tokens: guardar el access, la cookie ya quedó puesta.
│
└─ Entrar a la app
```

El correo llega por un job en segundo plano que se despacha cada 3 segundos: un
OTP puede tardar unos segundos en llegar. Conviene un botón de reenvío con
cooldown — aunque **no hay endpoint de reenvío para usuarios normales**
(`/auth/request-otp` solo existe del lado admin). Hoy el único reenvío posible
es repetir el registro, que falla con `409` porque el correo ya existe. Vale la
pena planificarlo con backend.

## Flujo de login

```ts
async function login(email: string, password: string) {
  try {
    const auth = await authApi.login({ email, password });
    setAccessToken(auth.access_token);
    return { ok: true, user: auth.user } as const;
  } catch (e) {
    if (e instanceof ApiError) {
      // 403 en login = usuario no verificado → mandarlo a la pantalla de OTP
      if (e.status === 403) return { ok: false, reason: "unverified", email } as const;
      if (e.status === 401) return { ok: false, reason: "bad_credentials" } as const;
    }
    throw e;
  }
}
```

⚠️ **El `403` de login no es "sin permiso", es "cuenta sin verificar".** Si se
trata como un error genérico, el usuario queda encerrado sin saber que le falta
confirmar el correo.

Nota de seguridad: usuario inexistente y contraseña incorrecta devuelven **ambos
`401`**, a propósito. No se debe inferir ni mostrar "este correo no está
registrado".

## Restaurar la sesión al cargar la app

El access token no está en una cookie, así que si se guarda solo en memoria se
pierde al recargar. Dos opciones:

```ts
// Opción A — pedir uno nuevo con la cookie de refresh (más seguro)
async function bootstrapSession() {
  const res = await fetch(`${API}/auth/refresh`, { method: "POST", credentials: "include" });
  if (!res.ok) return null;                      // sin sesión → login
  const { access_token } = await res.json();
  setAccessToken(access_token);
  return authApi.me();                           // hidratar el usuario
}

// Opción B — persistir el access token en localStorage
// Más simple y sobrevive recargas, pero queda expuesto a XSS y dura 3 días.
```

La opción A es preferible, **pero depende de que la cookie llegue**: hoy, con
`Secure=false` y `SameSite=Lax`, no se envía si el frontend y la API están en
dominios distintos en producción. Ver
[00_CONVENTIONS.md](00_CONVENTIONS.md#la-cookie-de-refresh--leer-con-atención).
Mientras eso no se corrija, un despliegue cross-domain obliga a la opción B.

## Recuperación de contraseña

```
POST /auth/forgot-password { email }
  → 200 { message: "Si el correo existe, se enviara un codigo de recuperacion" }
```

Respuesta deliberadamente ambigua: **no revela si el correo existe**. La UI debe
mostrar ese mismo mensaje sin prometer que llegó un correo.

```
POST /auth/reset-password { email, code, new_password }
  → 200 { message: "Contrasena actualizada correctamente" }
```

El `code` es el mismo mecanismo de OTP de 6 dígitos. No emite tokens: al
terminar hay que mandar al usuario a login.

## Validación en el cliente

Replicar estas reglas, porque el `422` no dice qué campo falló:

| Campo | Regla | Dónde |
|---|---|---|
| `name` | requerido, 2-80 caracteres | register |
| `email` | requerido, formato correo | todos |
| `password` | requerido, **mínimo 8** | register |
| `code` | requerido, **exactamente 6** caracteres | verify-otp, reset-password |
| `new_password` | requerido, mínimo 8 | reset-password |

No hay requisito de complejidad de contraseña más allá del largo: si se quiere
exigir mayúsculas o símbolos, es decisión del cliente.

El `code` se valida con `len=6` sobre la cadena: un input que recorte espacios
o acepte 5 dígitos produce `422` genérico. Conviene un input de 6 casillas.

## Errores

| Código | Endpoint | Significado | Acción en UI |
|---|---|---|---|
| `409` | register | Correo ya registrado | "Ya existe una cuenta con este correo" + enlace a login |
| `401` | login | Credenciales incorrectas | Error genérico, sin detallar |
| `403` | login | **Cuenta sin verificar** | Ir a la pantalla de OTP |
| `422` | verify-otp | Código inválido o vencido | "Código incorrecto o expirado" + reenviar |
| `401` | refresh | Cookie ausente o vencida | Limpiar estado y ir a login |
| `404` | forgot-password | Correo inexistente | **No mostrar**: mantener el mensaje ambiguo |

## Trampas

1. **`register` no autentica.** Devuelve `201` con un mensaje, no tokens. Si la
   UI asume que ya hay sesión, se rompe.
2. **`refresh` devuelve solo `access_token`** y no renueva la cookie. A los 7
   días del login, sesión terminada sin excepción.
3. **`logout` solo borra la cookie.** El access token sigue válido hasta 3 días:
   hay que descartarlo también del lado del cliente. No hay revocación en el
   servidor.
4. **El refresh necesita `credentials: "include"`.** Sin eso devuelve `401`
   aunque el usuario tenga sesión, porque la cookie nunca se manda.
5. **No hay reenvío de OTP** para usuarios normales.
6. **`role` es `estudiante` | `moderador`** — vocabulario heredado de otro
   producto, sin relación con el dominio de investigación, y **no determina
   permisos**. Los permisos reales salen de `GET /projects/:id/members` (G-12).
7. **El access token dura 3 días**, no minutos: no hace falta refrescar de forma
   preventiva en cada navegación.
