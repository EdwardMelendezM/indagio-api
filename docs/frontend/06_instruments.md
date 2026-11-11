# 06 · Instrumentos

## Pantallas

Lista de instrumentos del proyecto · creación · edición de nombre, presentación
y estado.

Un instrumento tiene **dos mitades** y se administran por separado:

| Mitad | Qué define | Dónde |
|---|---|---|
| Identidad y presentación | nombre, tipo, etiquetas de respuesta | este doc |
| Puntuación | ítems, subescalas, algoritmo, baremos | [07_scoring_catalog.md](07_scoring_catalog.md) |

Crear el instrumento **no alcanza** para poder aplicarlo: hace falta además
publicar un catálogo de puntuación. Es el error más común al montar un proyecto.

## Tipos

```ts
interface CreateInstrumentRequest {
  name: string;                  // 2-160, requerido
  kind: InstrumentKind;          // requerido
  config?: unknown;              // JSON libre; no se valida contra "kind"
}

interface UpdateInstrumentRequest {
  name?: string;
  config?: unknown;
  status?: InstrumentStatus;
}

interface InstrumentResponse {
  id: string;
  project_id: string;
  name: string;
  kind: InstrumentKind;
  config: unknown;               // tal cual se guardó
  version: number;               // ⚠️ versión del CATÁLOGO, no del instrumento
  status: InstrumentStatus;
  created_by: string;
  created_at: string;
  updated_at: string;
}
```

### `config` no tiene esquema

El backend solo verifica que sea JSON válido: **no valida su forma contra el
`kind`**. Un `single_choice` sin opciones se acepta sin objeción. El contrato es
una convención entre frontend y equipo, no algo que la API imponga.

La forma que usan los ejemplos del repositorio:

```ts
interface ChoiceInstrumentConfig {
  description?: string;
  response_labels: string[];     // etiquetas que ve el participante
}

// Ejemplo real (PHQ-8)
{
  description: "Patient Health Questionnaire - 8 items",
  response_labels: [
    "Ningún día", "Varios días",
    "Más de la mitad de los días", "Casi todos los días",
  ],
}
```

⚠️ **`response_labels` es solo presentación.** La traducción de esas etiquetas a
números para puntuar vive en el `value_map` de cada ítem del catálogo, que es
otra estructura y otro endpoint. Son dos cosas que hay que mantener coherentes a
mano: si se agrega una etiqueta aquí, hay que actualizar los `value_map` allá.

Conviene tipar `config` por `kind` en el cliente y validarlo antes de enviar,
porque el backend no lo hará:

```ts
type InstrumentConfig =
  | { kind: "single_choice" | "multi_choice" | "scale"; response_labels: string[]; description?: string }
  | { kind: "text" | "numeric" | "audio" | "video"; description?: string };
```

## Endpoints

| Método | Ruta | Respuesta |
|---|---|---|
| `POST` | `/projects/:id/instruments` | `201 InstrumentResponse` |
| `GET` | `/projects/:id/instruments` | `200 InstrumentResponse[]` |
| `PATCH` | `/projects/:id/instruments/:instrumentID` | `200 InstrumentResponse` |

```ts
export const instrumentsApi = {
  create: (projectId: string, body: CreateInstrumentRequest) =>
    api.post<InstrumentResponse>(`/projects/${projectId}/instruments`, body),

  list: (projectId: string) =>
    api.get<InstrumentResponse[]>(`/projects/${projectId}/instruments`),

  update: (projectId: string, instrumentId: string, body: UpdateInstrumentRequest) =>
    api.patch<InstrumentResponse>(
      `/projects/${projectId}/instruments/${instrumentId}`,
      body,
    ),
};
```

**No hay `GET` de un instrumento puntual ni `DELETE`.** Para mostrar el detalle
hay que tomarlo del listado, y un instrumento creado por error solo se puede
pasar a `archived`.

```ts
// Detalle: filtrar del listado, no hay endpoint directo
const instrument = (await instrumentsApi.list(projectId))
  .find(i => i.id === instrumentId);
```

## `version` significa otra cosa de lo que parece

`InstrumentResponse.version` **no** es la versión del instrumento: es el
contador del **catálogo de puntuación**, y lo mueve
[07_scoring_catalog.md](07_scoring_catalog.md) al publicar.

Un instrumento recién creado llega con `version: 1` **sin tener catálogo
publicado**. Es decir: `version: 1` no significa "tiene catálogo". No se puede
usar ese campo para saber si el instrumento es aplicable.

La única forma confiable de saberlo es preguntar por las versiones publicadas:

```ts
async function isReadyToApply(projectId: string, instrumentId: string) {
  const versions = await scoringApi.listVersions(projectId, instrumentId);
  return versions.length > 0;     // [] = sin catálogo = no se puede aplicar
}
```

Conviene mostrarlo en la lista de instrumentos, porque es la diferencia entre
"se puede usar" y "falla al intentar aplicarlo":

```tsx
{ready
  ? <Badge variant="success">Listo para aplicar</Badge>
  : <Badge variant="warning">Falta publicar la puntuación</Badge>}
```

Hace una llamada por instrumento. En una lista larga, pedirlas en paralelo:

```ts
const readiness = await Promise.all(
  instruments.map(async i => [i.id, (await scoringApi.listVersions(projectId, i.id)).length > 0] as const),
);
```

## Estados

```
draft ──> active ──> archived
```

⚠️ **El `status` no tiene ningún efecto en el backend.** No se consulta en
ninguna parte: un instrumento `draft` o `archived` se puede aplicar igual. Si la
UI quiere que `archived` signifique "no usar", tiene que filtrarlo ella:

```ts
const usable = instruments.filter(i => i.status !== "archived");
```

Tampoco se validan las transiciones (G-15).

## Editar `config` después de recolectar: cuidado

Cambiar `config` **no versiona nada**. El catálogo de puntuación sí es
inmutable y versionado; la presentación no.

Eso significa que editar `response_labels` después de haber recogido respuestas
**cambia retroactivamente cómo se leen los datos ya guardados**: una respuesta
almacenada como `2` seguirá siendo `2`, pero la etiqueta que el investigador ve
al revisarla será la nueva.

Recomendación para la UI: si el instrumento ya tiene aplicaciones, advertir
antes de permitir editar `config`, o bloquear la edición de `response_labels` y
ofrecer solo el cambio de `name` y `description`.

```ts
// Hay aplicaciones si algún participante tiene administrations de este instrumento.
// No hay un endpoint que lo diga directo: hay que preguntarlo por participante
// (ver 08_administrations.md), o asumir que sí en cuanto haya respuestas.
```

## Validación en el cliente

| Campo | Regla |
|---|---|
| `name` | requerido, 2-160 caracteres |
| `kind` | requerido; uno de los 7 valores del enum |
| `config` | opcional; JSON válido. **Su forma no se valida** |
| `status` | opcional; `draft`\|`active`\|`archived` |

`kind` es **inmutable**: `UpdateInstrumentRequest` no lo incluye. Si se eligió
mal, hay que crear otro instrumento. Vale la pena confirmarlo en el formulario
de alta.

## Errores

| Código | Significado | Acción en UI |
|---|---|---|
| `400` | UUID de ruta malformado | Error de navegación |
| `404` | El instrumento no existe | "No encontrado" |
| `422` | `name` fuera de rango, `kind` inválido, `config` mal formado | Resaltar el campo |

## Trampas

1. **Crear el instrumento no basta para aplicarlo**: falta publicar el catálogo
   de puntuación. Sin él, `StartAdministration` falla.
2. **`version` es la versión del catálogo**, y vale `1` incluso sin catálogo
   publicado. No sirve para saber si el instrumento está listo.
3. **`config` no se valida contra `kind`.** La coherencia es responsabilidad del
   cliente.
4. **`response_labels` (presentación) y `value_map` (puntuación) son estructuras
   separadas** que hay que mantener sincronizadas a mano.
5. **`status` no hace nada**: no impide aplicar un instrumento `archived`.
6. **`kind` no se puede cambiar** después de crear.
7. **No hay `GET` individual ni `DELETE`**: detalle desde el listado, y baja
   lógica solo vía `archived`.
8. **Editar `config` afecta retroactivamente** la lectura de respuestas ya
   tomadas.
9. **Los instrumentos son por proyecto**: no hay plantillas reutilizables. Montar
   el PHQ-8 en un proyecto nuevo obliga a recrear instrumento **y** catálogo
   completos. Si la UI puede guardar plantillas en el cliente y reproducir las
   dos llamadas, le ahorra mucho trabajo al investigador.
10. **`GET /projects/:id/instruments` no pagina** (poco probable que moleste: son
    pocos por proyecto).
