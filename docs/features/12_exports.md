# 12 · Exportación de datos

## Propósito

Entregar los datos del proyecto en un formato analizable (CSV, JSON, ZIP con
media) sin bloquear la request HTTP: se solicita, se procesa en segundo plano y
se descarga cuando está listo.

⚠️ **Solo existe la mitad de la solicitud.** La fila se crea y se queda en
`pending` para siempre: no hay nada que la procese. Ver
[G-05](00_GAPS.md#g-05--los-exports-nunca-se-procesan).

## Entidades y tablas

### `project_exports`

| Columna | Tipo | Nota |
|---|---|---|
| `project_id` | `UUID` → `projects` | `ON DELETE CASCADE` |
| `created_by` | `UUID` → `users` | |
| `format` | `VARCHAR(32)` | `csv` \| `json` \| `zip` |
| `status` | `VARCHAR(32)` | `pending` \| `processing` \| `completed` \| `failed` |
| `options` | `JSONB` | esquema no definido |
| `file_key` | `TEXT` NULL | ruta en R2 del resultado; **nunca se llena** |
| `error_log` | `TEXT` NULL | nunca se llena |
| `started_at` / `completed_at` | `TIMESTAMPTZ` NULL | nunca se llenan |

El esquema está completo y bien pensado para un trabajo asíncrono. Lo que falta
es el trabajador.

## Endpoints

| Método | Ruta | Descripción |
|---|---|---|
| `POST` | `/api/projects/:id/exports` | Solicitar un export |
| `GET` | `/api/projects/:id/exports` | Listar los del proyecto |

`export_handler.go:142-145`.

No existe `GET /projects/:id/exports/:exportId`, aunque
`ExportUsecase.GetExport` está implementado: no se puede consultar un export
puntual ni, cuando exista, obtener su enlace de descarga.

## Contratos

```jsonc
// POST
{ "format": "csv", "options": { /* forma libre */ } }
→ 201 { "id", "project_id", "format": "csv", "status": "pending",
        "file_key": null, "created_at", ... }
```

`options` solo se valida como JSON bien formado. Queda por definir qué admite:
rango de fechas, instrumentos a incluir, si se anonimizan los códigos de
participante, si se incluyen puntajes o solo respuestas crudas.

## Reglas de negocio

`RequestExport` (`exports usecase:22`) valida ids no nulos, normaliza `format` a
minúsculas y lo contrasta contra `{csv, json, zip}`, y verifica `options` como
JSON válido. Luego inserta con `status = pending` y responde.

Ahí termina todo el ciclo de vida implementado.

## Autorización

| | |
|---|---|
| **Implementada** | Solo `actorID != uuid.Nil`. |
| **Esperada** | Miembro del proyecto, posiblemente `owner`/`admin`: un export es una copia completa de los datos del estudio. |

Dentro de [G-02](00_GAPS.md#g-02--sin-verificación-de-pertenencia-al-proyecto-fuera-del-módulo-projects),
este es el caso de mayor impacto por volumen: un solo pedido pretende sacar el
conjunto entero de datos de un proyecto. Cuando el procesamiento exista, la
verificación de membresía tiene que estar antes.

## Dependencias

- **Requiere:** [04_projects](04_projects.md); al implementarse, leerá de
  [09_answers](09_answers.md), [08_administrations](08_administrations.md)
  (`instrument_scores`) y [05_participants](05_participants.md), y escribirá en
  R2 vía [10_media_storage](10_media_storage.md).
- **Necesita:** [13_background_jobs](13_background_jobs.md) — el tipo de job no
  existe.

## Gaps conocidos

- **[G-05](00_GAPS.md#g-05--los-exports-nunca-se-procesan)** — no hay
  `JobTypeProcessExport`, el `JobWorker` no conoce exports y `UpdateStatus` no se
  llama desde ningún lado. Todo export queda `pending`.
- **[G-02](00_GAPS.md#g-02--sin-verificación-de-pertenencia-al-proyecto-fuera-del-módulo-projects)** — sin verificación de proyecto.
- `GetExport` implementado y sin ruta; sin endpoint de descarga.
- Esquema de `options` sin definir.
- Sin anonimización: `docs/1_MAIN_FEATURES.md` §12 pide separar el código
  identificable de los datos de respuesta. Es una decisión a tomar **antes** de
  implementar el formato de salida, porque define las columnas.
- Sin retención: no se define cuánto vive el archivo generado en R2.

## Orden de trabajo sugerido

1. Definir el esquema de `options` y las columnas de salida por formato,
   resolviendo primero la anonimización.
2. Agregar `JobTypeProcessExport` y su handler en el worker: `pending` →
   `processing` → generar → subir a R2 → `completed` con `file_key`, o `failed`
   con `error_log`.
3. Exponer `GET /projects/:id/exports/:exportId` con URL de descarga firmada y
   de corta vida.
4. Agregar la verificación de membresía antes de todo lo anterior.

## Requests de ejemplo

[`requests/exports.http`](../../requests/exports.http)
