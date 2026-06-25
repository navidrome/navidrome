const stylesheet = `
  .react-jinke-music-player-main.light-theme svg,
  .react-jinke-music-player .music-player-controller,
  .react-jinke-music-player .audio-circle-process-bar circle[class='stroke'] {
    color: #797593;
    stroke: #797593;
  }

  .react-jinke-music-player-main svg:active,
  .react-jinke-music-player-main svg:hover {
    color: #907aa9;
  }

  .react-jinke-music-player-main.light-theme svg:active,
  .react-jinke-music-player-main.light-theme svg:hover {
    color: #907aa9;
  }

  .react-jinke-music-player-mobile-play-model-tip,
  .react-jinke-music-player-main.light-theme .play-mode-title {
    background-color: #d7827e;
    color: #faf4ed;
  }

  .react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle,
  .react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-track {
    background-color: #d7827e;
  }

  .react-jinke-music-player-main ::-webkit-scrollbar-thumb {
    background-color: #d7827e;
  }

  .react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle:active {
    box-shadow: 0 0 2px #d7827e;
  }

  .react-jinke-music-player-main .audio-item.playing svg {
    color: #d7827e;
  }

  .react-jinke-music-player-main .audio-item.playing .player-singer {
    color: #d7827e !important;
  }

  .react-jinke-music-player-main .loading svg {
    color: #d7827e !important;
  }

  .react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle {
    border: none;
    box-shadow:
      rgba(70, 66, 97, 0.12) 0px 4px 6px,
      rgba(70, 66, 97, 0.08) 0px 5px 7px;
  }

  .rc-slider-rail,
  .rc-slider-track {
    height: 6px;
  }

  .rc-slider {
    padding: 3px 0;
  }

  .react-jinke-music-player-main.light-theme .rc-switch-checked {
    background-color: #d7827e !important;
    border: 1px solid #d7827e;
  }

  .sound-operation > div:nth-child(4) {
    transform: translateX(-50%) translateY(5%) !important;
  }

  .sound-operation {
    padding: 4px 0;
  }

  .react-jinke-music-player-main .music-player-panel {
    background-color: #fffaf3;
    color: #464261;
    box-shadow: 0 0 8px rgba(70, 66, 97, 0.12);
  }

  .react-jinke-music-player-main.light-theme .music-player-panel {
    color: #464261;
  }

  .audio-lists-panel {
    background-color: #fffaf3;
    bottom: 6.25rem;
    box-shadow:
      rgba(70, 66, 97, 0.12) 0px 4px 6px,
      rgba(70, 66, 97, 0.08) 0px 5px 7px;
  }

  .audio-lists-panel-content .audio-item.playing {
    background-color: rgba(0, 0, 0, 0);
  }

  .audio-lists-panel-content .audio-item:nth-child(2n+1) {
    background-color: rgba(0, 0, 0, 0);
  }

  .audio-lists-panel-header {
    border-bottom: 1px solid #f2e9e1;
    box-shadow: none;
  }

  .react-jinke-music-player-main .music-player-panel .panel-content .player-content .audio-lists-btn {
    background-color: rgba(0, 0, 0, 0);
    box-shadow: 0 0 0 0;
  }

  .react-jinke-music-player-main.light-theme .audio-lists-panel-header {
    background-color: #fffaf3;
    color: #464261;
  }

  .audio-lists-panel-content .audio-item {
    line-height: 32px;
    color: #464261;
  }

  .react-jinke-music-player-main .music-player-panel .panel-content .img-content {
    box-shadow:
      rgba(70, 66, 97, 0.12) 0px 4px 6px,
      rgba(70, 66, 97, 0.08) 0px 5px 7px;
  }

  .react-jinke-music-player-main .music-player-lyric {
    color: #797593;
    -webkit-text-stroke: 0.35px #faf4ed;
    font-weight: bolder;
  }

  .react-jinke-music-player-main .lyric-btn-active,
  .react-jinke-music-player-main .lyric-btn-active svg {
    color: #797593 !important;
  }

  .audio-lists-panel-content .audio-item.playing,
  .audio-lists-panel-content .audio-item.playing svg {
    color: #d7827e;
  }

  .audio-lists-panel-content .audio-item:active .group:not(.player-delete) svg,
  .audio-lists-panel-content .audio-item:hover .group:not(.player-delete) svg {
    color: #d7827e;
  }

  .audio-lists-panel-content .audio-item .player-icons {
    scale: 75%;
  }

  .audio-lists-panel-content .audio-item:active,
  .audio-lists-panel-content .audio-item:hover {
    background-color: #f2e9e1;
  }

  /* Mobile */
  .react-jinke-music-player-mobile-cover {
    border: none;
    box-shadow:
      rgba(70, 66, 97, 0.12) 0px 4px 6px,
      rgba(70, 66, 97, 0.08) 0px 5px 7px;
  }

  .react-jinke-music-player .music-player-controller {
    border: none;
    background-color: #fffaf3;
    border-color: #fffaf3;
    box-shadow:
      rgba(70, 66, 97, 0.12) 0px 4px 6px,
      rgba(70, 66, 97, 0.08) 0px 5px 7px;
    color: #d7827e;
  }

  .react-jinke-music-player .music-player-controller.music-player-playing:before {
    border: 1px solid rgba(70, 66, 97, 0.18);
  }

  .react-jinke-music-player .music-player-controller .music-player-controller-setting {
    background: rgba(215, 130, 126, 0.2);
    color: #faf4ed;
  }

  .react-jinke-music-player-mobile-progress .rc-slider-handle,
  .react-jinke-music-player-mobile-progress .rc-slider-track {
    background-color: #d7827e;
  }

  .react-jinke-music-player-mobile-progress .rc-slider-handle {
    border: none;
  }
`

export default stylesheet
