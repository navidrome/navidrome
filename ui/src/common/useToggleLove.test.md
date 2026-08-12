# useToggleLove — Documentación de tests

## Mocks globales

Aplicados en todos los tests vía `beforeEach`:

| Mock | Reemplaza |
|---|---|
| `subsonic.star` / `subsonic.unstar` | Llamadas HTTP a la API Subsonic |
| `useDataProvider` → `{ getOne }` | Acceso al data provider de React Admin |
| `useNotify` → `mockNotify` | Notificaciones que el hook dispara al usuario |

## Casos de prueba

| Función bajo prueba | Caso de prueba | Mocks utilizados |
|---|---|---|
| `toggleLove` (acción de star) | Usa `mediaFileId` para llamar a `star()` cuando el record lo tiene | `subsonic.star`, `dataProvider.getOne` |
| `toggleLove` (acción de star) | Usa `record.id` como fallback cuando no hay `mediaFileId` | `subsonic.star`, `dataProvider.getOne` |
| `toggleLove` (acción de unstar) | Llama a `unstar()` cuando el record ya tiene `starred: true` | `subsonic.unstar` |
| `refreshRecord` (playlist track) | Hace `getOne` tanto al playlist track como a la song cuando hay `mediaFileId` + `playlistId` | `subsonic.star`, `dataProvider.getOne` (×2) |
| `refreshRecord` (playlist track) | Incluye el filtro `playlist_id` al refrescar el playlist track | `subsonic.unstar`, `dataProvider.getOne` |
| `refreshRecord` (song directa) | Solo hace un `getOne` al resource original cuando no hay `mediaFileId` | `subsonic.star`, `dataProvider.getOne` (×1) |
| `refreshRecord` (song directa) | No incluye filtro `playlist_id` para recursos que no son playlist | `subsonic.star`, `dataProvider.getOne` |
| `toggleLove` (error handling) | Llama a `notify` con mensaje de error cuando `star()` falla | `subsonic.star` (rechazado), `useNotify` |
| `toggleLove` (error handling) | Llama a `notify` con mensaje de error cuando `unstar()` falla | `subsonic.unstar` (rechazado), `useNotify` |
| `loading` (estado) | Es `true` mientras la llamada está pendiente y `false` al resolverse | `subsonic.star` (promesa manual con `resolveToggle`) |
| `loading` (estado) | Vuelve a `false` incluso cuando la llamada falla | `subsonic.star` (rechazado) |
