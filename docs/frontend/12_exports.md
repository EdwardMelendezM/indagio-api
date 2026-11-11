# 12 · Exportación de datos

## Pantallas

Panel de descargas del proyecto · solicitud de export · seguimiento de estado.

⚠️ **Solo existe la mitad del flujo.** Se puede solicitar un export —responde
`202 Accepted`, prometiendo un procesamiento asíncrono— pero **nada lo
procesa**: queda en `pending` para siempre, `file_key` nunca se llena y no hay
nada que descargar (G-05).

Antes de construir esta pantalla conviene acordar con backend el alcance real.
Lo que sigue documenta lo que hay.

## Tipos

```ts
interface RequestExportRequest {
  format: ExportFormat;        // "csv" | "json" | "zip" — requerido
  options?: unknown;           // ⚠️ SIN ESQUEMA DEFINIDO
}

interface ExportResponse {
  id: string;
  project_id: string;
  created_by: string;
  format: ExportFormat;
  status: ExportStatus;        // siempre "pending" hoy
  file_key?: string;           // ausente mientras no haya archivo → siempre
  error_log?: string;          // ausente si no hay error → siempre
  created_at: string;
  updated_at: string;
  completed_at?: string;       // ausente → siempre
  // Sin started_at: existe en la tabla pero no se expone.
}
```

Los cuatro campos opcionales tienen `omitempty`: **desaparecen del JSON** cuando
están vacíos. En TypeScript son `?:`, no `| null`. Y como nada procesa los
exports, hoy **nunca** aparecen.

### `options` no tiene contrato

Solo se valida como JSON. Queda por definir con backend, y conviene acordarlo
**antes** de implementar el procesamiento, porque determina las columnas de
salida:

```ts
// PROPUESTA — no implementada
interface ExportOptions {
  instrument_ids?: string[];      // qué instrumentos incluir
  date_from?: string;             // rango de fechas
  date_to?: string;
  include_scores?: boolean;       // puntajes además de respuestas crudas
  include_media?: boolean;        // solo aplica a zip
  anonymize?: boolean;            // ⚠️ decisión de diseño, no solo de formato
}
```

El punto de `anonymize` no es cosmético: `docs/1_MAIN_FEATURES.md` §12 pide
separar el código identificable de los datos de respuesta. Si el export incluye
`display_name` y `external_identifier`, deja de ser anónimo — relevante si hay
comité de ética involucrado.

## Endpoints

| Método | Ruta | Respuesta |
|---|---|---|
| `POST` | `/projects/:id/exports` | `202 ExportResponse` |
| `GET` | `/projects/:id/exports` | `200 ExportResponse[]` |

```ts
export const exportsApi = {
  request: (projectId: string, format: ExportFormat, options?: unknown) =>
    api.post<ExportResponse>(`/projects/${projectId}/exports`, { format, options }),

  list: (projectId: string) =>
    api.get<ExportResponse[]>(`/projects/${projectId}/exports`),
};
```

**No existe `GET /projects/:id/exports/:exportId`.** El backend tiene `GetExport`
implementado pero sin ruta: no se puede consultar un export puntual ni —cuando
exista— obtener su enlace de descarga. Para seguir el estado hay que listar todos
y filtrar:

```ts
const mine = (await exportsApi.list(projectId)).find(e => e.id === exportId);
```

## El polling que no termina nunca

El patrón natural sería solicitar y hacer polling hasta `completed`. Hoy eso es
un bucle infinito, porque el estado jamás cambia.

```ts
// ⚠️ Con el backend actual esto nunca termina: el status se queda en "pending".
async function pollExport(projectId: string, exportId: string, signal: AbortSignal) {
  const TERMINAL: ExportStatus[] = ["completed", "failed"];
  while (!signal.aborted) {
    const item = (await exportsApi.list(projectId)).find(e => e.id === exportId);
    if (item && TERMINAL.includes(item.status)) return item;
    await new Promise(r => setTimeout(r, 3000));
  }
}
```

Si la pantalla se implementa igual, **ponerle un límite de intentos y un mensaje
honesto** en vez de dejar un spinner eterno:

```ts
const MAX_ATTEMPTS = 20;   // ~1 minuto

if (attempts >= MAX_ATTEMPTS && item?.status === "pending") {
  showWarning(
    "La exportación sigue en cola. El procesamiento automático todavía no " +
    "está disponible: avisá al equipo técnico para obtener los datos.",
  );
}
```

Es preferible a prometer una descarga que no va a llegar.

## Qué mostrar según el estado

```ts
const LABEL: Record<ExportStatus, string> = {
  pending:    "En cola",
  processing: "Generando…",
  completed:  "Listo",
  failed:     "Falló",
};

function exportActions(item: ExportResponse) {
  switch (item.status) {
    case "completed":
      // ⚠️ file_key NO es una URL descargable: es la clave del objeto en R2,
      // y no hay endpoint que devuelva una URL firmada. Estado inalcanzable hoy.
      return { canDownload: Boolean(item.file_key), downloadKey: item.file_key };
    case "failed":
      return { canRetry: true, reason: item.error_log };
    default:
      return { waiting: true };
  }
}
```

⚠️ **`file_key` no se puede usar como URL.** Es la ruta del objeto dentro del
bucket. Para descargarlo hace falta una URL firmada que ningún endpoint entrega
—ver [10_media.md](10_media.md), que tiene el mismo problema. Falta ese endpoint
tanto como el procesamiento.

## Alternativa mientras no exista el procesamiento

Si la necesidad de exportar es inmediata, se puede armar un CSV **en el cliente**
con los endpoints que sí funcionan. Sirve para proyectos chicos.

```ts
async function exportInClient(
  projectId: string, participants: ParticipantResponse[], instruments: InstrumentResponse[],
) {
  const rows: Record<string, unknown>[] = [];

  for (const p of participants) {
    const answers = await answersApi.listByParticipant(projectId, p.id);
    for (const a of answers) {
      rows.push({
        codigo: p.code,
        identificador: p.external_identifier ?? "",
        instrumento: a.instrument?.name ?? "",
        pregunta: a.question_key,
        // El valor real está dentro del envoltorio {value: X}
        valor: (a.value as { value?: unknown })?.value ?? "",
        fecha: a.created_at,
      });
    }
  }
  return toCsv(rows);
}
```

Limitaciones a tener en cuenta:

- **Una llamada por participante**: con cientos de sujetos es lento y pesado.
- **Los puntajes requieren más llamadas**: hay que recorrer aplicaciones por
  participante × instrumento ([08_administrations.md](08_administrations.md)).
- **No distingue intentos** del mismo instrumento (G-09): pretest y postest
  vienen mezclados.
- **No incluye media**: los archivos no son accesibles.
- **Hay que decidir la anonimización en el cliente**: si se incluye `codigo` y
  `external_identifier`, el archivo contiene datos identificables y hay que
  tratarlo como tal.

## Validación en el cliente

| Campo | Regla |
|---|---|
| `format` | requerido; `csv`\|`json`\|`zip`. Se normaliza a minúsculas en el backend |
| `options` | opcional, JSON válido |

Mandar un `format` fuera de la lista devuelve `422`. `CSV` en mayúsculas sí
funciona: el backend lo pasa a minúsculas antes de validar.

## Errores

| Código | Significado | Acción en UI |
|---|---|---|
| `400` | `:id` no es UUID | Error de navegación |
| `422` | `format` inválido u `options` mal formado | Resaltar el campo |
| `500` | Error interno | Mensaje genérico |

Nótese que **no hay `403`**: este módulo no verifica pertenencia al proyecto
(G-02). Un `project_id` ajeno devuelve `202` y crea un export contra el proyecto
de otro equipo. Es el caso de mayor impacto del gap, porque un export pretende
sacar el conjunto completo de datos de un estudio. Validar el `project_id` en el
cliente.

## Trampas

1. **Nada procesa los exports** (G-05): el estado queda `pending` para siempre.
   No prometer descargas en la UI.
2. **No hay `GET` individual**: filtrar del listado.
3. **No hay endpoint de descarga**, y `file_key` no es una URL.
4. **`options` no tiene esquema**: nada que mandar todavía.
5. **Los campos opcionales desaparecen del JSON**: usar `?:` en los tipos.
6. **`started_at` no se expone**, aunque esté en la tabla.
7. **Sin verificación de proyecto** (G-02), y aquí el impacto es máximo.
8. **La anonimización no está resuelta**: es una decisión de diseño pendiente,
   no un detalle de formato.
9. **Un polling ingenuo no termina nunca**: poner límite y mensaje honesto.
