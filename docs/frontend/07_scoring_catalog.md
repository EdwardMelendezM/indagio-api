# 07 · Catálogo de puntuación

## Pantallas

Constructor de ítems · definición de subescalas · configuración del algoritmo ·
editor de baremos · historial de versiones.

Es la pantalla **más compleja del sistema** y la que hace aplicable un
instrumento: define qué preguntas lo componen, cómo se combinan en un puntaje y
cómo se interpreta el número.

## Tipos

```ts
// ── Request ────────────────────────────────────────────────────────────────
interface ItemRequest {
  question_key: string;          // requerido, máx. 120 — la clave de la respuesta
  subscale_key?: string | null;  // null = solo cuenta para el total
  order_index: number;           // orden de presentación
  item_type: string;             // requerido, libre (ej. "likert_0_3")
  reverse_scored?: boolean;      // default false
  weight?: number;               // default 0, que el motor trata como 1
  value_map?: Record<string, number>;   // { "etiqueta": número }
  is_scored?: boolean | null;    // ⚠️ si se OMITE → true. Si se manda false → false
}

interface SubscaleRequest {
  key: string;                   // requerido, máx. 64
  name: string;                  // requerido, máx. 160
  description?: string | null;
}

interface ScoringRuleRequest {
  subscale_key?: string | null;  // null = REGLA DEL PUNTAJE TOTAL
  algorithm: ScoringAlgorithm;   // requerido
  min_items_required?: number | null;
  missing_strategy: MissingStrategy;  // requerido
  config?: unknown;
}

interface ScoreBandRequest {
  subscale_key?: string | null;
  min_score?: number | null;     // null = extremo abierto
  max_score?: number | null;
  label: string;                 // requerido, máx. 120
  description?: string | null;
}

interface PublishVersionRequest {
  items: ItemRequest[];          // requerido, mínimo 1
  subscales: SubscaleRequest[];  // puede ir vacío
  rules: ScoringRuleRequest[];   // requerido, mínimo 1
  bands: ScoreBandRequest[];     // puede ir vacío
}

// ── Response ───────────────────────────────────────────────────────────────
interface CatalogResponse {
  instrument_id: string;
  version: number;
  items:     ItemResponse[]     | null;   // ⚠️ null, no [], cuando está vacío
  subscales: SubscaleResponse[] | null;   // ⚠️
  rules:     ScoringRuleResponse[] | null;// ⚠️
  bands:     ScoreBandResponse[] | null;  // ⚠️
}

interface ItemResponse {
  id: string;                    // el request no lo lleva; la respuesta sí
  question_key: string;
  subscale_key: string | null;
  order_index: number;
  item_type: string;
  reverse_scored: boolean;
  weight: number;
  value_map: Record<string, number>;
  is_scored: boolean;
}

interface SubscaleResponse    { key: string; name: string; description: string | null }
interface ScoringRuleResponse {
  subscale_key: string | null; algorithm: ScoringAlgorithm;
  min_items_required: number | null; missing_strategy: MissingStrategy; config: unknown;
}
interface ScoreBandResponse   {
  subscale_key: string | null; min_score: number | null;
  max_score: number | null; label: string; description: string | null;
}
```

### ⚠️ Los arrays de la respuesta llegan como `null`

A diferencia del resto de la API, los cuatro arrays de `CatalogResponse` se
construyen sobre un slice nil: **cuando están vacíos, el JSON trae `null`, no
`[]`**. Un PHQ-8 publicado con `subscales: []` se lee de vuelta como
`"subscales": null`.

Normalizar siempre al recibir:

```ts
interface Catalog {
  instrument_id: string; version: number;
  items: ItemResponse[]; subscales: SubscaleResponse[];
  rules: ScoringRuleResponse[]; bands: ScoreBandResponse[];
}

function normalizeCatalog(raw: CatalogResponse): Catalog {
  return {
    instrument_id: raw.instrument_id,
    version: raw.version,
    items:     raw.items     ?? [],
    subscales: raw.subscales ?? [],
    rules:     raw.rules     ?? [],
    bands:     raw.bands     ?? [],
  };
}
```

Sin eso, `catalog.items.map(...)` revienta con "cannot read property map of
null" en cualquier instrumento sin subescalas — es decir, en la mayoría.

## Endpoints

| Método | Ruta | Respuesta |
|---|---|---|
| `POST` | `/projects/:id/instruments/:instrumentID/versions` | `201 CatalogResponse` |
| `GET` | `/projects/:id/instruments/:instrumentID/versions[?version=N]` | `200 CatalogResponse` |
| `GET` | `/projects/:id/instruments/:instrumentID/versions/list` | `200 number[]` |

```ts
export const scoringApi = {
  publish: (projectId: string, instrumentId: string, body: PublishVersionRequest) =>
    api.post<CatalogResponse>(
      `/projects/${projectId}/instruments/${instrumentId}/versions`, body,
    ).then(normalizeCatalog),

  // Sin version → la última publicada
  getCatalog: (projectId: string, instrumentId: string, version?: number) =>
    api.get<CatalogResponse>(
      `/projects/${projectId}/instruments/${instrumentId}/versions${
        version !== undefined ? `?version=${version}` : ""
      }`,
    ).then(normalizeCatalog),

  listVersions: (projectId: string, instrumentId: string) =>
    api.get<number[]>(`/projects/${projectId}/instruments/${instrumentId}/versions/list`),
};
```

`listVersions` **sí** devuelve `[]` cuando no hay nada publicado (ese slice se
inicializa bien). Un array vacío significa "instrumento no aplicable".

## Pedir la versión correcta — regla de oro

```ts
// ❌ MAL: al renderizar un formulario, "la última" puede no ser la del intento
const catalog = await scoringApi.getCatalog(projectId, instrumentId);

// ✅ BIEN: la versión congelada que devolvió StartAdministration
const catalog = await scoringApi.getCatalog(
  projectId, instrumentId, administration.instrument_version,
);
```

La aplicación congela su `instrument_version` al iniciarse. Si alguien publica
una versión nueva mientras la entrevista está en curso, pedir "la última"
mostraría un formulario que **no corresponde** al puntaje que se va a calcular.

Usar sin `version` solo en pantallas de administración del catálogo, nunca en el
runner. Ver [08_administrations.md](08_administrations.md).

## Publicar: el body va completo y es inmutable

Cada publicación manda **todo** el catálogo, no un diff. El backend incrementa
`instruments.version` e inserta un juego nuevo de filas; nunca toca las
anteriores. Por eso una aplicación vieja siempre se puede recalcular con el
catálogo original.

Consecuencias para la UI:

- **No hay edición.** Corregir una errata obliga a publicar una versión completa
  nueva. Conviene que el constructor permita "duplicar la versión N y editar",
  porque es la operación real:
  ```ts
  const base = await scoringApi.getCatalog(projectId, instrumentId);
  const draft: PublishVersionRequest = {
    items: base.items.map(({ id, ...rest }) => rest),   // quitar los id
    subscales: base.subscales,
    rules: base.rules,
    bands: base.bands,
  };
  // el usuario edita draft, y luego:
  await scoringApi.publish(projectId, instrumentId, draft);
  ```
- **No hay despublicar.** Una versión errónea queda en el historial para siempre.
  Vale la pena un diálogo de confirmación que muestre un resumen antes de
  publicar: nº de ítems, subescalas, reglas y rango de puntajes posible.
- Las subescalas se **declaran en cada publicación**: no se heredan de la versión
  anterior. Si se omite el array, los ítems que las referencien fallan.

## Lo que el backend valida y lo que no

### Sí valida

1. `items` no vacío (mínimo 1).
2. `rules` no vacío (mínimo 1).
3. **Integridad de `subscale_key`**: todo valor citado en `items`, `rules` o
   `bands` debe existir en el array `subscales` del **mismo** request. Si no:
   `422` con mensaje que nombra la clave huérfana.

### No valida — hay que hacerlo en el cliente

| Falta | Qué pasa si se ignora |
|---|---|
| `question_key` único dentro del request | Error de constraint crudo desde Postgres, mensaje ilegible |
| Que exista la regla del total (`subscale_key: null`) | Se publica sin puntaje total y nadie avisa |
| Una regla por cada subescala declarada | La subescala no produce puntaje |
| Forma de `value_map` (`{string: number}`) | Falla recién al puntuar, mucho después |
| Baremos sin solapamientos ni huecos | Gana el primero que coincida, por orden de inserción |
| Que `algorithm` sea del enum | Se guarda y rompe el cálculo con `unknown algorithm` |

Validador recomendado antes de enviar:

```ts
function validateCatalog(req: PublishVersionRequest): string[] {
  const errors: string[] = [];

  if (req.items.length === 0) errors.push("Se requiere al menos un ítem.");
  if (req.rules.length === 0) errors.push("Se requiere al menos una regla.");

  // question_key duplicados
  const keys = req.items.map(i => i.question_key);
  const dups = keys.filter((k, idx) => keys.indexOf(k) !== idx);
  if (dups.length) errors.push(`Claves de pregunta duplicadas: ${[...new Set(dups)].join(", ")}`);

  // la regla del total es la que el backend NO exige pero todos esperan
  if (!req.rules.some(r => r.subscale_key == null))
    errors.push("Falta la regla del puntaje total (subscale_key: null).");

  // cada subescala declarada debería tener su regla
  for (const s of req.subscales)
    if (!req.rules.some(r => r.subscale_key === s.key))
      errors.push(`La subescala "${s.name}" no tiene regla de puntuación.`);

  // integridad referencial (el backend también la valida, pero mejor antes)
  const declared = new Set(req.subscales.map(s => s.key));
  for (const i of req.items)
    if (i.subscale_key && !declared.has(i.subscale_key))
      errors.push(`El ítem "${i.question_key}" referencia una subescala inexistente.`);

  // value_map numérico
  for (const i of req.items)
    for (const [label, v] of Object.entries(i.value_map ?? {}))
      if (typeof v !== "number" || Number.isNaN(v))
        errors.push(`value_map de "${i.question_key}": "${label}" no es numérico.`);

  // baremos solapados, por subescala
  const byScale = new Map<string, ScoreBandRequest[]>();
  for (const b of req.bands) {
    const k = b.subscale_key ?? "__total__";
    byScale.set(k, [...(byScale.get(k) ?? []), b]);
  }
  for (const [scale, bands] of byScale) {
    const sorted = [...bands].sort((a, b) => (a.min_score ?? -Infinity) - (b.min_score ?? -Infinity));
    for (let i = 1; i < sorted.length; i++)
      if ((sorted[i].min_score ?? -Infinity) <= (sorted[i - 1].max_score ?? Infinity))
        errors.push(`Baremos solapados en ${scale}: "${sorted[i - 1].label}" y "${sorted[i].label}".`);
  }

  return errors;
}
```

## Algoritmos: dos no están implementados

| `algorithm` | Qué hace realmente |
|---|---|
| `sum` | Suma de `valor × peso`, pasada por `missing_strategy` |
| `weighted_sum` | Idéntico a `sum` (el peso ya se aplica siempre) |
| `average` | `suma / suma_de_pesos` |
| `lookup_table` | ⚠️ **devuelve la suma simple**: no está implementado |
| `custom` | ⚠️ **devuelve la suma simple**: no está implementado |

Si la UI ofrece `lookup_table` o `custom`, el usuario creerá que configuró algo
que no se aplica, y el puntaje saldrá mal sin ningún aviso. **Recomendación:
ocultarlos del selector** hasta que el backend los implemente, o marcarlos como
"no disponible".

## `missing_strategy`: el nombre engaña en un caso

| Valor | Qué hace |
|---|---|
| `prorate` | Escala la suma: `suma × (esperados / respondidos)` |
| `zero` | Los no respondidos valen 0. Sin ninguna respuesta, el puntaje es `0`, no `null` |
| `null_if_missing` | ⚠️ **no anula nada**: se comporta igual que `zero` |

Si el estudio necesita "puntaje nulo cuando falta un ítem", **no se consigue con
`null_if_missing`**: se consigue poniendo `min_items_required` igual al número
total de ítems de la regla.

```ts
// "Exigir todos los ítems" se expresa así, no con null_if_missing
{ subscale_key: null, algorithm: "sum",
  min_items_required: items.filter(i => i.is_scored).length,
  missing_strategy: "zero", config: {} }
```

Conviene exponer `min_items_required` como un control visible —"mínimo de ítems
para dar un puntaje válido"— porque es el único mecanismo que realmente
descarta un puntaje incompleto.

## `is_scored` y `weight`: dos defaults asimétricos

```ts
// is_scored es *bool en Go: OMITIR ≠ mandar false
{ question_key: "q1" }                      // → is_scored: true  (por omisión)
{ question_key: "q1", is_scored: false }     // → is_scored: false
```

```ts
// weight es float64 (no puntero): omitirlo manda 0, y el motor trata 0 como 1
{ question_key: "q1" }                       // → weight 0 → el motor usa 1
{ question_key: "q1", weight: 2 }            // → weight 2
```

El resultado es el mismo, pero el valor guardado difiere: la respuesta traerá
`weight: 0` si se omitió. Para que la UI muestre lo que el usuario puso,
**enviar siempre `weight: 1` explícito**.

## `reverse_scored` depende de `value_map`

La inversión se calcula como `max(value_map) - valor`. Si el ítem no tiene
`value_map`, el máximo es 0 y **la inversión no hace nada** silenciosamente.

```ts
// ⚠️ reverse_scored sin value_map no invierte nada
if (item.reverse_scored && Object.keys(item.value_map ?? {}).length === 0)
  errors.push(`"${item.question_key}": reverse_scored requiere value_map.`);
```

Agregar esa comprobación al validador de arriba.

## Ejemplo completo: PHQ-8

```ts
const PHQ8: PublishVersionRequest = {
  items: Array.from({ length: 8 }, (_, i) => ({
    question_key: `phq8_${i + 1}`,
    order_index: i + 1,
    item_type: "likert_0_3",
    weight: 1,
    is_scored: true,
    value_map: { "0": 0, "1": 1, "2": 2, "3": 3 },
  })),
  subscales: [],                      // sin subescalas: un solo puntaje total
  rules: [{
    subscale_key: null,               // la regla del total
    algorithm: "sum",
    min_items_required: 6,            // con menos de 6 respuestas, no hay puntaje
    missing_strategy: "prorate",
    config: {},
  }],
  bands: [
    { subscale_key: null, min_score: 0,  max_score: 4,  label: "Mínima" },
    { subscale_key: null, min_score: 5,  max_score: 9,  label: "Leve" },
    { subscale_key: null, min_score: 10, max_score: 14, label: "Moderada" },
    { subscale_key: null, min_score: 15, max_score: 19, label: "Moderadamente severa" },
    { subscale_key: null, min_score: 20, max_score: 24, label: "Severa" },
  ],
};
```

Un instrumento **con** subescalas necesita declararlas y darle una regla a cada
una, más la del total:

```ts
{
  subscales: [
    { key: "depresion", name: "Depresión" },
    { key: "ansiedad",  name: "Ansiedad" },
  ],
  items: [
    { question_key: "d1", subscale_key: "depresion", /* ... */ },
    { question_key: "a1", subscale_key: "ansiedad",  /* ... */ },
  ],
  rules: [
    { subscale_key: null,        algorithm: "sum", missing_strategy: "prorate" },
    { subscale_key: "depresion", algorithm: "sum", missing_strategy: "prorate" },
    { subscale_key: "ansiedad",  algorithm: "sum", missing_strategy: "prorate" },
  ],
  bands: [ /* se pueden definir por subescala y para el total */ ],
}
```

## Validación en el cliente

| Campo | Regla |
|---|---|
| `items` | mínimo 1 |
| `items[].question_key` | requerido, máx. 120, **único en el request** |
| `items[].item_type` | requerido (string libre) |
| `subscales[].key` | requerido, máx. 64 |
| `subscales[].name` | requerido, máx. 160 |
| `rules` | mínimo 1 |
| `rules[].algorithm` | requerido, del enum |
| `rules[].missing_strategy` | requerido, del enum |
| `bands[].label` | requerido, máx. 120 |

Todos los arrays usan `dive`: el error de un elemento invalida el request
completo, y el `422` no dice cuál. De ahí la importancia de validar antes.

## Errores

| Código | Significado | Acción en UI |
|---|---|---|
| `400` | UUID de ruta malformado, o `?version` no numérico | Error de navegación |
| `404` | La versión pedida no existe, o no hay catálogo publicado | "Sin catálogo publicado" |
| `422` | Validación: arrays vacíos, `subscale_key` huérfana, campos fuera de rango | Mostrar los errores del **validador del cliente**, no los de la API |
| `500` | `question_key` duplicado (constraint de Postgres) | Mensaje genérico; prevenir en el cliente |

⚠️ **El `422` no dice qué falló.** El backend construye internamente mensajes
precisos —`item "phq8_1" references unknown subscale "x"`— pero los envuelve
sobre un error centinela, y el manejador solo devuelve el mensaje del centinela.
La respuesta real es siempre:

```json
{ "error": "validation failed" }
```

El detalle queda en los logs del servidor. Para el formulario del constructor de
catálogos eso es especialmente grave, porque es el body más complejo de la API:
un error en cualquiera de los cuatro arrays devuelve el mismo mensaje opaco.

**Por eso el validador del cliente de la sección anterior no es un adorno: es la
única fuente de mensajes útiles para el usuario.** Conviene ejecutarlo siempre
antes de enviar y no depender nunca de la respuesta de la API para señalar el
campo.

## Trampas

1. **Los cuatro arrays llegan como `null` si están vacíos.** Normalizar siempre.
2. **En el runner, pedir la versión congelada**, nunca la última.
3. **Publicar es inmutable**: no hay editar ni despublicar. Ofrecer "duplicar y
   editar".
4. **Las subescalas se declaran en cada publicación**: no se heredan.
5. **`lookup_table` y `custom` no están implementados** y devuelven la suma
   simple sin avisar. Ocultarlos.
6. **`null_if_missing` no anula**: usar `min_items_required` para eso.
7. **Omitir `is_scored` da `true`**; omitir `weight` guarda `0`. Enviar ambos
   explícitos.
8. **`reverse_scored` sin `value_map` no invierte nada.**
9. **El backend no exige la regla del total**, pero sin ella no hay puntaje
   global. Exigirla en el cliente.
10. **`question_key` duplicado da `500`**, no `422`: validar antes.
11. **Los baremos se evalúan por orden de inserción**, así que con rangos
    solapados gana el primero. El orden del array importa.
12. **Verificar que haya catálogo antes de ofrecer "aplicar"**: `listVersions`
    devolviendo `[]` significa que el instrumento no es aplicable.
