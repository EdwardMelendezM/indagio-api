# 01 · Autenticación y OTP

## Propósito

Alta y acceso de los investigadores que operan la plataforma. Registro con
verificación por OTP de 6 dígitos, login con password, tokens JWT de acceso y
refresco, y recuperación de contraseña. Es el paso 1-3 del
[flujo](../2_PROJECT_FLOW.md).

## Entidades y tablas

### `users`

| Columna | Tipo | Nota |
|---|---|---|
| `id` | `UUID` PK | |
| `name` | `VARCHAR(100)` | |
| `email` | `TEXT` UNIQUE | |
| `password` | `VARCHAR(255)` | hash, nunca el valor plano |
| `role` | `ENUM user_role` | `estudiante` \| `moderador` — ver G-12 |
| `avatar_url` | `TEXT` NULL | JSON con variantes `thumb`/`medium`/`full` |
| `avatar_version` | `INTEGER` | contador para cache-busting |
| `verified` | `BOOLEAN` | `false` hasta consumir el OTP |
| `blocked` | `BOOLEAN` | bloqueo administrativo |
| `deleted_at` | `TIMESTAMPTZ` NULL | borrado lógico |

### `otps`

| Columna | Tipo | Nota |
|---|---|---|
| `email` | `TEXT` | no hay FK a `users` |
| `code` | `VARCHAR(20)` | 6 dígitos en la práctica |
| `expires_at` | `TIMESTAMPTZ` | |
| `used` | `BOOLEAN` | consumo de un solo uso |

Índice `idx_otps_email_active (email, used, expires_at DESC)`: la validación
siempre busca el OTP activo más reciente de ese correo.

## Endpoints

| Método | Ruta | Auth | Descripción |
|---|---|---|---|
| `POST` | `/api/auth/register` | — | Crea el usuario y encola el OTP |
| `POST` | `/api/auth/verify-otp` | — | Consume el OTP, marca `verified`, devuelve tokens |
| `POST` | `/api/auth/login` | — | Password → tokens |
| `POST` | `/api/auth/refresh` | — | Refresh token → nuevo par de tokens |
| `POST` | `/api/auth/logout` | — | |
| `POST` | `/api/auth/forgot-password` | — | Encola `send_password_reset` |
| `POST` | `/api/auth/reset-password` | — | Código + nueva contraseña |
| `GET` | `/api/auth/me` | Bearer | Perfil del token |

Registradas en `auth_handler.go:309-318`. `/me` es la única con
`authMiddleware` aplicado a nivel de ruta.

## Contratos

```go
RegisterRequest      { name: min=2,max=80 · email · password: min=8 }
VerifyOTPRequest     { email · code: len=6 }
LoginRequest         { email · password }
ForgotPasswordRequest{ email }
ResetPasswordRequest { email · code: len=6 · new_password: min=8 }
```

```go
AuthResponse {
  access_token, refresh_token string
  user { id, name, email, role, blocked, avatar_url, created_at }
}
```

El hash de contraseña nunca sale: `internalToPublic` (`auth_usecase.go:174`)
convierte `UserInternal` → `User` descartando el campo.

## Reglas de negocio

- **Registro** (`auth_usecase.go:58`): valida largo de nombre y contraseña,
  verifica unicidad del correo contra `FindByEmail`, hashea, crea, y llama
  `SendOTP`. Si `FindByEmail` encuentra algo → `ErrUserAlreadyExists` (409).
- **Encolado de OTP** (`auth_usecase.go:43`): el OTP se guarda en Postgres y el
  envío se delega a `background_jobs`. Si el `Enqueue` falla, **se registra por
  `fmt.Printf` y el registro se considera exitoso** — el usuario queda creado sin
  recibir correo. Es deliberado (no bloquear el registro por el correo) pero
  debería usar el `slog` del proceso y no `stdout` crudo.
- **Verificación** (`auth_usecase.go:96`): `ValidateAndConsume` es atómico —
  valida vigencia y marca `used` en una operación, así el mismo código no sirve
  dos veces.
- **Login** (`auth_usecase.go:117`): usuario inexistente y contraseña incorrecta
  devuelven ambos `ErrInvalidCredentials` (401), sin distinguir cuál falló.
  Un usuario no verificado recibe `403 ErrForbidden`.

## Autorización

| | |
|---|---|
| **Implementada** | `AuthMiddleware(tokenSvc)` valida el access token y deja `userID` (`uuid.UUID`) y el rol en el contexto de Gin. Todos los handlers lo leen con `c.Get("userID")`. |
| **Esperada** | Lo mismo. Este módulo es el único cuya autorización está completa. |

El rol viaja en los claims del JWT pero **no autoriza nada** en el flujo de
investigación: el acceso real lo determina la membresía en `project_members`.

## Dependencias

- **Requiere:** [13_background_jobs](13_background_jobs.md) para el envío de
  correos; `utils.EmailService`, `PasswordService`, `TokenService`.
- **Lo consume:** todos los demás módulos — el `userID` del token es el
  `actorID` de cada usecase.

## Gaps conocidos

- **[G-12](00_GAPS.md#g-12--los-roles-de-usuario-no-son-los-del-dominio)** —
  los roles `estudiante`/`moderador` vienen de otro producto y no se parecen a
  los del dominio de investigación. No existe acceso de participante
  (entrevistado): no hay token de un solo uso ni portal propio, todo pasa por un
  investigador autenticado.
- El fallo de `Enqueue` en `SendOTP` se traga con `fmt.Printf`
  (`auth_usecase.go:53`), no con el logger estructurado.
- `otps.email` no tiene FK ni limpieza: la tabla crece indefinidamente con
  códigos vencidos.

## Requests de ejemplo

[`requests/auth.http`](../../requests/auth.http)
