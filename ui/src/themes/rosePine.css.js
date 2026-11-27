const stylesheet = `

.react-jinke-music-player-main svg:active, .react-jinke-music-player-main svg:hover {
    color: #c4a7e7
}

.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle, .react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-track {
    background-color: #ebbcba
}

.react-jinke-music-player-main ::-webkit-scrollbar-thumb {
    background-color: #ebbcba;
}

.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle:active {
    box-shadow: 0 0 2px #ebbcba
}

.react-jinke-music-player-main .audio-item.playing svg {
    color: #ebbcba
}

.react-jinke-music-player-main .audio-item.playing .player-singer {
    color: #ebbcba !important
}

.react-jinke-music-player-main .loading svg {
    color: #ebbcba !important
}


.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle {
    border: none;
    box-shadow:rgba(25, 23, 36, 0.35) 0px 4px 6px, rgba(25, 23, 36, 0.2) 0px 5px 7px;
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

.react-jinke-music-player-main .music-player-panel {
    background-color: #1f1d2e;
    color: #e0def4;
    box-shadow: 0 0 8px rgba(25, 23, 36, 0.35);
}

.audio-lists-panel {
    background-color: #1f1d2e;
    bottom: 6.25rem;
    box-shadow:rgba(25, 23, 36, 0.35) 0px 4px 6px, rgba(25, 23, 36, 0.2) 0px 5px 7px;
}

.audio-lists-panel-content .audio-item.playing {
    background-color: rgba(0, 0, 0, 0);
}

.audio-lists-panel-content .audio-item:nth-child(2n+1) {
    background-color: rgba(0, 0, 0, 0);
}


.react-jinke-music-player-main .music-player-panel .panel-content .player-content .audio-lists-btn {
    background-color:rgba(0,0,0,0);
    box-shadow:0 0 0 0;
}

.audio-lists-panel-content .audio-item {
    line-height: 32px;
}

.react-jinke-music-player-main .music-player-panel .panel-content .img-content {
    box-shadow:rgba(25, 23, 36, 0.35) 0px 4px 6px, rgba(25, 23, 36, 0.2) 0px 5px 7px;
}

.react-jinke-music-player-main .music-player-lyric {
    color: #908caa;
    -webkit-text-stroke: 0.5px #191724;
    font-weight: bolder;
}

.react-jinke-music-player-main .lyric-btn-active, .react-jinke-music-player-main .lyric-btn-active svg {
    color: #908caa !important;
}

.audio-lists-panel-header {
    border-bottom:1px solid #26233a;
    box-shadow:none;
}

.audio-lists-panel-content .audio-item.playing, .audio-lists-panel-content .audio-item.playing svg {
    color: #ebbcba
}

.audio-lists-panel-content .audio-item:active .group:not(.player-delete) svg, .audio-lists-panel-content .audio-item:hover .group:not(.player-delete) svg {
    color: #ebbcba
}

.audio-lists-panel-content .audio-item .player-icons {
    scale: 75%;
}

.audio-lists-panel-content .audio-item:active,
.audio-lists-panel-content .audio-item:hover {
    background-color: #26233a;
}

/* Mobile */

.react-jinke-music-player-mobile-cover {
    border: none;
    box-shadow:rgba(25, 23, 36, 0.35) 0px 4px 6px, rgba(25, 23, 36, 0.2) 0px 5px 7px;
}

.react-jinke-music-player .music-player-controller {
    border: none;
    background-color: #1f1d2e;
    border-color: #1f1d2e;
    box-shadow:rgba(25, 23, 36, 0.35) 0px 4px 6px, rgba(25, 23, 36, 0.2) 0px 5px 7px;
    color: #ebbcba;
}

.react-jinke-music-player .music-player-controller .music-player-controller-setting {
    color: rgba(196,167,231,.3);
}

.react-jinke-music-player-mobile-progress .rc-slider-handle, .react-jinke-music-player-mobile-progress .rc-slider-track {
    background-color: #ebbcba;
}

.react-jinke-music-player-mobile-progress .rc-slider-handle {
    border: none;
}
`

export default stylesheet
