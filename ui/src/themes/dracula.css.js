const stylesheet = `

/* Icon hover: pink */
.react-jinke-music-player-main svg:active, .react-jinke-music-player-main svg:hover {
    color: #ff79c6
}

/* Progress bar: purple */
.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle, .react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-track {
    background-color: #bd93f9
}

/* Volume bar: green */
.sound-operation .rc-slider-handle, .sound-operation .rc-slider-track {
    background-color: #50fa7b !important
}

.sound-operation .rc-slider-handle:active {
    box-shadow: 0 0 2px #50fa7b !important
}

/* Scrollbar: comment */
.react-jinke-music-player-main ::-webkit-scrollbar-thumb {
    background-color: #6272a4;
}

.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle:active {
    box-shadow: 0 0 2px #bd93f9
}

/* Now playing icon: cyan */
.react-jinke-music-player-main .audio-item.playing svg {
    color: #8be9fd
}

/* Now playing artist: cyan */
.react-jinke-music-player-main .audio-item.playing .player-singer {
    color: #8be9fd !important
}

/* Loading spinner: orange */
.react-jinke-music-player-main .loading svg {
    color: #ffb86c !important
}

.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle {
    border: hidden;
    box-shadow: rgba(20, 21, 28, 0.25) 0px 4px 6px, rgba(20, 21, 28, 0.1) 0px 5px 7px;
}

.rc-slider-rail, .rc-slider-track {
    height: 6px;
}

.rc-slider {
    padding: 3px 0;
}

.sound-operation > div:nth-child(4) {
    transform: translateX(-50%) translateY(5%) !important;
}

.sound-operation {
    padding: 4px 0;
}

/* Player panel background */
.react-jinke-music-player-main .music-player-panel {
    background-color: #282a36;
    color: #f8f8f2;
    box-shadow: 0 0 8px rgba(0, 0, 0, 0.25);
}

/* Song title in player: foreground */
.react-jinke-music-player-main .music-player-panel .panel-content .player-content .audio-title {
    color: #f8f8f2;
}

/* Duration/time text: yellow */
.react-jinke-music-player-main .music-player-panel .panel-content .player-content .duration, .react-jinke-music-player-main .music-player-panel .panel-content .player-content .current-time {
    color: #f1fa8c
}

/* Audio list panel */
.audio-lists-panel {
    background-color: #282a36;
    bottom: 6.25rem;
    box-shadow: rgba(20, 21, 28, 0.25) 0px 4px 6px, rgba(20, 21, 28, 0.1) 0px 5px 7px;
}

.audio-lists-panel-content .audio-item.playing {
    background-color: transparent;
}

.audio-lists-panel-content .audio-item:nth-child(2n+1) {
    background-color: transparent;
}

/* Playlist hover: current line */
.audio-lists-panel-content .audio-item:active,
.audio-lists-panel-content .audio-item:hover {
    background-color: #44475a;
}

.audio-lists-panel-header {
    border-bottom: 1px solid rgba(0, 0, 0, 0.25);
    box-shadow: none;
}

/* Playlist header text: orange */
.audio-lists-panel-header-title {
    color: #ffb86c;
}

.react-jinke-music-player-main .music-player-panel .panel-content .player-content .audio-lists-btn {
    background-color: transparent;
    box-shadow: none;
}

.audio-lists-panel-content .audio-item {
    line-height: 32px;
}

.react-jinke-music-player-main .music-player-panel .panel-content .img-content {
    box-shadow: rgba(20, 21, 28, 0.25) 0px 4px 6px, rgba(20, 21, 28, 0.1) 0px 5px 7px;
}

/* Lyrics: yellow */
.react-jinke-music-player-main .music-player-lyric {
    color: #f1fa8c;
    -webkit-text-stroke: 0.5px #282a36;
    font-weight: bolder;
}

/* Lyric button active: yellow */
.react-jinke-music-player-main .lyric-btn-active, .react-jinke-music-player-main .lyric-btn-active svg {
    color: #f1fa8c !important;
}

/* Playlist now playing: cyan */
.audio-lists-panel-content .audio-item.playing, .audio-lists-panel-content .audio-item.playing svg {
    color: #8be9fd
}

/* Playlist hover icons: pink */
.audio-lists-panel-content .audio-item:active .group:not(.player-delete) svg, .audio-lists-panel-content .audio-item:hover .group:not(.player-delete) svg {
    color: #ff79c6
}

.audio-lists-panel-content .audio-item .player-icons {
    scale: 75%;
}

/* Mobile */

.react-jinke-music-player-mobile-cover {
    border: none;
    box-shadow: rgba(20, 21, 28, 0.25) 0px 4px 6px, rgba(20, 21, 28, 0.1) 0px 5px 7px;
}

.react-jinke-music-player .music-player-controller {
    border: none;
    box-shadow: rgba(20, 21, 28, 0.25) 0px 4px 6px, rgba(20, 21, 28, 0.1) 0px 5px 7px;
    color: #bd93f9;
}

.react-jinke-music-player .music-player-controller .music-player-controller-setting {
    color: rgba(189, 147, 249, 0.3);
}

/* Mobile progress: green */
.react-jinke-music-player-mobile-progress .rc-slider-handle, .react-jinke-music-player-mobile-progress .rc-slider-track {
    background-color: #50fa7b;
}

.react-jinke-music-player-mobile-progress .rc-slider-handle {
    border: none;
}
`

export default stylesheet
