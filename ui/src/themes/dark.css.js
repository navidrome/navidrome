const stylesheet = `

.react-jinke-music-player-main svg:active, .react-jinke-music-player-main svg:hover {
    color: #7171d5
}

.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle, .react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-track {
    background-color: #5f5fc4
}

.react-jinke-music-player-main ::-webkit-scrollbar-thumb {
    background-color: #5f5fc4;
}

.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle:active {
    box-shadow: 0 0 2px #5f5fc4
}

.react-jinke-music-player-main .audio-item.playing svg {
    color: #5f5fc4
}

.react-jinke-music-player-main .audio-item.playing .player-singer {
    color: #5f5fc4 !important
}

.audio-lists-panel-content .audio-item.playing, .audio-lists-panel-content .audio-item.playing svg {
    color: #5f5fc4
}
.audio-lists-panel-content .audio-item:active .group:not([class=".player-delete"]) svg, .audio-lists-panel-content .audio-item:hover .group:not([class=".player-delete"]) svg {
    color: #5f5fc4
}
`

export default stylesheet
