# 03 · Acceso de administración de plataforma

## Pantallas

Login de admin (solo correo + OTP, sin contraseña) · perfil de admin.

Es una **aplicación o sección aparte** de la de investigadores: tabla de
usuarios distinta, tokens distintos y middleware distinto. No se comparte
sesión con `/auth/*`.

## Tipos

```ts
interface AdminLoginRequest     { email: string }
interface VerifyAdminOTPRequest { email: string; code: string }
interface AdminRequestOTPRequest{ email: string }

interface AdminLoginResponse { message: string }

interface UserAdminResponse {
  id: string;
  email: string;
  // Nada más: sin name, sin role, sin avatar.
}

interface AuthAdminResponse {
  access_token: string;
  refresh_token: string;
  user: UserAdminResponse;     // ⚠️ la clave JSON es "user", no "admin"
}
```

⚠️ El campo de Go se llama `Admin` pero su etiqueta JSON es `"user"`. En el
cliente se lee `response.user`, no `response.admin`.

## Endpoints

| Método | Ruta | Auth | Respuesta |
|---|---|---|---|
| `POST` | `/admin/auth/login` | — | `200 AdminLoginResponse` |
| `POST` | `/admin/auth/verify-otp` | — | `200 AuthAdminResponse` |
| `POST` | `/admin/auth/request-otp` | Admin | reenvía el OTP |
| `GET` | `/admin/me` | Admin | `200 UserAdminResponse` |

```ts
export const adminApi = {
  login: (email: string) =>
    api.post<AdminLoginResponse>("/admin/auth/login", { email }),

  verifyOtp: (body: VerifyAdminOTPRequest) =>
    api.post<AuthAdminResponse>("/admin/auth/verify-otp", body),

  requestOtp: (email: string) =>
    api.post<AdminLoginResponse>("/admin/auth/request-otp", { email }),

  me: () => api.get<UserAdminResponse>("/admin/me"),
};
```

## Flujo de login

Sin contraseña: el correo es la única credencial de primer factor.

```
┌─ Formulario: solo correo
│  POST /admin/auth/login { email }
│  → 200 { message: ... }   ← NO devuelve tokens
│
├─ Pantalla de OTP
│  POST /admin/auth/verify-otp { email, code }
│  → 200 AuthAdminResponse  ← ahora sí hay tokens
│
└─ Entrar al panel
```

A diferencia del login de investigador, aquí **el primer paso nunca autentica**:
siempre hay dos pasos. Y `request-otp` requiere estar ya autenticado, así que no
sirve como "reenviar código" en la pantalla de OTP — solo para un admin con
sesión activa que quiere un código nuevo.

Para reenviar durante el login, repetir `POST /admin/auth/login`.

## Tokens: no mezclar con la sesión de investigador

Los tokens de admin se validan con `AdminAuthMiddleware`, que hace **dos**
comprobaciones: el JWT y, además, que el sujeto siga existiendo en la tabla
`user_adm`. Un token válido de usuario normal **no sirve** para rutas de admin,
y viceversa.

Si la app de admin convive con la de investigadores en el mismo origen, hay que
mantener los dos access tokens separados en el cliente:

```ts
// Dos almacenes distintos; no reutilizar el cliente HTTP con un único token.
const adminToken  = { current: null as string | null };
const userToken   = { current: null as string | null };
```

⚠️ **La cookie de refresh se comparte y colisiona.** El login de admin fija la
misma cookie `refresh_token` con el mismo `Path=/api/auth` que el de
investigador. Loguearse como admin **sobrescribe** la cookie de refresh del
investigador y al revés. Si las dos aplicaciones se usan en el mismo navegador,
el refresh de una va a devolver un token de la otra identidad. Separarlas por
origen (subdominio distinto) o no depender del refresh por cookie en el panel de
admin.

## Validación en el cliente

| Campo | Regla |
|---|---|
| `email` | requerido, formato correo |
| `code` | requerido, exactamente 6 caracteres |

## Errores

| Código | Endpoint | Significado | Acción |
|---|---|---|---|
| `404` | login | El correo no está en `user_adm` | Mensaje genérico; no confirmar si existe |
| `422` | verify-otp | Código inválido o vencido | Reintentar o reenviar |
| `401` | `/admin/me` y protegidas | Token ausente, inválido, o ya no es admin | Volver al login de admin |

## Trampas

1. **La clave JSON del admin es `user`**, aunque el campo de Go sea `Admin`.
2. **No hay registro de admins por API.** Las filas de `user_adm` se insertan a
   mano en la base. La UI no puede dar de alta un administrador.
3. **`request-otp` requiere sesión**: no sirve como reenvío en el login.
4. **La cookie de refresh choca con la sesión de investigador** (mismo nombre,
   mismo path).
5. **`UserAdminResponse` solo trae `id` y `email`.** No hay nombre ni rol que
   mostrar en la cabecera del panel.
6. **El panel no tiene casi nada que consumir.** Bloquear usuarios está
   implementado en el backend pero **sin ruta HTTP**, y no existen endpoints de
   listado de usuarios, auditoría ni estadísticas. Hoy un panel de admin solo
   puede mostrar "sesión iniciada como `<email>`". Vale la pena acordar el
   alcance con backend antes de diseñar pantallas. Ver
   `docs/features/03_admin.md` → Gaps.
