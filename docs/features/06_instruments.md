# 06 · Instrumentos

## Propósito

Un instrumento es una herramienta de medición del proyecto (PHQ-8, DASS-21, una
entrevista semiestructurada). Esta tabla guarda su **identidad y presentación**;
cómo se puntúa vive aparte, en el catálogo versionado de
[07_scoring_catalog](07_scoring_catalog.md). Paso 8 del
[flujo](../2_PROJECT_FLOW.md).

La separación es deliberada: el instrumento es estable, su esquema de puntuación
evoluciona por versiones sin perder la trazabilidad de lo ya recolectado.

## Entidades y tablas

### `instruments`

| Columna | Tipo | Nota |
|---|---|---|
| `id` | `UUID` PK | |
| `project_id` | `UUID` → `projects` | `ON DELETE CASCADE` |
| `name` | `VARCHAR(160)` | |
| `kind` | `VARCHAR(64)` | ver enum abajo |
| `config` | `JSONB` | libre, default `{}` |
| `version` | `INTEGER` | default 1; lo incrementa `PublishVersion` |
| `status` | `VARCHAR(32)` | `draft` \| `active` \| `archived` |
| `created_by` | `UUID` → `users` | |

`version` **no se gestiona desde este módulo**: es el contador que mueve
[07_scoring_catalog](07_scoring_catalog.md) al publicar un catálogo nuevo. Un
instrumento recién creado está en `version = 1` sin tener catálogo publicado —
condición en la que `StartAdministration` falla.

### `kind`

`domain/instrument.go:12-20`:

`text` · `numeric` · `scale` · `single_choice` · `multi_choice` · `audio` · `video`

Se valida que el valor pertenezca al enum, pero **no se valida `config` contra
el `kind`**: un `single_choice` sin opciones se acepta sin objeción.

### `config` por convención

No hay esquema declarado. Lo que usan los ejemplos del repo:

```jsonc
{
  "description": "Patient Health Questionnaire - 8 items",
  "response_labels": ["Ningún día", "Varios días",
                      "Más de la mitad de los días", "Casi todos los días"]
}
```

`response_labels` es la presentación; el mapeo a números para puntuar es el
`value_map` de cada item del catálogo. Son dos cosas distintas y conviven.

## Ciclo de vida

```
draft ──> active ──> archived
```

Sin transiciones validadas. El `status` no se consulta en ninguna parte: una
aplicación puede iniciarse sobre un instrumento `draft` o `archived` sin problema.

## Endpoints

| Método | Ruta | Descripción |
|---|---|---|
| `POST` | `/api/projects/:id/instruments` | Crear |
| `GET` | `/api/projects/:id/instruments` | Listar los del proyecto |
| `PATCH` | `/api/projects/:id/instruments/:instrumentID` | Actualizar nombre, `config`, `status` |

`instrument_handler.go:165-170`. No hay `GET /:instrumentID` ni `DELETE`.

## Reglas de negocio

- `CreateInstrument` (`instrument_usecase.go:22`) valida ids no nulos, nombre y
  que `kind` esté en el enum. `config` solo se valida como JSON bien formado.
- `UpdateInstrument` acepta `name`, `config` y `status` como punteros: solo se
  actualiza lo enviado.
- Actualizar `config` **no versiona nada**. El catálogo de puntuación sí se
  versiona; la presentación no. Cambiar `response_labels` después de recolectar
  altera retroactivamente cómo se lee una respuesta ya guardada.

## Autorización

| | |
|---|---|
| **Implementada** | Solo `actorID != uuid.Nil`. |
| **Esperada** | Miembro para leer; `owner`/`admin` para crear y modificar. |

`UpdateInstrument` recibe solo `instrumentID`, sin `project_id`: como en
participantes, haría falta resolver el proyecto por el instrumento antes de
poder autorizar.

## Dependencias

- **Requiere:** [04_projects](04_projects.md).
- **Lo consume:** [07_scoring_catalog](07_scoring_catalog.md) (cuelga del
  `instrument_id` + `version`), [08_administrations](08_administrations.md),
  [09_answers](09_answers.md).

## Gaps conocidos

- **[G-02](00_GAPS.md#g-02--sin-verificación-de-pertenencia-al-proyecto-fuera-del-módulo-projects)** — sin verificación de proyecto.
- **[G-13](00_GAPS.md#g-13--1_main_featuresmd-está-desactualizado)** — la tabla
  `questionnaires` fue eliminada en la migración `20260912211033`, pero
  `requests/instruments.http` todavía documenta
  `POST /projects/:id/questionnaires` y `.../publish`. `domain.Questionnaire`
  sigue declarado en `domain/instrument.go:46` sin tabla detrás.
- **[G-15](00_GAPS.md#g-15--sin-transiciones-de-estado-validadas)** — `status`
  sin transiciones y sin efecto: no impide aplicar un instrumento `archived`.
- `config` no se valida contra `kind`.
- Cambiar `config` afecta retroactivamente la lectura de respuestas ya tomadas.
- No hay `GET` de un instrumento puntual ni borrado.
- Los instrumentos son por proyecto: no hay plantillas reutilizables entre
  proyectos, como preveía `docs/1_MAIN_FEATURES.md` §4. Montar el PHQ-8 en un
  proyecto nuevo obliga a recrear instrumento y catálogo completos.

## Requests de ejemplo

[`requests/instruments.http`](../../requests/instruments.http) — ignorar la
sección de `questionnaires`.
