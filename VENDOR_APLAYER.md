# Vendoring APlayer Assets (✅ COMPLETED)

The APlayer integration now uses locally vendored assets instead of CDN-hosted files. This provides better reliability, offline support, and privacy.

## Implementation Status: ✅ Complete

The following has been implemented:

1. ✅ Asset handlers created (`server/public/handle_aplayer_assets.go`)
2. ✅ Routes added for `/public/aplayer/APlayer.min.css` and `/public/aplayer/APlayer.min.js`
3. ✅ Template updated to use local assets
4. ✅ Files downloaded to `resources/` folder

## Benefits

- ✅ Works in offline/intranet environments
- ✅ No external dependencies
- ✅ Better privacy (no CDN tracking)
- ✅ Consistent versioning
- ✅ Faster load times (no external requests)
- ✅ Assets cached for 1 year for performance

## How It Works

1. APlayer CSS and JS files are stored in `resources/` directory
2. Go's embed.FS automatically embeds them into the binary
3. Public routes serve the files at `/public/aplayer/APlayer.min.css` and `/public/aplayer/APlayer.min.js`
4. The HTML template references these local URLs
5. Browser caches assets for optimal performance

## Updating APlayer Version

To update APlayer to a newer version:

1. Download new files:
   ```bash
   curl -o resources/APlayer.min.css https://cdn.jsdelivr.net/npm/aplayer@VERSION/dist/APlayer.min.css
   curl -o resources/APlayer.min.js https://cdn.jsdelivr.net/npm/aplayer@VERSION/dist/APlayer.min.js
   ```

2. Rebuild Navidrome:
   ```bash
   go build
   ```

The new version will be embedded automatically.

## Files Involved

- `resources/APlayer.min.css` - APlayer stylesheet (12.5 KB)
- `resources/APlayer.min.js` - APlayer library (59.3 KB)
- `server/public/handle_aplayer_assets.go` - Asset serving handlers
- `server/public/public.go` - Route registration
- `resources/aplayer.html` - Template with local asset references
