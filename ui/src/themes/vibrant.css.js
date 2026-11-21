// Stylesheet for the audio player (react-jinke-music-player), which does not use the MUI theme.
// `c` is the resolved color set and `r` the radii from vibrant.js.
const createVibrantPlayerStylesheet = (c, r) => `
.react-jinke-music-player-main .music-player-panel {
  background-color: ${c.paper};
  color: ${c.text};
  border-radius: ${r.card} ${r.card} 0 0;
  box-shadow: 0 -8px 32px ${c.shadow};
}

.react-jinke-music-player-main .music-player-panel .music-player-controller,
.react-jinke-music-player-main .music-player-panel .player-content {
  background-color: transparent;
}

.react-jinke-music-player-main svg {
  color: ${c.text};
}

.react-jinke-music-player-main svg:hover,
.react-jinke-music-player-main svg:active {
  color: ${c.accentText};
}

.react-jinke-music-player-main .music-player-panel .panel-content .player-content .play-btn svg {
  color: ${c.onAccent};
  background-color: ${c.accent};
  border-radius: ${r.round};
  padding: 4px;
  box-shadow: 0 4px 14px ${c.accentGlow};
}

.react-jinke-music-player-main .music-player-panel .panel-content .player-content .play-btn svg:hover {
  color: ${c.onAccent};
  filter: brightness(1.08);
}

.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-rail {
  background-color: ${c.elevated};
}

.react-jinke-music-player-main .rc-slider-rail,
.react-jinke-music-player-main .rc-slider-track {
  height: 4px;
  border-radius: ${r.pill};
}

.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle,
.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-track,
.react-jinke-music-player-mobile-progress .rc-slider-handle,
.react-jinke-music-player-mobile-progress .rc-slider-track {
  background-color: ${c.accent};
  border-color: ${c.accent};
}

.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle:active {
  box-shadow: 0 0 0 6px ${c.accentSoft};
}

.react-jinke-music-player-main .img-content,
.react-jinke-music-player-mobile-cover {
  border-radius: ${r.small};
}

.react-jinke-music-player-main.mini-player {
  border-radius: ${r.card};
}

.react-jinke-music-player-main .audio-lists-panel {
  background-color: ${c.elevated};
  color: ${c.text};
  border-radius: ${r.card};
  box-shadow: 0 16px 48px ${c.shadow};
  overflow: hidden;
}

.react-jinke-music-player-main .audio-lists-panel-header {
  background-color: ${c.elevated};
  border-bottom: 1px solid ${c.divider};
}

.audio-lists-panel-content .audio-item {
  background-color: transparent !important;
  color: ${c.text};
  border-radius: ${r.small};
}

.audio-lists-panel-content .audio-item:hover {
  background-color: ${c.accentHover} !important;
}

.audio-lists-panel-content .audio-item.playing,
.audio-lists-panel-content .audio-item.playing svg,
.react-jinke-music-player-main .audio-item.playing .player-singer {
  color: ${c.accentText} !important;
}

.audio-lists-panel-content .audio-item.playing {
  background-color: ${c.accentSoft} !important;
}

.react-jinke-music-player-main ::-webkit-scrollbar-thumb {
  background-color: ${c.textSecondary};
  border-radius: ${r.pill};
}

.react-jinke-music-player-main .loading svg {
  color: ${c.accentText} !important;
}

.react-jinke-music-player-mobile-play-model-tip {
  background-color: ${c.accent};
  color: ${c.onAccent};
  border-radius: ${r.pill};
}
`

export default createVibrantPlayerStylesheet
