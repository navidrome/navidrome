const stylesheet = `

.react-jinke-music-player-main svg:active, .react-jinke-music-player-main svg:hover {
    color: #00a4dc
}

.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle, .react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-track {
    background-color: #00a4dc
}

.react-jinke-music-player-main ::-webkit-scrollbar-thumb {
    background-color: #00a4dc;
}

.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle:active {
    box-shadow: 0 0 2px #00a4dc
}

.react-jinke-music-player-main .audio-item.playing svg {
    color: #00a4dc
}

.react-jinke-music-player-main .audio-item.playing .player-singer {
    color: #00a4dc !important
}

.react-jinke-music-player-main .loading svg {
    color: #00a4dc !important
}


.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle {
    border: hidden;
    box-shadow:rgba(15, 17, 21, 0.25) 0px 4px 6px, rgba(15, 17, 21, 0.1) 0px 5px 7px;
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
    background-color: #1e2030;
    color: #24273a
    box-shadow: 0 0 8px rgba(0, 0, 0, 0.25);
}

.audio-lists-panel {
    background-color: #1e2030;
    bottom: 6.25rem;
    box-shadow:rgba(15, 17, 21, 0.25) 0px 4px 6px, rgba(15, 17, 21, 0.1) 0px 5px 7px;
}

.audio-lists-panel-content .audio-item.playing {
    background-color: rgba(0, 0, 0, 0);
}

.audio-lists-panel-content .audio-item:nth-child(2n+1) {
    background-color: rgba(0, 0, 0, 0);
}

.audio-lists-panel-content .audio-item:active,
.audio-lists-panel-content .audio-item:hover {
    background-color:rgba(255, 255, 255, 0.08);
}

.audio-lists-panel-header {
    border-bottom:1px solid #242936;
}

.react-jinke-music-player-main .music-player-panel .panel-content .player-content .audio-lists-btn {
    background-color:rgba(0,0,0,0);
    box-shadow:0 0 0 0;
}

.audio-lists-panel-content .audio-item {
    line-height: 32px;
}

.react-jinke-music-player-main .music-player-panel .panel-content .img-content {
    box-shadow:rgba(15, 17, 21, 0.25) 0px 4px 6px, rgba(15, 17, 21, 0.1) 0px 5px 7px;
}

.react-jinke-music-player-main .music-player-lyric {
    color: #acb3d2;
    -webkit-text-stroke: 0.5px #2E3440;
    font-weight: bolder;
}

.react-jinke-music-player-main .lyric-btn-active, .react-jinke-music-player-main .lyric-btn-active svg {
    color: #acb3d2 !important;
}

.audio-lists-panel-header {
    border-bottom:1px solid rgba(0, 0, 0, 0.25);
    box-shadow:none;
}

.audio-lists-panel-content .audio-item.playing, .audio-lists-panel-content .audio-item.playing svg {
    color: #00a4dc
}

.audio-lists-panel-content .audio-item:active .group:not([class=".player-delete"]) svg, .audio-lists-panel-content .audio-item:hover .group:not([class=".player-delete"]) svg {
    color: #00a4dc
}

.audio-lists-panel-content .audio-item .player-icons {
    scale: 75%;
}

.audio-lists-panel-content .audio-item:active,
.audio-lists-panel-content .audio-item:hover {
    background-color: #363a4f;
}

/* Mobile */

.react-jinke-music-player-mobile-cover {
    border: none;
    box-shadow:rgba(15, 17, 21, 0.25) 0px 4px 6px, rgba(15, 17, 21, 0.1) 0px 5px 7px;
}

.react-jinke-music-player .music-player-controller {
    border: none;
    box-shadow:rgba(15, 17, 21, 0.25) 0px 4px 6px, rgba(15, 17, 21, 0.1) 0px 5px 7px;
    color: #00a4dc;
}

.react-jinke-music-player .music-player-controller .music-player-controller-setting {
    color: rgba(122,160,236,.3);
}

.react-jinke-music-player-mobile-progress .rc-slider-handle, .react-jinke-music-player-mobile-progress .rc-slider-track {
    background-color: #00a4dc;
}

.react-jinke-music-player-mobile-progress .rc-slider-handle {
    border: none;
}
`

export default stylesheet
