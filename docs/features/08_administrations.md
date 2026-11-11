# 08 · Aplicaciones y cálculo de puntajes

## Propósito

Una *administration* es **un intento concreto de un participante con un
instrumento**: agrupa las respuestas de esa sesión y fija la versión de catálogo
con la que se puntuará. Es la unidad de análisis — un mismo participante puede
responder el PHQ-8 en el pretest y en el postest, y cada toma es una aplicación
independiente con su propio puntaje.

Pasos 10 y 13 del [flujo](../2_PROJECT_FLOW.md).

## Entidades y tablas

### `instrument_administrations`

| Columna | Tipo | Nota |
|---|---|---|
| `project_id`, `participant_id`, `instrument_id` | `UUID` | `ON DELETE CASCADE` |
| `instrument_version` | `INTEGER` | **congelado** al iniciar |
| `status` | `VARCHAR(32)` | `in_progress` \| `completed` \| `abandoned` |
| `sync_status` | `VARCHAR(32)` | default `pending` |
| `client_generated_id` | `VARCHAR(128)` NULL | `UNIQUE (project_id, participant_id, client_generated_id)` |
| `started_at` / `completed_at` | `TIMESTAMPTZ` | |

`client_generated_id` da la idempotencia que necesita el cliente móvil: si la
llamada se reintenta tras un corte de red, el `UNIQUE` evita el intento duplicado.

### `instrument_scores`

| Columna | Tipo | Nota |
|---|---|---|
| `administration_id` | `UUID` | `UNIQUE (administration_id, subscale_key)` |
| `subscale_key` | `VARCHAR(64)` NULL | **`NULL` = puntaje total** |
| `raw_score` | `NUMERIC` NULL | `NULL` si no se alcanzó `min_items_required` |
| `scaled_score` | `NUMERIC` NULL | columna presente, el motor **nunca la llena** |
| `items_answered` / `items_expected` | `INTEGER` | permite auditar la completitud |
| `band_id` | `UUID` → `instrument_score_bands` | `ON DELETE SET NULL` |
| `computation_version` | `VARCHAR(32)` NULL | previsto para marcar algoritmos especiales; hoy siempre `NULL` |
| `metadata` | `JSONB` | default `{}` |

`answer_records.administration_id` se agregó en la migración
`20260923234729`: es la FK que permite recoger las respuestas de un intento.

## Ciclo de vida

```
in_progress ──> completed   (calcula puntajes)
            └─> abandoned   (no calcula nada)
```

- `CompleteAdministration` rechaza con `409 ErrConflict` si ya estaba
  `abandoned` (`instrument_administration.go:44`).
- `AbandonAdministration` (`:58`) **no verifica nada**: puede abandonar una
  aplicación ya `completed`, dejando los `instrument_scores` ya calculados
  colgando de una aplicación en estado `abandoned`.
- Volver a llamar `complete` sobre una aplicación `completed` recalcula y
  sobrescribe los puntajes.

## El cálculo

`computeAndSave` (`instrument_administration.go:90`) encadena cuatro pasos:

```
GetCatalogByVersion(instrument_id, instrument_version)   ← la versión CONGELADA
      ↓
ListAnswersByAdministration(administration_id)
      ↓
ScoringEngine.Compute(items, rules, bands, answers)      ← puro, sin BD
      ↓
SaveScores(administration_id, scores)
```

El `ScoringEngine` (`internal/domain/scoring.go`) es una función pura y
determinista: mismas entradas, mismas salidas, sin acceso a base de datos. Eso
es lo que permite reproducir un puntaje meses después para auditarlo, y lo que
lo hace testeable contra fixtures.

### Algoritmo, paso a paso

Por cada regla del catálogo (`computeOne`, `scoring.go:81`):

1. Selecciona los ítems de su alcance: los de la subescala, o **todos los
   `is_scored`** si es la regla del total.
2. Por cada ítem con respuesta, extrae el valor numérico (ver abajo).
3. Si `reverse_scored`, lo invierte: `max(value_map) - value`.
4. Acumula `sum += value × weight` y `weightSum += weight`. Un `weight` de 0 se
   trata como 1.
5. Si `items_answered < min_items_required` → **sale con `raw_score = NULL`**.
   El puntaje parcial se descarta deliberadamente: con muy pocos ítems no es
   interpretable.
6. Aplica el algoritmo:
   - `sum` / `weighted_sum` → la suma, pasada por `missing_strategy`;
   - `average` → `sum / weightSum`;
   - `lookup_table` / `custom` → **la suma simple**, sin implementar;
   - desconocido → error `unknown algorithm`.
7. Busca la primera banda que contenga el `raw_score` y adjunta su `label`.

### `missing_strategy`

- `prorate` — escala la suma como si se hubieran respondido todos los ítems:
  `sum × (expected / answered)` (`scoring.go:152`).
- `zero` — deja la suma tal cual; los no respondidos valen 0. Si no hay ninguna
  respuesta, fija `raw_score = 0` en lugar de `NULL`.
- `null_if_missing` — deja la suma tal cual. ⚠️ **El nombre promete más de lo que
  hace**: no anula el puntaje cuando falta un ítem, se comporta igual que `zero`.

### Extracción del valor — punto crítico

`extractItemScore` (`scoring.go:194`) espera `answer_records.value` con la forma
`{"value": X}`:

- número → se usa directo;
- string → se traduce con el `value_map` del ítem; si la etiqueta no está en el
  mapa, **error**; si el ítem no tiene `value_map`, **error**;
- clave `value` ausente → `ok = false`, el ítem cuenta como **no respondido, sin
  error**.

Ese último caso es
[G-01](00_GAPS.md#g-01--el-envoltorio-de-value-no-coincide-con-el-motor-de-puntuación):
los ejemplos del repo mandan `{"raw": 2}` y el cálculo entero sale `NULL` sin
decir por qué.

## Endpoints

| Método | Ruta | Descripción |
|---|---|---|
| `POST` | `/api/projects/:id/participants/:participantID/instruments/:instrumentID/administrations` | Iniciar; congela la versión |
| `GET` | `/api/projects/:id/participants/:participantID/instruments/:instrumentID/administrations` | Historial de intentos |
| `PATCH` | `/api/administrations/:administrationId/complete` | Completar y puntuar |
| `PATCH` | `/api/administrations/:administrationId/abandon` | Abandonar |
| `GET` | `/api/administrations/:administrationId/scores` | Leer puntajes guardados |
| `POST` | `/api/administrations/:administrationId/scores/recompute` | Recalcular |

`instrument_administrations_handler.go:234-245`. Las cuatro últimas van por un
grupo plano `/administrations`, sin `project_id` en la ruta — razón por la cual
no pueden autorizar por proyecto sin una consulta adicional.

## Reglas de negocio

- `StartAdministration` (`:24`) resuelve `GetLatestCatalog` y congela
  `catalog.Version`. **Si el instrumento no tiene catálogo publicado, falla
  aquí**: es el error más común al montar un proyecto nuevo.
- No valida que el participante pertenezca al proyecto, ni el instrumento, ni que
  no haya ya una aplicación `in_progress` del mismo par
  participante/instrumento. Se pueden abrir intentos concurrentes.
- `GetScores` solo lee; no recalcula.
- `RecomputeScores` (`:79`) repite `computeAndSave` sobre las mismas respuestas
  y el mismo catálogo congelado, así que hoy siempre da el mismo resultado —
  [G-07](00_GAPS.md#g-07--no-se-puede-editar-una-respuesta-así-que-recompute-no-sirve).

## Autorización

| | |
|---|---|
| **Implementada** | Solo `actorID != uuid.Nil`. |
| **Esperada** | Miembro del proyecto; el proyecto habría que resolverlo desde `administration_id` en los endpoints planos. |

## Dependencias

- **Requiere:** [05_participants](05_participants.md),
  [07_scoring_catalog](07_scoring_catalog.md) (obligatorio: sin catálogo no
  arranca), [09_answers](09_answers.md) para tener algo que puntuar.
- **Lo consume:** [12_exports](12_exports.md) sería el consumidor natural de
  `instrument_scores`, cuando se implemente.

## Gaps conocidos

- **[G-01](00_GAPS.md#g-01--el-envoltorio-de-value-no-coincide-con-el-motor-de-puntuación)** — el envoltorio de `value` rompe el cálculo en silencio. **P0.**
- **[G-02](00_GAPS.md#g-02--sin-verificación-de-pertenencia-al-proyecto-fuera-del-módulo-projects)** — sin verificación de proyecto.
- **[G-07](00_GAPS.md#g-07--no-se-puede-editar-una-respuesta-así-que-recompute-no-sirve)** — `recompute` no puede cambiar nada.
- **[G-15](00_GAPS.md#g-15--sin-transiciones-de-estado-validadas)** — `abandon`
  no valida el estado previo y puede desconectar puntajes ya calculados.
- `scaled_score` y `computation_version` nunca se escriben.
- `lookup_table` y `custom` devuelven la suma simple sin avisar.
- `null_if_missing` no hace lo que su nombre indica.
- No se impide abrir varias aplicaciones `in_progress` del mismo par
  participante/instrumento.
- `sync_status` nunca se mueve de `pending`.
- `CompleteAdministration` marca `completed` **antes** de calcular: si el cálculo
  falla, la aplicación queda cerrada y sin puntajes.

## Requests de ejemplo

[`requests/instrument_administrations.http`](../../requests/instrument_administrations.http)
