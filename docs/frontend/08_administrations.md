# 08 · Aplicaciones y puntajes

## Pantallas

Runner del instrumento (iniciar → responder → completar) · historial de intentos
de un participante · vista de resultados y baremos.

Una *administration* es **un intento concreto de un participante con un
instrumento**. Es la unidad que agrupa las respuestas y la que produce el
puntaje. Un mismo participante puede tener varios intentos del mismo instrumento
(pretest y postest, por ejemplo), cada uno con su puntaje independiente.

## Tipos

```ts
interface StartAdministrationRequest {
  client_generated_id?: string | null;   // recomendado: UUID generado en el cliente
}

interface AdministrationResponse {
  id: string;
  project_id: string;
  participant_id: string;
  instrument_id: string;
  instrument_version: number;            // ⚠️ CONGELADA: usarla para pedir el catálogo
  status: AdministrationStatus;
  sync_status: string;                   // siempre "pending" hoy
  client_generated_id: string | null;
  started_at: string;
  completed_at: string | null;
  // Sin created_at ni updated_at.
}

interface ScoreResponse {
  subscale_key: string | null;      // null = PUNTAJE TOTAL
  raw_score: number | null;         // null = no se alcanzó min_items_required
  scaled_score: number | null;      // ⚠️ siempre null: el backend no lo calcula
  items_answered: number;
  items_expected: number;
  band_label: string | null;        // la interpretación: "Moderada", "Leve"...
  computation_version: string | null;  // ⚠️ siempre null
  computed_at: string;
  metadata: unknown;                // siempre {}
}

interface CompleteAdministrationResponse {
  administration: AdministrationResponse;
  scores: ScoreResponse[];          // este sí viene como [] si está vacío
}
```

⚠️ `scaled_score`, `computation_version` y `metadata` existen en el contrato
pero **el backend nunca los llena**. No construir UI que dependa de ellos.

## Endpoints

| Método | Ruta | Respuesta |
|---|---|---|
| `POST` | `/projects/:id/participants/:participantID/instruments/:instrumentID/administrations` | `201 AdministrationResponse` |
| `GET` | `/projects/:id/participants/:participantID/instruments/:instrumentID/administrations` | `200 AdministrationResponse[]` |
| `PATCH` | `/administrations/:administrationId/complete` | `200 CompleteAdministrationResponse` |
| `PATCH` | `/administrations/:administrationId/abandon` | `200 AdministrationResponse` |
| `GET` | `/administrations/:administrationId/scores` | `200 ScoreResponse[]` |
| `POST` | `/administrations/:administrationId/scores/recompute` | `200 ScoreResponse[]` |

⚠️ **Las rutas cambian de forma a mitad del flujo.** Iniciar y listar van
anidados bajo el proyecto y el participante; completar, abandonar y leer
puntajes van en un grupo plano `/administrations/:id`, **sin `project_id`**.

```ts
export const administrationsApi = {
  start: (
    projectId: string, participantId: string, instrumentId: string,
    clientGeneratedId?: string,
  ) =>
    api.post<AdministrationResponse>(
      `/projects/${projectId}/participants/${participantId}/instruments/${instrumentId}/administrations`,
      { client_generated_id: clientGeneratedId ?? crypto.randomUUID() },
    ),

  list: (projectId: string, participantId: string, instrumentId: string) =>
    api.get<AdministrationResponse[]>(
      `/projects/${projectId}/participants/${participantId}/instruments/${instrumentId}/administrations`,
    ),

  // Rutas planas: NO llevan projectId
  complete: (administrationId: string) =>
    api.patch<CompleteAdministrationResponse>(`/administrations/${administrationId}/complete`),

  abandon: (administrationId: string) =>
    api.patch<AdministrationResponse>(`/administrations/${administrationId}/abandon`),

  scores: (administrationId: string) =>
    api.get<ScoreResponse[]>(`/administrations/${administrationId}/scores`),

  recompute: (administrationId: string) =>
    api.post<ScoreResponse[]>(`/administrations/${administrationId}/scores/recompute`),
};
```

## El flujo del runner, completo

```ts
async function runInstrument(
  projectId: string, participantId: string, instrumentId: string,
) {
  // 1 ─ Iniciar. Congela instrument_version.
  //     ⚠️ Falla con 404 si el instrumento no tiene catálogo publicado.
  const admin = await administrationsApi.start(projectId, participantId, instrumentId);

  // 2 ─ Pedir el catálogo de ESA versión, no la última.
  const catalog = await scoringApi.getCatalog(
    projectId, instrumentId, admin.instrument_version,
  );

  // 3 ─ Renderizar los ítems en orden y recoger las respuestas.
  const items = [...catalog.items].sort((a, b) => a.order_index - b.order_index);
  //    (una llamada POST .../answers por ítem — ver 09_answers.md)

  // 4 ─ Completar: dispara el cálculo y devuelve los puntajes.
  const { administration, scores } = await administrationsApi.complete(admin.id);
  return { administration, scores };
}
```

### Paso 1: el error más común

```ts
try {
  const admin = await administrationsApi.start(projectId, participantId, instrumentId);
} catch (e) {
  if (e instanceof ApiError && e.status === 404) {
    // Casi siempre: el instrumento no tiene catálogo de puntuación publicado.
    showError("Este instrumento todavía no tiene su puntuación configurada.");
  }
}
```

Mejor prevenirlo: no habilitar el botón "Aplicar" si
`listVersions(...)` devuelve `[]`. Ver
[06_instruments.md](06_instruments.md#version-significa-otra-cosa-de-lo-que-parece).

### `client_generated_id`: idempotencia

Generarlo **en el cliente** y reusarlo si hay que reintentar. El backend tiene un
`UNIQUE (project_id, participant_id, client_generated_id)`, así que un reintento
tras un corte de red no crea un segundo intento.

⚠️ **Pero el choque sale como `500`, no como `409`**: la violación de unicidad no
se traduce a un error de dominio. Un reintento legítimo es indistinguible de una
caída del servidor. Hay que resolverlo listando:

```ts
async function startOrResume(
  projectId: string, participantId: string, instrumentId: string, localId: string,
) {
  try {
    return await administrationsApi.start(projectId, participantId, instrumentId, localId);
  } catch (e) {
    if (!(e instanceof ApiError) || e.status !== 500) throw e;
    // Puede ser el UNIQUE: comprobar si el intento ya se había creado.
    const existing = (await administrationsApi.list(projectId, participantId, instrumentId))
      .find(a => a.client_generated_id === localId);
    if (existing) return existing;      // era un reintento: seguir con ese
    throw e;                            // era un error real
  }
}
```

```ts
// Generar una vez por sesión de runner y persistirlo localmente
const [localId] = useState(() => crypto.randomUUID());
```

Omitirlo es válido (`null`), pero entonces un reintento **sí** crea un intento
duplicado. Siempre mandarlo.

### Paso 4: completar

```
PATCH /administrations/:id/complete
```

Encadena: catálogo de la versión congelada → respuestas de este intento → motor
de puntuación → guardado. Devuelve la aplicación **y** los puntajes juntos, así
que no hace falta un `GET /scores` después.

Responde `409` si la aplicación ya estaba `abandoned`.

⚠️ **El backend marca `completed` antes de calcular.** Si el cálculo falla, la
aplicación queda cerrada y sin puntajes: un `500` en `complete` deja la
aplicación en un estado del que no se puede volver a `in_progress`. Lo único
disponible es `recompute`.

## Estados

```
in_progress ──> completed    (calcula puntajes)
            └─> abandoned    (no calcula nada)
```

| Transición | Validada por el backend |
|---|---|
| `abandoned` → `completed` | ✅ rechaza con `409` |
| `completed` → `abandoned` | ❌ **lo permite** |
| `completed` → `completed` | ❌ lo permite: recalcula y sobrescribe |

⚠️ Abandonar una aplicación ya completada deja los puntajes calculados colgando
de un intento `abandoned` — un estado incoherente que el backend no impide
(G-15). La UI debe bloquear esa acción:

```ts
const canAbandon  = admin.status === "in_progress";
const canComplete = admin.status === "in_progress";
```

## Leer e interpretar los puntajes

```ts
const scores = await administrationsApi.scores(administrationId);  // solo lee, no recalcula

// El puntaje total es el de subscale_key === null
const total     = scores.find(s => s.subscale_key === null);
const subscales = scores.filter(s => s.subscale_key !== null);
```

### `raw_score: null` tiene dos causas muy distintas

```ts
function describeScore(s: ScoreResponse): string {
  if (s.raw_score !== null)
    return `${s.raw_score}${s.band_label ? ` — ${s.band_label}` : ""}`;

  // Causa legítima: faltan respuestas para que el puntaje sea interpretable
  if (s.items_answered < s.items_expected)
    return `Incompleto (${s.items_answered} de ${s.items_expected} ítems)`;

  // Causa sospechosa: respondió todo y aun así no hay puntaje.
  // Casi siempre es el envoltorio de "value" mal formado — ver 09_answers.md, G-01.
  return "Sin puntaje calculable";
}
```

⚠️ **Si `items_answered` es 0 pero el participante respondió todo, el problema es
el envoltorio de `value`.** El motor no reconoce las respuestas y las cuenta
como ausentes. Es el síntoma de G-01 y la causa más probable de un puntaje nulo
inesperado. Ver [09_answers.md](09_answers.md).

Mostrar siempre `items_answered / items_expected` en la UI de resultados: es el
único indicador que delata este problema.

### Visualizar el baremo

`band_label` es la interpretación ya resuelta por el backend (`"Moderada"`). Para
ubicar el puntaje dentro de la escala hace falta el catálogo:

```ts
const catalog = await scoringApi.getCatalog(projectId, instrumentId, admin.instrument_version);
const bands = catalog.bands
  .filter(b => b.subscale_key === score.subscale_key)
  .sort((a, b) => (a.min_score ?? -Infinity) - (b.min_score ?? -Infinity));
// bands da los rangos para pintar una barra con la posición del puntaje
```

## Historial de intentos

```ts
const attempts = await administrationsApi.list(projectId, participantId, instrumentId);
```

Viene ordenado por `created_at DESC` (el más reciente primero). Útil para
pretest/postest:

```ts
const completed = attempts
  .filter(a => a.status === "completed")
  .sort((a, b) => a.started_at.localeCompare(b.started_at));   // cronológico
```

⚠️ **Hace falta un listado por instrumento**: no hay endpoint que devuelva todas
las aplicaciones de un participante de una vez, ni todas las de un proyecto. Para
una tabla "participantes × instrumentos" hay que hacer N×M llamadas. En un padrón
grande eso no escala; vale la pena pedirle al backend un endpoint agregado.

```ts
// N×M llamadas — aceptable para pocos instrumentos, no para un padrón grande
const matrix = await Promise.all(
  participants.flatMap(p =>
    instruments.map(async i => ({
      participant: p.id, instrument: i.id,
      attempts: await administrationsApi.list(projectId, p.id, i.id),
    })),
  ),
);
```

## `recompute`: hoy no sirve de nada

```
POST /administrations/:id/scores/recompute
```

Recalcula sobre **las mismas respuestas** y **el mismo catálogo congelado**, así
que siempre devuelve lo mismo. Solo tendrá efecto cuando exista edición de
respuestas, que hoy no está implementada (G-07).

**Recomendación: no exponerlo en la UI todavía.** Un botón "Recalcular" que no
cambia nada genera desconfianza en los datos.

## Nada de esto marca al participante como completado

Completar todas las aplicaciones de un participante **no** cambia su `status`.
Hay que hacerlo explícitamente:

```ts
await participantsApi.updateStatus(projectId, participantId, "completed");
```

Ver [05_participants.md](05_participants.md).

## Validación en el cliente

| Campo | Regla |
|---|---|
| `client_generated_id` | opcional, máx. 128. **Mandarlo siempre** |

El body de `complete`, `abandon` y `recompute` va vacío.

Lo que el backend **no** valida y conviene verificar en el cliente:

- que el participante pertenezca al proyecto;
- que el instrumento pertenezca al proyecto;
- que no haya ya una aplicación `in_progress` del mismo par
  participante/instrumento — **se pueden abrir intentos concurrentes**:

```ts
const attempts = await administrationsApi.list(projectId, participantId, instrumentId);
const open = attempts.find(a => a.status === "in_progress");
if (open) {
  // Ofrecer retomar ese intento en vez de abrir uno nuevo
  return resumeAdministration(open);
}
```

## Retomar un intento abierto: con un límite

Se puede detectar el intento abierto y volver a su formulario, pero **no se
pueden recuperar las respuestas ya enviadas de ese intento**: no existe un
endpoint que liste respuestas por aplicación (G-09). Lo más cercano es filtrar
las del participante por instrumento y cruzar a mano:

```ts
// Aproximación: trae las del participante para ese instrumento,
// pero NO distingue entre intentos distintos del mismo instrumento.
const answers = await answersApi.listByParticipant(projectId, participantId, instrumentId);
```

Con un solo intento por instrumento funciona; con pretest y postest, las
respuestas de ambos vienen mezcladas y no hay forma de separarlas desde el
cliente. Es una limitación a tener presente si el diseño del estudio contempla
medidas repetidas.

## Errores

| Código | Endpoint | Significado | Acción en UI |
|---|---|---|---|
| `400` | todos | UUID de ruta malformado | Error de navegación |
| `404` | `start` | **El instrumento no tiene catálogo publicado** | "Falta configurar la puntuación" |
| `404` | resto | La aplicación no existe | "No encontrado" |
| `500` | `start` | ⚠️ `client_generated_id` repetido — **no da `409`** | Probablemente ya existe: listar y retomar |
| `409` | `complete` | La aplicación está `abandoned` | Informar; no se puede completar |
| `500` | `complete` | Falló el cálculo (algoritmo desconocido, `value_map` malformado) | La aplicación quedó `completed` sin puntajes: avisar y revisar el catálogo |

## Trampas

1. **`start` falla con `404` si no hay catálogo publicado.** Es el error más
   común al montar un proyecto nuevo. Prevenirlo con `listVersions`.
2. **Usar `instrument_version` de la aplicación** para pedir el catálogo, nunca
   "la última".
3. **Las rutas cambian de forma**: `start`/`list` anidadas, el resto planas.
4. **`complete` ya devuelve los puntajes**: no hace falta un `GET /scores` extra.
5. **`raw_score: null` con `items_answered: 0` y el usuario habiendo respondido
   todo = envoltorio de `value` mal formado** (G-01). Mostrar siempre
   `items_answered / items_expected`.
6. **`scaled_score`, `computation_version` y `metadata` nunca se llenan.**
7. **Se puede abandonar una aplicación ya completada**: bloquearlo en la UI.
8. **Se pueden abrir intentos concurrentes**: comprobar `in_progress` antes de
   iniciar.
9. **`recompute` no cambia nada hoy**: no exponerlo.
10. **No se puede recuperar las respuestas de un intento concreto** (G-09): el
    "retomar formulario" es parcial, y no funciona con medidas repetidas.
11. **No hay vista agregada**: una tabla participantes × instrumentos cuesta N×M
    llamadas.
12. **Completar no marca al participante como `completed`**: es otra llamada.
13. **`AdministrationResponse` no trae `created_at`/`updated_at`**: para ordenar
    cronológicamente usar `started_at`.
