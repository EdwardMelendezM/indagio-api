# Documentación por feature — Indagio API

Un documento por módulo del backend, describiendo **lo que el código hace hoy**
(no lo que debería hacer). Cada doc cierra con una sección `Gaps conocidos`
donde se registra lo que falta o está roto, con referencia a `archivo:línea`.

Convención: la prosa está en español; los nombres de tablas, campos, endpoints,
enums y estados se citan textualmente en inglés, igual que en el código.

## Índice

| Doc | Módulo | Paquetes |
|---|---|---|
| [01_auth.md](01_auth.md) | Autenticación y OTP | `usecase/auth`, `delivery/http/auth` |
| [02_users.md](02_users.md) | Perfiles de usuario | `usecase/user`, `delivery/http/user` |
| [03_admin.md](03_admin.md) | Administración de plataforma | `usecase/admin_user`, `delivery/http/admin_user` |
| [04_projects.md](04_projects.md) | Proyectos, miembros e invitaciones | `usecase/projects`, `delivery/http/projects` |
| [05_participants.md](05_participants.md) | Participantes y códigos | `usecase/participants`, `delivery/http/participants` |
| [06_instruments.md](06_instruments.md) | Instrumentos | `usecase/instruments`, `delivery/http/instruments` |
| [07_scoring_catalog.md](07_scoring_catalog.md) | Catálogo de puntuación versionado | `usecase/instrument_scoring` |
| [08_administrations.md](08_administrations.md) | Aplicaciones y cálculo de puntajes | `usecase/instrument_administration`, `domain/scoring.go` |
| [09_answers.md](09_answers.md) | Respuestas y media records | `usecase/answers`, `delivery/http/answers` |
| [10_media_storage.md](10_media_storage.md) | Presign y almacenamiento R2 | `delivery/http/media`, `repository/storage` |
| [11_offline_sync.md](11_offline_sync.md) | Sincronización offline por lotes | `usecase/sync`, `delivery/http/sync` |
| [12_exports.md](12_exports.md) | Exportación de datos | `usecase/exports`, `delivery/http/exports` |
| [13_background_jobs.md](13_background_jobs.md) | Worker de jobs | `delivery/worker`, `repository/jobs` |

## Documentos transversales

- **[../2_PROJECT_FLOW.md](../2_PROJECT_FLOW.md)** — el flujo completo de un
  proyecto nuevo de punta a punta, los 13 pasos en orden. Empezar por aquí.
- **[00_GAPS.md](00_GAPS.md)** — todos los gaps consolidados y ordenados por
  severidad. Es el backlog técnico derivado de esta documentación.
- **[../frontend/](../frontend/)** — cómo consumir la API desde el cliente: tipos
  TypeScript, secuencias por pantalla y trampas de integración.
- **[../1_MAIN_FEATURES.md](../1_MAIN_FEATURES.md)** — plan original de
  arquitectura. Documento de visión, parcialmente desactualizado (ver
  `00_GAPS.md` → G-13).

## Plantilla

Cada doc de feature sigue la misma estructura:

1. **Propósito** — qué resuelve el módulo dentro del flujo de investigación.
2. **Entidades y tablas** — esquema real en Postgres.
3. **Ciclo de vida** — máquina de estados, si el módulo tiene una.
4. **Endpoints** — tabla de rutas tal como están registradas en `main.go`.
5. **Contratos** — request/response DTOs reales.
6. **Reglas de negocio y validaciones** — qué valida el usecase.
7. **Autorización** — *implementada* vs *esperada*. Casi siempre difieren.
8. **Dependencias** — qué features necesita antes y qué features lo consumen.
9. **Gaps conocidos** — defectos y faltantes con `archivo:línea`.
10. **Requests de ejemplo** — enlace al `.http` correspondiente.
