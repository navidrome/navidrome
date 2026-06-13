const stylesheet = `
  .react-jinke-music-player-main.light-theme svg,
  .react-jinke-music-player .music-player-controller,
  .react-jinke-music-player .audio-circle-process-bar circle[class='stroke'] {
    color: #6c6f85;
    stroke: #6c6f85;
  }

  .react-jinke-music-player-main svg:active,
  .react-jinke-music-player-main svg:hover {
    color: #7c7f93;
  }

  .react-jinke-music-player-main.light-theme svg:active,
  .react-jinke-music-player-main.light-theme svg:hover {
    color: #7c7f93;
  }

  .react-jinke-music-player-mobile-play-model-tip,
  .react-jinke-music-player-main.light-theme .play-mode-title {
    background-color: #6c6f85;
    color: #eff1f5;
  }

  .react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle,
  .react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-track {
    background-color: #6c6f85;
  }

  .react-jinke-music-player-main ::-webkit-scrollbar-thumb {
    background-color: #6c6f85;
  }

  .react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle:active {
    box-shadow: 0 0 2px #6c6f85;
  }

  .react-jinke-music-player-main .audio-item.playing svg {
    color: #6c6f85;
  }

  .react-jinke-music-player-main .audio-item.playing .player-singer {
    color: #6c6f85 !important;
  }

  .react-jinke-music-player-main .loading svg {
    color: #6c6f85 !important;
  }

  .react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle {
    border: hidden;
    box-shadow:
      rgba(76, 79, 105, 0.12) 0px 4px 6px,
      rgba(76, 79, 105, 0.08) 0px 5px 7px;
  }

  .rc-slider-rail,
  .rc-slider-track {
    height: 6px;
  }

  .rc-slider {
    padding: 3px 0;
  }

  .react-jinke-music-player-main.light-theme .rc-switch-checked {
    background-color: #6c6f85 !important;
    border: 1px solid #6c6f85;
  }

  .sound-operation > div:nth-child(4) {
    transform: translateX(-50%) translateY(5%) !important;
  }

  .sound-operation {
    padding: 4px 0;
  }

  .react-jinke-music-player-main .music-player-panel {
    background-color: #e6e9ef;
    color: #4c4f69;
    box-shadow: 0 0 8px rgba(76, 79, 105, 0.15);
  }

  .react-jinke-music-player-main.light-theme .music-player-panel {
    color: #4c4f69;
  }

  .audio-lists-panel {
    background-color: #e6e9ef;
    bottom: 6.25rem;
    box-shadow:
      rgba(76, 79, 105, 0.12) 0px 4px 6px,
      rgba(76, 79, 105, 0.08) 0px 5px 7px;
  }

  .audio-lists-panel-content .audio-item.playing {
    background-color: rgba(0, 0, 0, 0);
  }

  .audio-lists-panel-content .audio-item:nth-child(2n+1) {
    background-color: rgba(0, 0, 0, 0);
  }

  .audio-lists-panel-content .audio-item:active,
  .audio-lists-panel-content .audio-item:hover {
    background-color: rgba(76, 79, 105, 0.08);
  }

  .audio-lists-panel-header {
    border-bottom: 1px solid #ccd0da;
    box-shadow: none;
  }

  .react-jinke-music-player-main .music-player-panel .panel-content .player-content .audio-lists-btn {
    background-color: rgba(0, 0, 0, 0);
    box-shadow: 0 0 0 0;
  }

  .react-jinke-music-player-main.light-theme .audio-lists-panel-header {
    background-color: #e6e9ef;
    color: #4c4f69;
  }

  .audio-lists-panel-content .audio-item {
    line-height: 32px;
    color: #4c4f69;
  }

  .react-jinke-music-player-main .music-player-panel .panel-content .img-content {
    box-shadow:
      rgba(76, 79, 105, 0.12) 0px 4px 6px,
      rgba(76, 79, 105, 0.08) 0px 5px 7px;
  }

  .react-jinke-music-player-main .music-player-lyric {
    color: #6c6f85; /* subtext0 */
    -webkit-text-stroke: 0.35px #eff1f5;
    font-weight: bolder;
  }

  .react-jinke-music-player-main .lyric-btn-active,
  .react-jinke-music-player-main .lyric-btn-active svg {
    color: #6c6f85 !important;
  }

  .audio-lists-panel-content .audio-item.playing,
  .audio-lists-panel-content .audio-item.playing svg {
    color: #6c6f85;
  }

  .audio-lists-panel-content .audio-item:active .group:not([class=".player-delete"]) svg,
  .audio-lists-panel-content .audio-item:hover .group:not([class=".player-delete"]) svg {
    color: #6c6f85;
  }

  .audio-lists-panel-content .audio-item .player-icons {
    scale: 75%;
  }

  .audio-lists-panel-content .audio-item:active,
  .audio-lists-panel-content .audio-item:hover {
    background-color: #dce0e8; /* surface1 */
  }

  /* Mobile */
  .react-jinke-music-player-mobile-cover {
    border: none;
    box-shadow:
      rgba(76, 79, 105, 0.12) 0px 4px 6px,
      rgba(76, 79, 105, 0.08) 0px 5px 7px;
  }

  .react-jinke-music-player .music-player-controller {
    border: none;
    background-color: #e6e9ef;
    border-color: #e6e9ef;
    box-shadow:
      rgba(76, 79, 105, 0.12) 0px 4px 6px,
      rgba(76, 79, 105, 0.08) 0px 5px 7px;
    color: #6c6f85;
  }

  .react-jinke-music-player .music-player-controller.music-player-playing:before {
    border: 1px solid rgba(76, 79, 105, 0.18);
  }

  .react-jinke-music-player .music-player-controller .music-player-controller-setting {
    background: rgba(108, 111, 133, 0.2);
    color: #eff1f5;
  }

  .react-jinke-music-player-mobile-progress .rc-slider-handle,
  .react-jinke-music-player-mobile-progress .rc-slider-track {
    background-color: #6c6f85;
  }

  .react-jinke-music-player-mobile-progress .rc-slider-handle {
    border: none;
  }
`

export default stylesheet
