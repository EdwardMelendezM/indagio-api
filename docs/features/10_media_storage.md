# 10 · Media y almacenamiento (Cloudflare R2)

## Propósito

Entregar URLs prefirmadas para que el cliente suba archivos **directamente al
bucket**, sin que los bytes pasen por el proceso Go. En Postgres solo queda la
referencia y la metadata.

⚠️ **Este módulo es herencia de otro producto** y hoy no sirve al flujo de
Indagio. Los `target_type` que acepta (`avatar`, `thread`, `comment`, `chat`) y
los DTOs de stickers vienen de una aplicación social. Falta lo que el flujo de
investigación necesita: audio, y un presign ligado a `project_id` /
`participant_id`. Ver
[G-10](00_GAPS.md#g-10--el-módulo-media-es-de-otro-producto).

## El patrón de subida

```
1. Cliente  → POST /api/media/presign            → { upload_url, file_url }
2. Cliente  → PUT  <upload_url>  (bytes a R2)    sin pasar por la API
3. Cliente  → POST /projects/:id/participants/:pid/media   guarda la metadata
```

Correcto en su diseño: archivos pesados de entrevista no deben atravesar el
servidor. El paso 3 es de [09_answers](09_answers.md), no de este módulo.

El eslabón débil es que **nada ata el paso 3 al paso 1**: el `file_key` que el
cliente declara al registrar la metadata no se verifica contra lo que realmente
se subió, ni contra el `key` que el servidor firmó.

## Endpoints

| Método | Ruta | Auth | Descripción |
|---|---|---|---|
| `POST` | `/api/media/presign` | Bearer | URL prefirmada genérica |
| `POST` | `/api/media/presigned-video` | Bearer | URL `PUT` para video, con topes |

`media_handler.go:141-145`.

## Contratos

```jsonc
// POST /api/media/presign
{ "target_type": "avatar",      // oneof: avatar | thread | comment | chat
  "content_type": "image/jpeg", // oneof: image/jpeg | image/png | image/webp | video/mp4
  "extension": "webp" }
→ { "upload_url": "...", "file_url": "https://<public_domain>/<key>" }

// POST /api/media/presigned-video
{ "content_type": "video/mp4", "max_size_bytes": 52428800 }
→ { "upload_url": "...", "raw_key": "videos/raw/<uuid>.mp4",
    "max_size_bytes": 52428800, "max_duration_s": 30 }
```

Ninguno de los dos `content_type` admite audio, pese a que
`InstrumentKindAudio` existe en el dominio.

## Prefijos de clave

`media_handler.go:69-78` — derivados del `target_type`:

| `target_type` | Prefijo |
|---|---|
| `avatar` | `users/<userID>/avatar` |
| `thread` | `threads/pending-<userID>/images` |
| `comment` | `comments/pending-<userID>` |
| `chat` | `chat/pending-<userID>` |

El `switch` no tiene `default`: un `target_type` fuera de la lista dejaría el
prefijo vacío y la clave como `/<uuid>.<ext>`. Hoy no ocurre porque el binding
`oneof` lo rechaza antes, pero es una dependencia frágil entre validación y
lógica.

Para video, la clave es `videos/raw/<uuid>.mp4` y la URL vence a los 15 minutos.

## Topes de video

Configurables por entorno (`config.go:114-122`):

| Variable | Default |
|---|---|
| `VIDEO_MAX_SIZE_MB` | 200 |
| `VIDEO_MAX_DURATION_S` | 30 |

El servidor **acota** lo que pide el cliente: si pide más que el tope, se
devuelve el tope del servidor como cifra autoritativa (`media_handler.go:117-120`).

El tope de tamaño no se puede imponer de verdad — una URL prefirmada de R2 no
limita bytes — así que es una indicación al cliente. La duración tampoco se
verifica: el comentario del DTO menciona un worker que rechazaría videos largos,
pero ese worker no existe en este repositorio.

## Interfaz de almacenamiento

`domain.StorageRepository` (`domain/storage.go`) abstrae el proveedor:
`UploadFile`, `DeleteFile`, `GetPresignedUploadURL`, `GetPresignedPutURL`,
`MoveObject`, `DownloadFile`, `DownloadFileBytes`, `AvatarVariantURL`,
`UploadDirectory`.

Implementación única: `repository/storage/cloudflare` (R2 vía SDK de AWS S3).
Que sea una interfaz permite sustituir el proveedor o usar MinIO en pruebas,
como preveía `docs/1_MAIN_FEATURES.md` §8.

`DownloadFileBytes` carga el objeto entero en memoria; su propio comentario
advierte de no usarlo con video.

## Autorización

| | |
|---|---|
| **Implementada** | Bearer en ambas rutas; el `userID` del token determina el prefijo de la clave en `/presign`. |
| **Esperada** | Para media de respuesta: miembro del proyecto, y clave bajo un prefijo del proyecto y del participante (p. ej. `projects/<pid>/participants/<id>/…`). |

`/presigned-video` **no usa el `userID` para nada**: la clave es
`videos/raw/<uuid>.mp4`, sin espacio de nombres por usuario ni por proyecto.
Cualquier usuario autenticado puede obtener una URL de escritura en ese prefijo
compartido.

## Dependencias

- **Requiere:** [01_auth](01_auth.md); R2 configurado (`ACCOUNT_ID`,
  `ACCESS_KEY_ID`, `SECRET_ACCESS_KEY`, `BUCKET_NAME`, `PUBLIC_DOMAIN`).
- **Lo consume:** [09_answers](09_answers.md) (`media_files`),
  [12_exports](12_exports.md) cuando exista (`project_exports.file_key`).

## Gaps conocidos

- **[G-10](00_GAPS.md#g-10--el-módulo-media-es-de-otro-producto)** — los
  `target_type` y los DTOs de stickers son de otro producto; falta audio y falta
  presign con alcance de proyecto/participante.
- `file_url` se construye sobre `PUBLIC_DOMAIN`, es decir, **acceso público
  directo al objeto**. `docs/1_MAIN_FEATURES.md` §8 exige explícitamente lo
  contrario: URLs firmadas de corta vida y nunca acceso público a archivos de
  entrevista. Para media de respuesta hace falta un endpoint de lectura firmada.
- `media_files.status` nunca sale de `pending`: no hay confirmación de subida.
- `file_key` lo declara el cliente y no se verifica contra R2 ni contra la clave
  firmada.
- Los topes de video son indicativos; el worker de validación no existe.
- Sin thumbnails, normalización de audio ni transcripción
  (`docs/1_MAIN_FEATURES.md` §6).
- Sin política de retención ni borrado — relevante con comité de ética.
