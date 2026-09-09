# Plan de Backend — Go (Enfoque de Features)

## 0. Decisiones de arquitectura de partida

- **Arquitectura**: monolito modular (no microservicios desde el día uno). Para un proyecto de investigación con equipo probablemente pequeño, microservicios añaden complejidad operativa sin beneficio real todavía. Se estructura el código en módulos internos bien separados (`internal/projects`, `internal/users`, etc.) para poder extraerlos como servicios independientes más adelante si el proyecto escala — esto es lo que da la escalabilidad sin pagar el costo ahora.
- **Framework HTTP**: Go con `chi` o `Fiber` (routing ligero, sin imponer demasiada estructura).
- **Base de datos principal**: PostgreSQL (relacional, encaja bien con proyectos/usuarios/preguntas/respuestas estructuradas, y soporta JSONB para las partes dinámicas como el constructor de preguntas).
- **Almacenamiento de media** (audio/video): no se guarda en la base de datos — se sube a almacenamiento de objetos compatible S3 (AWS S3, Cloudflare R2, o GCS), y en Postgres solo se guarda la referencia/metadata.
- **Autenticación**: JWT con refresh tokens, más un mecanismo de acceso simplificado (magic link/código) para el rol "entrevistado", que no necesita cuenta completa.

---

## 1. Feature: Autenticación y gestión de usuarios

- Registro/login para Owners y Miembros (email + password, o SSO si se requiere más adelante).
- Acceso diferenciado para Entrevistados: generación de un **token de un solo uso o de tiempo limitado** ligado a una sesión de entrevista específica, sin necesidad de crear cuenta.
- Roles del sistema:
  - `platform_admin` (opcional, gestión global)
  - `project_owner`
  - `project_member`
  - `interviewee` (acceso acotado a su propia sesión)
- Control de acceso basado en roles (RBAC) a nivel de middleware, verificando tanto el rol global como la pertenencia al proyecto específico (un miembro solo puede tocar los proyectos donde está asignado).
- Endpoints típicos: `POST /auth/login`, `POST /auth/refresh`, `POST /auth/invite`, `POST /auth/interview-access`.

---

## 2. Feature: Gestión de Proyectos

- CRUD de proyectos, cada uno con:
  - Owner (uno solo, transferible)
  - Lista de miembros con su rol dentro del proyecto
  - Estado (borrador, activo, pausado, cerrado)
  - Metadata general (nombre, descripción, fechas)
- Al crear un proyecto, se define automáticamente el creador como `project_owner`.
- Endpoint para invitar miembros por email, con expiración de invitación.

---

## 3. Feature: Códigos / Participantes

- Cada "código" representa un participante/entrevistado dentro de un proyecto.
- Soporta:
  - Creación individual
  - **Importación masiva** (CSV) para proyectos con muchos participantes
  - Generación de token de acceso único por código, para el portal del entrevistado
  - Estado del código: pendiente, en progreso, completado, expirado
- Relación 1:1 (o 1:N si se permiten múltiples sesiones) entre código y sesión de entrevista.

---

## 4. Feature: Instrumentos (PH8 y otros)

- Modelo de "instrumento" como una entidad configurable, no hardcodeada: nombre, tipo, parámetros (rango, unidad, tolerancia), y si aplica, reglas de validación de la medición.
- Los instrumentos se pueden definir como **plantillas reutilizables** a nivel de cuenta/organización, y luego asociarse a un proyecto específico.
- Dado que la configuración es manual, este módulo es principalmente CRUD + validación de esquema (usando JSONB en Postgres para los parámetros específicos de cada tipo de instrumento, ya que estos varían).
- Diseño pensado para que agregar un nuevo tipo de instrumento en el futuro no requiera cambios de esquema en la base de datos, solo nueva configuración/validación en la capa de aplicación.

---

## 5. Feature: Preguntas Dinámicas

- Modelo de cuestionario como un **grafo/árbol de preguntas** almacenado en JSONB (dada la naturaleza dinámica y variable de la lógica condicional), con una capa de validación en Go que garantiza consistencia (no referencias a preguntas inexistentes, no ciclos, etc.).
- Tipos de pregunta soportados desde el modelo de datos: texto, opción múltiple, escala, audio, video — extensible vía un campo `type` + `config` (JSONB).
- Versionado de cuestionarios: si se edita un cuestionario después de que ya hay respuestas, se debe versionar en vez de sobrescribir, para no corromper datos de investigación ya recolectados. Esto es crítico para la integridad de los datos de investigación.
- Endpoints: `GET/POST /projects/:id/questions`, `POST /projects/:id/questions/publish` (congela una versión).

---

## 6. Feature: Respuestas (Audio / Texto / Video)

- Flujo de subida de media:
  1. El cliente solicita una **URL prefirmada** de subida directa al bucket (no pasa el archivo por el servidor Go, para no sobrecargarlo con archivos pesados).
  2. El cliente sube el archivo directamente al almacenamiento de objetos.
  3. El cliente confirma al backend que la subida terminó, y el backend guarda la referencia + metadata (duración, tamaño, formato, checksum).
- Respuestas de texto se guardan directamente en Postgres.
- Procesamiento asíncrono post-subida (vía cola de trabajos, ver sección 9): generación de thumbnails para video, normalización de audio, y opcionalmente transcripción automática (integración con un servicio de speech-to-text) para facilitar el análisis posterior de investigación.
- Cada respuesta queda asociada a: proyecto, código/participante, pregunta específica, versión del cuestionario, timestamp, y estado de sincronización (importante porque el frontend puede enviar respuestas guardadas offline).

---

## 7. Feature: Soporte Mobile / Sincronización Offline

- Endpoint de **sincronización por lotes**: el cliente mobile puede enviar un conjunto de respuestas generadas offline en una sola request, con manejo de idempotencia (usando un `client_generated_id` por respuesta para evitar duplicados si la sincronización se reintenta).
- Diseño de API tolerante a conexiones intermitentes: respuestas parciales, reintentos, y confirmación explícita de qué se sincronizó y qué falló.

---

## 8. Feature: Almacenamiento en la nube

- Abstracción de storage (`internal/storage`) detrás de una interfaz Go, para no acoplar el código a un proveedor específico (S3, GCS, R2) — facilita cambiar de proveedor o usar uno distinto en testing (ej. MinIO local).
- Políticas de acceso: URLs firmadas con expiración corta, nunca acceso público directo a los archivos de entrevistas (datos sensibles de investigación).
- Consideración de retención/backup: definir política de cuánto tiempo se guardan los archivos y backups periódicos del bucket.

---

## 9. Feature: Procesamiento en segundo plano

- Cola de trabajos (ej. usando `asynq` sobre Redis, o SQS si ya se usa AWS) para tareas que no deben bloquear la respuesta HTTP:
  - Procesamiento de media (thumbnails, normalización)
  - Transcripción automática (opcional)
  - Envío de emails (invitaciones, notificaciones)
  - Generación de exportes grandes (ver siguiente sección)

---

## 10. Feature: Exportación de datos

- Exportar respuestas de un proyecto en formatos usables para análisis de investigación: CSV/Excel para respuestas de texto/opción múltiple, y enlaces/paquete ZIP para media.
- Para proyectos grandes, esto se ejecuta como un job asíncrono (ver sección 9) y se notifica al usuario cuando el export está listo, en vez de bloquear la request.

---

## 11. Administración de usuarios (plataforma)

- Panel de administración global (solo `platform_admin`) para ver todos los usuarios, desactivar cuentas, y auditar actividad básica.
- Logs de auditoría mínimos: quién creó/modificó qué proyecto, quién accedió a qué entrevista — relevante en un contexto de investigación con datos sensibles.

---

## 12. Seguridad y cumplimiento (relevante por ser datos de investigación)

- Cifrado en tránsito (TLS) y en reposo (cifrado del bucket de almacenamiento).
- Consentimiento del entrevistado registrado como un evento auditable, no solo un checkbox de UI.
- Anonimización/seudonimización de participantes: separar el "código" identificable de los datos de respuesta cuando sea posible, para minimizar exposición de datos personales.
- Definir política de retención y borrado de datos (importante si hay comités de ética de investigación involucrados).

---

## 13. Escalabilidad y mantenibilidad

- Estructura de carpetas sugerida:
  ```
  /cmd/api                 → entrypoint
  /internal/auth
  /internal/projects
  /internal/participants
  /internal/instruments
  /internal/questions
  /internal/answers
  /internal/storage
  /internal/jobs
  /internal/platform       → middleware, config, logging, db
  /pkg                     → utilidades compartidas sin lógica de negocio
  ```
- Cada módulo expone su propia interfaz de servicio y no accede directamente a las tablas de otro módulo — esto es lo que permite, si el proyecto crece, extraer por ejemplo el módulo de `answers` (el más pesado en tráfico/media) como servicio independiente sin reescribir todo.
- Migraciones de base de datos versionadas (ej. `golang-migrate`).
- Tests: unitarios por módulo + tests de integración contra una Postgres real (vía Docker) para los flujos críticos (sincronización offline, subida de media, versionado de cuestionarios).
- Observabilidad desde el inicio: logging estructurado, métricas básicas (requests, errores, duración de jobs), dado que en campo los fallos de sincronización son el punto más frágil del sistema.

---

## 14. Roadmap sugerido (MVP → escalable)

1. **MVP**: Auth + Proyectos + Códigos + Preguntas (tipos simples) + Respuestas de texto + Storage básico.
2. **V2**: Subida de audio/video vía URLs prefirmadas, sincronización offline por lotes, versionado de cuestionarios.
3. **V3**: Instrumentos configurables (PH8 y otros), procesamiento asíncrono (transcripción, thumbnails), exportación de datos.
4. **V4**: Auditoría avanzada, panel de admin global, posible extracción de módulos pesados a servicios independientes.

---

## 15. Stack técnico propuesto

- Go + `chi` o `Fiber`
- PostgreSQL + `golang-migrate`
- Redis + `asynq` (jobs en segundo plano)
- S3-compatible storage (AWS S3 / Cloudflare R2 / GCS) con URLs prefirmadas
- JWT (`golang-jwt`) para auth
- `zerolog` o `zap` para logging estructurado
- Docker + Docker Compose para desarrollo local
