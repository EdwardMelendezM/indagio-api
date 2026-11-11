# 07 · Catálogo de puntuación versionado

## Propósito

Define **cómo se puntúa** un instrumento: qué ítems lo componen, cómo se agrupan
en subescalas, con qué algoritmo se combinan y cómo se interpreta el número
resultante. Todo etiquetado con un número de versión inmutable.

Es el paso 9 del [flujo](../2_PROJECT_FLOW.md) y la **bisagra del sistema**: sin
un catálogo publicado, no se puede iniciar ninguna aplicación.

### Por qué versionado

Si se corrige el esquema de puntuación a mitad de un estudio, los datos ya
recolectados no deben cambiar de significado. `PublishVersion` nunca muta filas
de versiones anteriores: inserta un juego nuevo con el número siguiente. Cada
aplicación guarda el `instrument_version` con el que se tomó, así que siempre se
puede recalcular contra el catálogo original. Es lo que hace reproducible —
auditable — un puntaje de investigación.

## Entidades y tablas

Cuatro tablas, todas con `UNIQUE (instrument_id, instrument_version, …)`:

### `instrument_items` — los ítems

| Columna | Tipo | Nota |
|---|---|---|
| `question_key` | `VARCHAR(120)` | la clave que mandará la respuesta |
| `subscale_key` | `VARCHAR(64)` NULL | `NULL` = solo cuenta para el total |
| `order_index` | `INTEGER` | orden de presentación |
| `item_type` | `VARCHAR(32)` | default `likert`; libre (p. ej. `likert_0_3`) |
| `reverse_scored` | `BOOLEAN` | invierte el valor contra el máximo del `value_map` |
| `weight` | `NUMERIC(10,4)` | default 1; `0` se trata como 1 al computar |
| `value_map` | `JSONB` | `{"etiqueta": número}` |
| `is_scored` | `BOOLEAN` | `false` = informativo, se excluye del cálculo |

### `instrument_subscales` — las dimensiones

`key`, `name`, `description`. Un instrumento sin subescalas (como el PHQ-8) manda
`subscales: []`.

### `instrument_scoring_rules` — una regla por puntaje a calcular

| Columna | Nota |
|---|---|
| `subscale_key` | `NULL` = **regla del puntaje total**, obligatoria |
| `algorithm` | `sum` \| `average` \| `weighted_sum` \| `lookup_table` \| `custom` |
| `min_items_required` | si se responden menos, `raw_score` queda `NULL` |
| `missing_strategy` | `prorate` \| `zero` \| `null_if_missing` |
| `config` | `JSONB` para `lookup_table` / `custom` |

### `instrument_score_bands` — la interpretación

`min_score`, `max_score`, `label`, `description`. Rangos inclusivos en ambos
extremos; `NULL` en un extremo lo deja abierto. Se evalúan **en orden** y gana la
primera que contenga el puntaje (`scoring.go:69-75`), así que rangos solapados se
resuelven por orden de inserción.

## Endpoints

| Método | Ruta | Descripción |
|---|---|---|
| `POST` | `/api/projects/:id/instruments/:instrumentID/versions` | Publicar versión nueva |
| `GET` | `/api/projects/:id/instruments/:instrumentID/versions` | Catálogo; `?version=N` o la última |
| `GET` | `/api/projects/:id/instruments/:instrumentID/versions/list` | Números de versión publicados |

`instrument_scoring_handler.go:152-157`.

Sin `?version`, el `GET` devuelve la última. **Al renderizar un formulario hay
que pedir siempre la versión congelada que devolvió `StartAdministration`**, no
la última: si alguien publica entre el inicio y el fin de la entrevista, el
formulario dejaría de corresponder al puntaje que se va a calcular.

## Contrato de publicación

El body lleva el catálogo completo. Ejemplo mínimo (PHQ-8, sin subescalas):

```jsonc
{
  "items": [
    { "question_key": "phq8_1", "order_index": 1, "item_type": "likert_0_3",
      "weight": 1, "is_scored": true, "value_map": { "0": 0, "1": 1, "2": 2, "3": 3 } }
    // ... hasta phq8_8
  ],
  "subscales": [],
  "rules": [
    { "subscale_key": null, "algorithm": "sum", "min_items_required": 6,
      "missing_strategy": "prorate", "config": {} }
  ],
  "bands": [
    { "subscale_key": null, "min_score": 0,  "max_score": 4,  "label": "Mínima" },
    { "subscale_key": null, "min_score": 5,  "max_score": 9,  "label": "Leve" },
    { "subscale_key": null, "min_score": 10, "max_score": 14, "label": "Moderada" },
    { "subscale_key": null, "min_score": 15, "max_score": 19, "label": "Moderadamente severa" },
    { "subscale_key": null, "min_score": 20, "max_score": 24, "label": "Severa" }
  ]
}
```

## Reglas de negocio

Validaciones de `PublishVersion` (`instrument_scoring.go:20-51`), en orden:

1. `items` no vacío.
2. `rules` no vacío — "at least a total-score rule is required".
3. Integridad referencial de `subscale_key`: todo valor citado en `items`,
   `rules` o `bands` debe existir en el array `subscales` **del mismo request**.
   Por eso el orden dentro del body importa conceptualmente: las subescalas se
   declaran ahí, no se heredan de versiones anteriores.

Lo que **no** se valida:

- Que `question_key` sea único dentro del request (lo atrapa después el `UNIQUE`
  de la tabla, con un error de constraint crudo).
- Que exista la regla del total (`subscale_key: null`). Se exige "al menos una
  regla", pero un catálogo con solo reglas de subescala pasa y no produce puntaje
  total.
- Que haya una regla por cada subescala declarada.
- Que `value_map` sea un objeto `{string: number}` — un `value_map` malformado
  falla recién al puntuar.
- Que los rangos de `bands` no se solapen ni dejen huecos.
- Que `algorithm` pertenezca al enum: un valor desconocido se guarda y hace
  fallar el cálculo con `unknown algorithm` (`scoring.go:138`).

`lookup_table` y `custom` **no están implementados** en el motor: dejan la suma
simple como `raw_score` y confían en que alguien post-procese usando
`computation_version` como marca (`scoring.go:131-136`).

## Autorización

| | |
|---|---|
| **Implementada** | Solo `actorID != uuid.Nil`. |
| **Esperada** | `owner`/`admin` del proyecto para publicar; miembro para leer. |

Publicar es la operación más delicada del módulo: define cómo se puntúa un
estudio entero. Hoy cualquier usuario autenticado puede publicar una versión en
el instrumento de cualquier proyecto —
[G-02](00_GAPS.md#g-02--sin-verificación-de-pertenencia-al-proyecto-fuera-del-módulo-projects).

## Dependencias

- **Requiere:** [06_instruments](06_instruments.md) — el catálogo cuelga de
  `instrument_id` y mueve `instruments.version`.
- **Lo consume:** [08_administrations](08_administrations.md) —
  `StartAdministration` llama `GetLatestCatalog` para congelar la versión, y
  `computeAndSave` llama `GetCatalogByVersion` para puntuar.

## Gaps conocidos

- **[G-02](00_GAPS.md#g-02--sin-verificación-de-pertenencia-al-proyecto-fuera-del-módulo-projects)** — sin verificación de proyecto.
- `lookup_table` y `custom` aceptados pero no implementados; devuelven la suma
  simple en silencio.
- Sin validación de regla total, de cobertura de subescalas, de forma del
  `value_map`, ni de solapamiento de bandas.
- No se puede despublicar ni corregir una versión: el diseño inmutable es
  correcto, pero un error de tipeo obliga a publicar otra versión completa.
- No hay copia de catálogo entre instrumentos ni proyectos (ligado a la falta de
  plantillas reutilizables, [06_instruments](06_instruments.md)).

## Requests de ejemplo

[`requests/instrument_scoring.http`](../../requests/instrument_scoring.http) —
catálogo completo del PHQ-8, listo para copiar.
