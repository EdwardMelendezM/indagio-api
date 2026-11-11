# Integración con el frontend — Indagio API

Guía de consumo de la API desde el cliente. Un documento por feature, con los
tipos TypeScript exactos (derivados de los DTOs de Go), la secuencia de llamadas
por pantalla, las validaciones a replicar en el cliente y las trampas conocidas.

**Empezar por [00_CONVENTIONS.md](00_CONVENTIONS.md)**: base URL, manejo de
tokens, forma de los errores y las convenciones que aplican a todos los
endpoints. Sin eso, el resto no se entiende.

## Índice

| Doc | Cubre | Pantallas típicas |
|---|---|---|
| [00_CONVENTIONS.md](00_CONVENTIONS.md) | Reglas transversales | — |
| [01_auth.md](01_auth.md) | Registro, OTP, login, refresh, recuperación | Login, registro, verificación |
| [02_users.md](02_users.md) | Perfiles y buscador de usuarios | Perfil, selector de miembros |
| [03_admin.md](03_admin.md) | Acceso de plataforma | Login de admin |
| [04_projects.md](04_projects.md) | Proyectos, miembros, invitaciones | Lista y detalle de proyecto, equipo |
| [05_participants.md](05_participants.md) | Participantes y códigos | Padrón, alta de participante |
| [06_instruments.md](06_instruments.md) | Instrumentos | Lista y editor de instrumento |
| [07_scoring_catalog.md](07_scoring_catalog.md) | Catálogo de puntuación | Constructor de ítems y baremos |
| [08_administrations.md](08_administrations.md) | Aplicaciones y puntajes | Runner del instrumento, resultados |
| [09_answers.md](09_answers.md) | Respuestas y media | Formulario de ítems |
| [10_media.md](10_media.md) | Subida a R2 con URL prefirmada | Captura de audio/video |
| [11_offline_sync.md](11_offline_sync.md) | Lotes offline | Cliente móvil / PWA |
| [12_exports.md](12_exports.md) | Exportación | Panel de descargas |

El worker de jobs (`docs/features/13_background_jobs.md`) no tiene superficie
HTTP: no hay nada que integrar desde el cliente.

## Relación con la otra documentación

| Carpeta | Para qué |
|---|---|
| `docs/frontend/` | **cómo consumir** la API desde el cliente — esta carpeta |
| `docs/features/` | **qué hace** el backend por dentro: tablas, reglas, autorización |
| `docs/2_PROJECT_FLOW.md` | el flujo completo de un proyecto nuevo, de punta a punta |
| `docs/features/00_GAPS.md` | defectos y faltantes conocidos, por severidad |

Cuando un doc de esta carpeta dice "ver G-NN", es una entrada de
`docs/features/00_GAPS.md`. Varios gaps afectan directamente al cliente y están
señalados en cada feature como ⚠️.

## Las seis trampas que más tiempo cuestan

Si solo se lee una cosa antes de integrar, que sea esta lista.

1. **El `value` de una respuesta va envuelto en `{ "value": X }`.** Cualquier
   otra clave se acepta con `201` y luego el puntaje sale `null` sin error.
   → [09_answers.md](09_answers.md), G-01.
2. **Los arrays del catálogo llegan como `null`, no `[]`, cuando están vacíos.**
   Un PHQ-8 sin subescalas responde `"subscales": null`.
   → [07_scoring_catalog.md](07_scoring_catalog.md).
3. **El refresh token vive en una cookie `httpOnly` con `Path=/api/auth`.**
   Hace falta `credentials: "include"`, y hoy no funciona entre dominios
   distintos en producción. → [00_CONVENTIONS.md](00_CONVENTIONS.md).
4. **`POST /projects` ignora el `status` del body**: el proyecto siempre nace
   `draft` y hay que activarlo con un `PATCH` aparte.
   → [04_projects.md](04_projects.md), G-14.
5. **Ningún error dice qué campo falló.** Todos los `422` son
   `{"error": "validation failed"}`: el backend descarta el detalle antes de
   responder. Toda la validación útil tiene que estar en el cliente.
   → [00_CONVENTIONS.md](00_CONVENTIONS.md), G-17.
6. **Un reintento tras un corte de red devuelve `500`, no `409`.** Las
   violaciones de unicidad no se traducen, así que "ya estaba hecho" y "el
   servidor se cayó" son indistinguibles sin una llamada extra.
   → [00_CONVENTIONS.md](00_CONVENTIONS.md), G-16.
