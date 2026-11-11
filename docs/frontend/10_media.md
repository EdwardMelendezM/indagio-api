# 10 · Subida de archivos a R2

## Pantallas

Captura o selección de audio/video en el formulario de un ítem · adjuntar
archivos a una respuesta.

⚠️ **Este módulo viene de otro producto y hoy no sirve al flujo de Indagio.**
Los tipos que acepta (`avatar`, `thread`, `comment`, `chat`) son de una
aplicación social. **No admite audio**, y no hay presign con alcance de proyecto
ni de participante. Antes de diseñar una pantalla de captura, conviene acordar
con backend qué va a existir — ver G-10.

Lo que sigue documenta lo que hay, para poder trabajar con video mientras tanto.

## El patrón de tres pasos

```
1. Cliente → POST /media/presigned-video     → { upload_url, raw_key, ... }
2. Cliente → PUT  <upload_url>  (los bytes)  → directo a R2, sin pasar por la API
3. Cliente → POST /projects/:id/participants/:pid/media  → guarda la metadata
```

El paso 2 **no pasa por el servidor Go**: es lo correcto para archivos pesados.
El paso 3 es de [09_answers.md](09_answers.md).

⚠️ **Nada ata el paso 3 al paso 1.** El `file_key` que se declara al registrar la
metadata no se verifica contra lo que realmente se subió ni contra la clave que
el servidor firmó. Hay que llevar el `raw_key` del paso 1 hasta el paso 3 con
cuidado: un error ahí guarda metadata que apunta a la nada.

## Tipos

```ts
// ── Presign genérico ───────────────────────────────────────────────────────
interface PresignUploadRequest {
  target_type: "avatar" | "thread" | "comment" | "chat";   // ⚠️ nada de Indagio
  content_type: "image/jpeg" | "image/png" | "image/webp" | "video/mp4";
  extension: string;             // "webp", "mp4", sin punto
}

interface PresignUploadResponse {
  upload_url: string;            // URL temporal para el PUT
  file_url: string;              // URL pública final — ver nota de privacidad
}

// ── Presign de video (el único usable para respuestas) ─────────────────────
interface PresignedVideoRequest {
  content_type: "video/mp4";     // ⚠️ solo mp4
  max_size_bytes: number;        // requerido, > 0; el servidor lo acota
}

interface PresignedVideoResponse {
  upload_url: string;            // URL PUT firmada, vence en 15 minutos
  raw_key: string;               // "videos/raw/<uuid>.mp4" — llevarlo al paso 3
  max_size_bytes: number;        // el tope efectivo (puede ser menor al pedido)
  max_duration_s: number;        // tope de duración del servidor
}
```

## Endpoints

| Método | Ruta | Respuesta |
|---|---|---|
| `POST` | `/media/presign` | `200 PresignUploadResponse` |
| `POST` | `/media/presigned-video` | `200 PresignedVideoResponse` |

```ts
export const mediaApi = {
  presign: (body: PresignUploadRequest) =>
    api.post<PresignUploadResponse>("/media/presign", body),

  presignVideo: (maxSizeBytes: number) =>
    api.post<PresignedVideoResponse>("/media/presigned-video", {
      content_type: "video/mp4",
      max_size_bytes: maxSizeBytes,
    }),
};
```

## Subir un video, de punta a punta

```ts
async function uploadVideo(
  file: File, projectId: string, participantId: string, answerId?: string,
) {
  // 1 ─ Pedir la URL firmada. El servidor acota el tamaño a su propio tope.
  const { upload_url, raw_key, max_size_bytes, max_duration_s } =
    await mediaApi.presignVideo(file.size);

  // El tope del servidor es la cifra autoritativa: validar contra ella.
  if (file.size > max_size_bytes)
    throw new Error(`El video supera el máximo de ${(max_size_bytes / 1e6).toFixed(0)} MB.`);

  const duration = await readDuration(file);
  if (duration > max_duration_s)
    throw new Error(`El video supera los ${max_duration_s} segundos.`);

  // 2 ─ PUT directo a R2. Sin Authorization: la firma va en la URL.
  const put = await fetch(upload_url, {
    method: "PUT",
    body: file,
    headers: { "Content-Type": "video/mp4" },   // debe coincidir con el content_type firmado
  });
  if (!put.ok) throw new Error(`Falló la subida a R2: ${put.status}`);

  // 3 ─ Registrar la metadata con la MISMA clave del paso 1.
  return answersApi.createMedia(projectId, participantId, {
    answer_id: answerId,
    file_key: raw_key,
    mime_type: "video/mp4",
    size_bytes: file.size,
    duration_seconds: Math.round(duration),
  });
}

function readDuration(file: File): Promise<number> {
  return new Promise((resolve, reject) => {
    const video = document.createElement("video");
    video.preload = "metadata";
    video.onloadedmetadata = () => { resolve(video.duration); URL.revokeObjectURL(video.src); };
    video.onerror = () => reject(new Error("No se pudo leer el video"));
    video.src = URL.createObjectURL(file);
  });
}
```

### Puntos críticos del `PUT`

- **No mandar `Authorization`.** La firma viaja en la URL; agregar el header
  puede hacer que R2 rechace la petición.
- **No usar `credentials: "include"`.** Es otro origen; manda cookies que no
  corresponden.
- **El `Content-Type` debe coincidir** con el que se declaró al firmar. Si se
  firmó `video/mp4` y se sube con otro, la firma no valida.
- **La URL vence en 15 minutos.** Para archivos grandes en conexiones lentas,
  puede caducar a mitad: si falla por expiración, pedir una nueva y reintentar.
- `fetch` no informa progreso de subida. Para una barra de progreso hace falta
  `XMLHttpRequest`:

```ts
function putWithProgress(url: string, file: File, onProgress: (pct: number) => void) {
  return new Promise<void>((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("PUT", url);
    xhr.setRequestHeader("Content-Type", file.type);
    xhr.upload.onprogress = e => e.lengthComputable && onProgress(e.loaded / e.total);
    xhr.onload  = () => (xhr.status >= 200 && xhr.status < 300 ? resolve() : reject(new Error(String(xhr.status))));
    xhr.onerror = () => reject(new Error("error de red"));
    xhr.send(file);
  });
}
```

## Los topes son indicativos, no se imponen

| Variable de entorno | Default |
|---|---|
| `VIDEO_MAX_SIZE_MB` | 200 |
| `VIDEO_MAX_DURATION_S` | 30 |

El servidor **acota** lo pedido: si se piden 500 MB y el tope es 200, devuelve
200 en `max_size_bytes`. Esa es la cifra a respetar.

⚠️ Pero **el tope no se puede imponer de verdad**: una URL prefirmada de R2 no
limita bytes, y la duración no se verifica en ningún lado (el worker que debería
rechazar videos largos no existe en este repositorio). **La validación real es la
del cliente.** Si el cliente no valida, se sube lo que sea.

30 segundos de video por defecto es poco para una entrevista: vale la pena
confirmar con backend si ese valor es el previsto para el caso de uso.

## `/media/presign`: para qué sirve y para qué no

Los `target_type` disponibles no corresponden a nada de Indagio, y el prefijo de
la clave se deriva de ellos:

| `target_type` | Prefijo de la clave |
|---|---|
| `avatar` | `users/<userID>/avatar` |
| `thread` | `threads/pending-<userID>/images` |
| `comment` | `comments/pending-<userID>` |
| `chat` | `chat/pending-<userID>` |

Para media de respuesta **ninguno aplica**. Usarlo igual significaría guardar
archivos de entrevista bajo `chat/pending-<userID>/`, que es confuso y no
agrupable por proyecto.

⚠️ **Además, `file_url` es una URL pública directa** al objeto, construida sobre
`PUBLIC_DOMAIN`. Para datos de investigación eso contradice el propio plan de
arquitectura (`docs/1_MAIN_FEATURES.md` §8: URLs firmadas de corta vida, nunca
acceso público a archivos de entrevista). **No usar `/media/presign` para
contenido de participantes.**

Hasta que exista un presign propio, `/media/presigned-video` es la opción menos
mala: la clave va a `videos/raw/<uuid>.mp4`, sin dato personal en la ruta.

## Audio: no hay camino hoy

`InstrumentKind` incluye `audio`, pero ninguno de los dos endpoints acepta un
`content_type` de audio (`audio/mpeg`, `audio/webm`, `audio/mp4`). Un ítem de
audio **no se puede capturar** con la API actual.

Alternativa temporal, si el caso lo tolera: grabar en `video/mp4` con pista de
audio. No es lo correcto, pero es lo único que el presign admite.

## No hay forma de leer los archivos subidos

No existe endpoint que devuelva una URL de descarga o lectura firmada para un
`media_files`. La respuesta de `createMedia` trae el `file_key`, no una URL
utilizable.

Para reproducir un video de entrevista en la UI hoy habría que construir la URL
pública a mano sobre `PUBLIC_DOMAIN` — lo que implica exponer públicamente datos
sensibles. **Recomendación: no implementar reproducción hasta que exista un
endpoint de lectura firmada.**

## Validación en el cliente

| Campo | Regla |
|---|---|
| `target_type` | requerido; `avatar`\|`thread`\|`comment`\|`chat` |
| `content_type` (presign) | requerido; `image/jpeg`\|`image/png`\|`image/webp`\|`video/mp4` |
| `extension` | requerido, sin punto |
| `content_type` (video) | requerido; **solo** `video/mp4` |
| `max_size_bytes` | requerido, mínimo 1 |

Y la validación que de verdad importa, porque nadie más la hace: tamaño contra
`max_size_bytes` y duración contra `max_duration_s`, **antes** del `PUT`.

## Errores

| Código | Dónde | Significado | Acción en UI |
|---|---|---|---|
| `422` | presign | `content_type` o `target_type` fuera de la lista | Rechazar el archivo antes de pedir la URL |
| `401` | presign | Falta el Bearer | Refresh o login |
| `500` | presign | R2 mal configurado en el servidor | Error genérico; no es del usuario |
| `403` | `PUT` a R2 | Firma vencida o `Content-Type` distinto al firmado | Pedir una URL nueva y reintentar |
| `400` | `PUT` a R2 | Petición mal formada | Revisar headers del `PUT` |

Los errores de R2 vienen en **XML**, no JSON: no intentar parsearlos con
`res.json()`.

```ts
if (!put.ok) {
  const xml = await put.text();        // XML de S3/R2, no JSON
  console.error("R2 rechazó la subida:", put.status, xml);
}
```

## Trampas

1. **Módulo heredado de otro producto**: los `target_type` no corresponden a
   Indagio (G-10).
2. **No hay soporte de audio**, aunque el dominio tenga `InstrumentKind.audio`.
3. **`/media/presign` genera URLs públicas**: no usarlo para datos de
   participantes.
4. **No hay presign con alcance de proyecto/participante**: la clave no agrupa
   por estudio.
5. **No hay endpoint de lectura firmada**: no se puede reproducir lo subido sin
   exponerlo públicamente.
6. **Los topes no se imponen en el servidor**: la validación del cliente es la
   única real.
7. **La URL firmada vence en 15 minutos.**
8. **El `Content-Type` del `PUT` debe coincidir** con el firmado.
9. **Sin `Authorization` ni `credentials` en el `PUT`.**
10. **Los errores de R2 son XML.**
11. **El `file_key` no se valida contra R2** al registrar la metadata, y
    `status` queda `pending` para siempre: no hay confirmación de subida.
12. **`/media/presigned-video` ignora el usuario**: la clave es
    `videos/raw/<uuid>.mp4`, en un prefijo compartido por todos.
