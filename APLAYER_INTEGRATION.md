# APlayer Integration for Navidrome Shares

This integration allows you to share music from Navidrome using APlayer, a beautiful HTML5 music player, without requiring authentication.

## Features

- 🎵 Beautiful, responsive music player interface
- 🔐 No authentication required - works with public share links
- ⏰ Respects share expiration dates
- 🎨 Clean, modern design
- 📱 Mobile-friendly
- 🔗 Easy to embed on external websites

## How to Use

### 1. Create a Share in Navidrome

1. In Navidrome, select songs, albums, or playlists you want to share
2. Click the share button and create a share link
3. Configure the share settings (expiration, description, etc.)

### 2. Get the APlayer URL

1. Go to the Navidrome admin panel
2. Navigate to "Shares" in the menu
3. Click on your share to edit it
4. You'll see two URLs:
   - **Share URL**: The regular Navidrome share page
   - **APlayer Embed URL**: The APlayer player page

### 3. Share or Embed

You can either:

- **Direct link**: Share the APlayer URL directly for people to listen in their browser
- **Embed in website**: Use an iframe to embed the player on your own website

#### Embed Example

```html
<iframe
  src="http://your-navidrome-server/share/SHARE_ID/aplayer"
  width="100%"
  height="500"
  frameborder="0"
  allow="autoplay">
</iframe>
```

## Technical Details

### How It Works

1. The APlayer page loads the share data from the server (no authentication needed)
2. Track streaming uses JWT tokens embedded in the share link
3. Tokens automatically expire when the share expires
4. All streaming is done through Navidrome's public API endpoints

### Security

- No username/password required
- Uses the same security model as regular Navidrome shares
- JWT tokens are scoped to specific shares
- Respects share expiration dates
- Cannot access data outside the shared content

### Files Added/Modified

**New Files:**
- `resources/aplayer.html` - HTML template for the APlayer page
- `resources/aplayer-share.js` - JavaScript that initializes APlayer with share data

**Modified Files:**
- `server/public/public.go` - Added route for `/share/:id/aplayer`
- `server/public/handle_shares.go` - Added handler for APlayer page
- `ui/src/utils/urls.js` - Added `shareAPlayerUrl()` function
- `ui/src/share/ShareEdit.jsx` - Added APlayer URL display

## Customization

### Styling

You can customize the appearance by modifying `resources/aplayer.html`. The default theme uses a purple gradient background, but you can change:

- Colors and gradients
- Player theme color
- Layout and spacing
- Font styles

### Player Options

Edit `resources/aplayer-share.js` to modify APlayer settings:

```javascript
const ap = new APlayer({
  autoplay: false,    // Auto-start playback
  theme: '#b7daff',   // Player color theme
  loop: 'all',        // Loop mode (all/one/none)
  volume: 0.7,        // Default volume (0-1)
  // ... more options
});
```

For all available options, see [APlayer documentation](https://aplayer.js.org/).

## Credits

- [Navidrome](https://github.com/navidrome/navidrome) - Modern Music Server
- [APlayer](https://github.com/DIYgod/APlayer) - Beautiful HTML5 Music Player
- [AplayerForNavidrome](https://github.com/maytom2016/AplayerForNavidrome) - Original inspiration

## License

This integration follows the same license as Navidrome (GPL-3.0).
