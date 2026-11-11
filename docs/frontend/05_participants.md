# 05 · Participantes y códigos

## Pantallas

Padrón de participantes del proyecto · alta individual · búsqueda por código ·
cambio de estado.

El **código** de 8 caracteres es la pieza central: identifica al sujeto sin
exponer sus datos personales, y es lo que se dicta o transcribe en campo.

## Tipos

```ts
interface CreateParticipantRequest {
  display_name: string;          // 2-160, requerido
  external_identifier?: string;  // opcional, máx. 160
}

interface UpdateParticipantStatusRequest { status: ParticipantStatus }

interface ParticipantResponse {
  id: string;
  project_id: string;
  code: string;                      // generado por el backend, 8 caracteres
  display_name: string;
  external_identifier?: string;      // ⚠️ AUSENTE del JSON si está vacío
  status: ParticipantStatus;
  created_at: string;
  updated_at: string;
  // Sin last_access_at: no se expone (y el backend nunca lo escribe).
}
```

⚠️ `external_identifier` tiene `omitempty`: cuando no se cargó, **la clave no
aparece** en el JSON. En TypeScript es `?:` (opcional), no `| null`. Usar
`p.external_identifier ?? "—"` al renderizar.

## Endpoints

| Método | Ruta | Respuesta |
|---|---|---|
| `POST` | `/projects/:id/participants` | `201 ParticipantResponse` |
| `GET` | `/projects/:id/participants` | `200 ParticipantResponse[]` |
| `GET` | `/projects/:id/participants/code/:code` | `200 ParticipantResponse` |
| `PATCH` | `/projects/:id/participants/:participantID/status` | `200 ParticipantResponse` |

```ts
export const participantsApi = {
  create: (projectId: string, body: CreateParticipantRequest) =>
    api.post<ParticipantResponse>(`/projects/${projectId}/participants`, body),

  list: (projectId: string) =>
    api.get<ParticipantResponse[]>(`/projects/${projectId}/participants`),

  byCode: (projectId: string, code: string) =>
    api.get<ParticipantResponse>(
      `/projects/${projectId}/participants/code/${encodeURIComponent(code.trim())}`,
    ),

  updateStatus: (projectId: string, participantId: string, status: ParticipantStatus) =>
    api.patch<ParticipantResponse>(
      `/projects/${projectId}/participants/${participantId}/status`,
      { status },
    ),
};
```

## El código

Lo genera el backend; **el cliente no lo elige ni lo propone**.

- 8 caracteres del alfabeto `ABCDEFGHJKLMNPQRSTUVWXYZ23456789`.
- Sin `I`, `O`, `0` ni `1`: se confunden al dictar o transcribir a mano.
- Único por proyecto.

Para mostrarlo, conviene agrupar los caracteres y usar fuente monoespaciada:

```tsx
// MXSK2EBN → MXSK 2EBN
<code className="font-mono tracking-wider">
  {participant.code.replace(/(.{4})/, "$1 ")}
</code>
```

La búsqueda por código **normaliza a mayúsculas y recorta espacios en el
backend**, así que se puede teclear en minúsculas. Pero si la UI muestra el
código con espacios, hay que quitarlos antes de enviar:

```ts
const clean = input.replace(/\s+/g, "").toUpperCase();
await participantsApi.byCode(projectId, clean);
```

⚠️ Un código inexistente devuelve `404`, no un array vacío. Manejarlo como
"no encontrado", no como "sin resultados".

## Estados

```
pending ──> active ──> completed
                   └─> expired
```

| Estado | Significado práctico |
|---|---|
| `pending` | Creado, todavía no empezó |
| `active` | En curso |
| `completed` | Terminó su participación |
| `expired` | Venció sin completar |

⚠️ **Ningún otro módulo mueve este estado.** Iniciar o completar una aplicación
de instrumento **no** cambia al participante de `pending`. Si la UI quiere
reflejar el avance real, tiene que llamar al `PATCH` explícitamente:

```ts
// Al iniciar la primera aplicación del participante
await participantsApi.updateStatus(projectId, participantId, "active");
// Al terminar todos los instrumentos previstos
await participantsApi.updateStatus(projectId, participantId, "completed");
```

Y como el backend no valida transiciones (G-15), conviene imponerlas en el
cliente:

```ts
const ALLOWED: Record<ParticipantStatus, ParticipantStatus[]> = {
  pending:   ["active", "expired"],
  active:    ["completed", "expired"],
  completed: [],
  expired:   ["active"],
};
```

## Padrón: paginar y buscar en el cliente

`GET /projects/:id/participants` devuelve **todos** los participantes de una
vez: no hay `page`, `limit`, `q` ni `status`. En un estudio con cientos de
sujetos, la respuesta es grande y el filtrado es tarea del cliente.

```ts
const all = await participantsApi.list(projectId);     // una sola vez, cachear

const filtered = all.filter(p =>
  (!statusFilter || p.status === statusFilter) &&
  (!query ||
    p.display_name.toLowerCase().includes(query.toLowerCase()) ||
    p.code.includes(query.toUpperCase()) ||
    (p.external_identifier ?? "").includes(query)),
);
```

El orden que devuelve el backend es por `created_at DESC` (los más nuevos
primero). Si la UI necesita otro orden, ordenar en el cliente.

## Carga masiva: no existe

`docs/1_MAIN_FEATURES.md` preveía importación por CSV, pero **no hay endpoint**.
Para cargar un padrón hay que llamar al `POST` una vez por participante. Si la
UI ofrece "importar CSV", tiene que serializar las llamadas en el cliente:

```ts
// Secuencial a propósito: evita golpear la API y permite informar el avance.
async function importRows(projectId: string, rows: CreateParticipantRequest[]) {
  const created: ParticipantResponse[] = [];
  const failed: { row: CreateParticipantRequest; error: string }[] = [];

  for (const row of rows) {
    try {
      created.push(await participantsApi.create(projectId, row));
    } catch (e) {
      failed.push({ row, error: e instanceof ApiError ? e.message : "error" });
    }
  }
  return { created, failed };
}
```

Importa ser tolerante a fallos parciales: un `external_identifier` repetido hace
fallar **esa** fila, no el lote. ⚠️ Pero el error llega como **`500`**, no `409`:
la violación de unicidad no se traduce. Conviene filtrar duplicados contra el
padrón ya cargado antes de empezar, para distinguir "repetido" de "se cayó la
API".

## Validación en el cliente

| Campo | Regla |
|---|---|
| `display_name` | requerido, 2-160 caracteres |
| `external_identifier` | opcional, máximo 160 |
| `status` | requerido; uno de los cuatro valores del enum |

`external_identifier` debe ser **único dentro del proyecto**. Conviene
verificarlo contra el padrón ya cargado antes de enviar, para dar un error en el
campo en vez de un `500` indescifrable:

```ts
const duplicate = all.some(p => p.external_identifier === value);
```

⚠️ Esta comprobación no es opcional: si se omite, el usuario recibe un `500`
genérico y no hay forma de saber que la causa fue un identificador repetido.

## Errores

| Código | Significado | Acción en UI |
|---|---|---|
| `400` | UUID de ruta malformado | Error de navegación |
| `404` | Código o participante inexistente | "No encontrado" |
| `500` | ⚠️ `external_identifier` duplicado — **no da `409`**, sale como error de servidor | Comprobar duplicados antes de enviar |
| `422` | `display_name` fuera de rango, `status` inválido | Resaltar el campo |

## Trampas

1. **`external_identifier` desaparece del JSON cuando está vacío**: es opcional,
   no `null`.
2. **El código lo genera el backend**: no se puede proponer ni editar.
3. **Buscar por código devuelve `404`**, no una lista vacía.
4. **El estado no avanza solo.** Las aplicaciones de instrumento no lo mueven;
   hay que hacerlo con el `PATCH`.
5. **Sin transiciones validadas**: `completed` puede volver a `pending`.
6. **Sin paginación, búsqueda ni filtros del lado del servidor.** Todo en el
   cliente, y la respuesta trae el padrón completo.
7. **Sin carga masiva**: una llamada por participante.
8. **No hay borrado ni edición.** No se puede corregir un `display_name` mal
   escrito ni eliminar un participante cargado por error: solo pasarlo a
   `expired`. Vale la pena advertirlo en el formulario de alta.
9. **El código no habilita nada todavía.** No hay portal del participante ni
   token de acceso propio: todo el flujo pasa por un investigador autenticado.
   El código es, por ahora, una etiqueta para uso del equipo (G-12).
