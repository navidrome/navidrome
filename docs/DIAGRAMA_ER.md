# Diagrama Entidad-Relación: Navidrome

## Descripción General

Navidrome es un servidor de música personal con soporte para múltiples usuarios, librerías, playlists, ratings y compartición. El esquema SQLite está optimizado para búsqueda full-text, scrobbling a servicios externos y reproducción streaming.

## Diagrama ER

```mermaid
erDiagram
    USER ||--o{ ANNOTATION : "rating/stars"
    USER ||--o{ PLAYER : "owns"
    USER ||--o{ PLAYQUEUE : "has"
    USER ||--o{ PLAYLIST : "creates"
    USER ||--o{ BOOKMARK : "creates"
    USER ||--o{ USER_PROPS : "has"
    USER ||--o{ SHARE : "creates"
    USER ||--o{ USER_LIBRARY : "accesses"

    ARTIST ||--o{ ALBUM : "releases"
    ARTIST ||--o{ MEDIA_FILE : "performs"
    ARTIST ||--o{ ARTIST_GENRES : "has"
    ARTIST ||--o{ LIBRARY_ARTIST : "belongs_to"

    ALBUM ||--o{ MEDIA_FILE : "contains"
    ALBUM ||--o{ ANNOTATION : "rates"
    ALBUM ||--o{ ALBUM_GENRES : "categorized"
    ALBUM ||--o{ ALBUM_ARTISTS : "has_artists"

    MEDIA_FILE ||--o{ ANNOTATION : "plays"
    MEDIA_FILE ||--o{ PLAYLIST_TRACKS : "part_of"
    MEDIA_FILE ||--o{ BOOKMARK : "bookmarks"
    MEDIA_FILE ||--o{ PLAYQUEUE : "queued"
    MEDIA_FILE ||--o{ MEDIA_FILE_GENRES : "categorized"
    MEDIA_FILE ||--o{ MEDIA_FILE_ARTISTS : "has_artists"

    PLAYLIST ||--o{ PLAYLIST_TRACKS : "contains"
    PLAYLIST_TRACKS ||--o{ MEDIA_FILE : "references"

    GENRE ||--o{ ALBUM_GENRES : "labels"
    GENRE ||--o{ MEDIA_FILE_GENRES : "labels"
    GENRE ||--o{ ARTIST_GENRES : "labels"

    LIBRARY ||--o{ LIBRARY_ARTIST : "contains"
    LIBRARY ||--o{ LIBRARY_TAG : "categorizes"
    LIBRARY ||--o{ USER_LIBRARY : "accessed_by"

    FOLDER ||--o{ MEDIA_FILE : "contains"

    TAG ||--o{ LIBRARY_TAG : "is_library_tag"

    PLAYQUEUE ||--o{ MEDIA_FILE : "queue_item"

    PLAYER ||--o{ TRANSCODING : "uses"

    SCROBBLE_BUFFER ||--o{ ANNOTATION : "submit_to_service"

    RADIO : radio_id PK
    RADIO : name
    RADIO : stream_url
    RADIO : homepage_url

    USER : user_id PK
    USER : user_name UK
    USER : password
    USER : is_admin
    USER : created_at

    ARTIST : artist_id PK
    ARTIST : name
    ARTIST : album_count

    ALBUM : album_id PK
    ALBUM : name
    ALBUM : artist_id FK
    ALBUM : year
    ALBUM : duration
    ALBUM : genre

    MEDIA_FILE : media_file_id PK
    MEDIA_FILE : path
    MEDIA_FILE : title
    MEDIA_FILE : album_id FK
    MEDIA_FILE : artist_id FK
    MEDIA_FILE : duration
    MEDIA_FILE : bit_rate
    MEDIA_FILE : created_at

    PLAYLIST : playlist_id PK
    PLAYLIST : name
    PLAYLIST : owner FK "user_id"
    PLAYLIST : public
    PLAYLIST : created_at

    PLAYLIST_TRACKS : playlist_id FK
    PLAYLIST_TRACKS : media_file_id FK
    PLAYLIST_TRACKS : track_position

    ANNOTATION : user_id FK
    ANNOTATION : item_id
    ANNOTATION : item_type
    ANNOTATION : play_count
    ANNOTATION : rating
    ANNOTATION : starred

    PLAYER : player_id PK
    PLAYER : user_id FK
    PLAYER : name

    PLAYQUEUE : playqueue_id PK
    PLAYQUEUE : user_id FK
    PLAYQUEUE : media_file_id FK
    PLAYQUEUE : current_index

    BOOKMARK : bookmark_id PK
    BOOKMARK : user_id FK
    BOOKMARK : media_file_id FK
    BOOKMARK : position

    TRANSCODING : transcoding_id PK
    TRANSCODING : name
    TRANSCODING : target_format

    GENRE : genre_id PK
    GENRE : name

    ALBUM_GENRES : album_id FK
    ALBUM_GENRES : genre_id FK

    MEDIA_FILE_GENRES : media_file_id FK
    MEDIA_FILE_GENRES : genre_id FK

    ARTIST_GENRES : artist_id FK
    ARTIST_GENRES : genre_id FK

    LIBRARY : library_id PK
    LIBRARY : name
    LIBRARY : path

    LIBRARY_ARTIST : library_id FK
    LIBRARY_ARTIST : artist_id FK

    LIBRARY_TAG : library_id FK
    LIBRARY_TAG : tag_id FK

    USER_LIBRARY : user_id FK
    USER_LIBRARY : library_id FK

    TAG : tag_id PK
    TAG : name

    FOLDER : folder_id PK
    FOLDER : path
    FOLDER : parent_path

    ALBUM_ARTISTS : album_id FK
    ALBUM_ARTISTS : artist_id FK

    MEDIA_FILE_ARTISTS : media_file_id FK
    MEDIA_FILE_ARTISTS : artist_id FK

    SHARE : share_id PK
    SHARE : name
    SHARE : resource_type
    SHARE : created_by FK "user_id"

    SCROBBLE_BUFFER : scrobble_id PK
    SCROBBLE_BUFFER : user_id FK
    SCROBBLE_BUFFER : media_file_id FK

    USER_PROPS : user_id FK
    USER_PROPS : property_name
```

## Dominios Principales

### 🔐 Usuarios & Acceso
- **USER**: Información de usuarios del sistema (admin, credenciales, timestamps)
- **USER_PROPS**: Propiedades personalizadas de cada usuario
- **PLAYER**: Reproductores/dispositivos registrados por usuario
- **USER_LIBRARY**: Control de acceso a librerías

### 🎵 Contenido Musical
- **ARTIST**: Artistas individuales
- **ALBUM**: Álbumes con metadatos (año, duración, género)
- **MEDIA_FILE**: Canciones/archivos de audio (path, bitrate, duration)
- **FOLDER**: Estructura de carpetas del sistema de archivos
- **GENRE**: Géneros musicales

### 🏷️ Relaciones Múltiples
- **ALBUM_ARTISTS, MEDIA_FILE_ARTISTS**: Soporte para múltiples artistas
- **{ALBUM|MEDIA_FILE|ARTIST}_GENRES**: Géneros en tabla de unión
- **LIBRARY_ARTIST, LIBRARY_TAG**: Agrupaciones por librería

### ▶️ Reproducción & Escucha
- **PLAYLIST**: Listas de reproducción (públicas o privadas)
- **PLAYLIST_TRACKS**: Tabla de unión que preserva orden
- **PLAYQUEUE**: Cola actual del usuario
- **BOOKMARK**: Posiciones guardadas en canciones
- **ANNOTATION**: Ratings, play_count, starred (polimórfico)

### 🌐 Distribución & Integración
- **SHARE**: Enlaces compartidos con expiración y contador de visitas
- **TRANSCODING**: Configuraciones de transcodificación (MP3, WEBM, etc.)
- **SCROBBLE_BUFFER**: Buffer para enviar plays a Last.fm, ListenBrainz
- **RADIO**: Radios de internet

### 📚 Multi-Librería
- **LIBRARY**: Colecciones de música independientes
- **USER_LIBRARY**: Asignación de usuarios a librerías
- **LIBRARY_TAG**: Tags a nivel de librería

## Características del Diseño

1. ✅ **Foreign Keys con Integridad Referencial**: Eliminación en cascada configurada
2. ✅ **Tablas de Unión**: Soporta relaciones muchos-a-muchos (`PLAYLIST_TRACKS`, géneros)
3. ✅ **Anotaciones Polimórficas**: Una tabla `ANNOTATION` para múltiples entidades
4. ✅ **Usuario Central**: Presente en `ANNOTATION`, `PLAYQUEUE`, `PLAYLIST`, `BOOKMARK`
5. ✅ **Timestamps de Auditoría**: `created_at`, `updated_at` en tablas principales
6. ✅ **Full-Text Search**: Índices en `NOCASE` para búsquedas rápidas
7. ✅ **WAL Mode**: SQLite en Write-Ahead Logging para mejor concurrencia

## Para Tests de Base de Datos

Al probar persistencia, enfócate en:

- **Foreign Keys**: `PLAYLIST_TRACKS` → `MEDIA_FILE`, `ALBUM`, etc.
- **Constraints Únicos**: `user_name`, `email` en `USER`
- **Triggers** (si existen): Actualizaciones de contadores (`song_count`, `album_count`)
- **Cascadas**: Eliminación de usuario → eliminación de anotaciones, playlists, etc.
- **Orden en Uniones**: `PLAYLIST_TRACKS` preserva posición numérica

## Referencias

- Ubicación del DB: `./data/navidrome.db` (SQLite)
- Migraciones: `db/migrations/`
- Repositorios: `persistence/`
- Modelos: `model/`
