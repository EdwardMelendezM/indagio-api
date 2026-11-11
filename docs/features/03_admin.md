# 03 · Administración de plataforma

## Propósito

Acceso de operadores de plataforma, separado del de investigadores: login
**solo por OTP**, sin contraseña, contra una tabla propia. Permite bloquear
usuarios. Módulo auxiliar, fuera del flujo de recolección.

## Entidades y tablas

### `user_adm`

| Columna | Tipo | Nota |
|---|---|---|
| `id` | `UUID` PK | |
| `email` | `TEXT` UNIQUE | |
| `verified` | `BOOLEAN` | |
| `deleted_at` | `TIMESTAMPTZ` NULL | |

Sin columna de contraseña: la autenticación es siempre por OTP. Los códigos se
comparten con la tabla `otps` de [01_auth](01_auth.md).

## Endpoints

| Método | Ruta | Auth | Descripción |
|---|---|---|---|
| `POST` | `/api/admin/auth/login` | — | Envía OTP al correo si existe en `user_adm` |
| `POST` | `/api/admin/auth/verify-otp` | — | Consume el OTP y devuelve tokens |
| `POST` | `/api/admin/auth/request-otp` | Admin | Reenvía OTP a un admin ya autenticado |
| `GET` | `/api/admin/me` | Admin | Perfil del admin |

`admin_handler.go:158-173`.

## Reglas de negocio

- `Login` (`admin_user usecase:91`) no recibe contraseña: verifica que el correo
  exista en `user_adm` y dispara el OTP. El correo es la única credencial de
  primer factor.
- `VerifyOTP` (`:114`) consume el código y emite el par de tokens.
- `BlockUser` (`:150`) marca `users.blocked = true`. Está implementado en el
  usecase pero **no ruteado**: no hay `POST /admin/users/:id/block`.

## Autorización

| | |
|---|---|
| **Implementada** | `AdminAuthMiddleware(tokenSvc, adminSvc)` — valida el JWT y luego `adminSvc.Verify(id)` comprueba contra `user_adm` que el sujeto siga siendo admin. Doble verificación: un token válido de usuario normal no sirve. |
| **Esperada** | Correcta para lo expuesto. |

Es el modelo a seguir para [G-02](00_GAPS.md#g-02--sin-verificación-de-pertenencia-al-proyecto-fuera-del-módulo-projects):
el middleware resuelve la autorización consultando la base, no confiando en los
claims del token.

## Dependencias

- **Requiere:** [01_auth](01_auth.md) (`TokenService`, tabla `otps`),
  [13_background_jobs](13_background_jobs.md) (envío del OTP).
- **Lo consume:** nada del flujo de proyecto.

## Gaps conocidos

- `BlockUser` implementado y sin ruta.
- No hay alta de admins por API: las filas de `user_adm` se insertan a mano.
- `docs/1_MAIN_FEATURES.md` §11 pide panel global con auditoría (quién creó o
  modificó qué, quién accedió a qué entrevista). No existe tabla de auditoría ni
  endpoints de listado. En un contexto con comité de ética esto es un faltante
  relevante.

## Requests de ejemplo

[`requests/admin_auth.http`](../../requests/admin_auth.http)
