const stylesheet = `

.react-jinke-music-player-main svg:active, .react-jinke-music-player-main svg:hover {
    color: #bd4aff;
}

.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle, .react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-track {
    background-color: #8800cb;
}

.react-jinke-music-player-main ::-webkit-scrollbar-thumb {
    background-color: #8800cb;
}

.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle:active {
    box-shadow: 0 0 2px #8800cb;
}

.react-jinke-music-player-main .audio-item.playing svg {
    color: #8800cb;
}

.react-jinke-music-player-main .audio-item.playing .player-singer {
    color: #8800cb !important;
}

.audio-lists-panel-content .audio-item.playing, .audio-lists-panel-content .audio-item.playing svg {
    color: #8800cb;
}
.audio-lists-panel-content .audio-item:active .group:not([class=".player-delete"]) svg, .audio-lists-panel-content .audio-item:hover .group:not([class=".player-delete"]) svg {
    color: #8800cb;
}

.react-jinke-music-player-mobile-progress .rc-slider-handle, .react-jinke-music-player-mobile-progress .rc-slider-track {
    background-color: #8800cb;
}
`

export default stylesheet
