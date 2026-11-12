module.exports = `

:root {
    --lighter: #a6ccff;
    --main: #0084ff;
    --darker: #0062f6;
    --main-background: #14172e;
    --light-background: #181d37;
    --lighter-background: #222541;
    --background-highlight: #32375b;
    --darker-background-highlight: #464b77;
}

.react-jinke-music-player-main svg:active, .react-jinke-music-player-main svg:hover {
    color: var(--lighter);
}

.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle, .react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-track {
    background-color: var(--main)
}

.react-jinke-music-player-main ::-webkit-scrollbar-thumb {
    background-color: var(--main);
}

.react-jinke-music-player-main .music-player-panel .panel-content {
    padding-top: 5px;
}

.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle:active {
    box-shadow: 0 0 2px var(--main)
}

.react-jinke-music-player-main .audio-item.playing svg {
    color: var(--main)
}

.react-jinke-music-player-main .audio-item.playing .player-singer {
    color: var(--main) !important
}

.react-jinke-music-player-main .loading svg {
    color: var(--main) !important
}


.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle {
    border: 0px solid var(--main-background);
}


.rc-slider-rail, .rc-slider-track {
    height: 5px;
}

.rc-slider {
    padding: 3px 0;
}

.progress-bar > div:nth-child(2) > div:nth-child(4) {
    transform: translateX(-50%) translateY(5%) !important;
}

.react-jinke-music-player-main .music-player-panel .panel-content .progress-bar-content .progress-bar {
    margin: 0;
    position: absolute;
    width: 100.5%;
    bottom: 74.5px;
    left: 50%;
    right: 50%;
    transform: translateX(-50%);
}

.react-jinke-music-player-main .music-player-panel .panel-content .progress-bar-content .progress-bar .progress-load-bar {
    background-color: unset;
}

.react-jinke-music-player-main .music-player-panel .panel-content .progress-bar-content .progress-bar .rc-slider-handle {
    opacity: 0;
    transition: opacity .2s;
}

.react-jinke-music-player-main .music-player-panel .panel-content .progress-bar-content .progress-bar .rc-slider-handle:hover {
    opacity: 1;
}

.react-jinke-music-player-main .music-player-panel .panel-content .progress-bar-content .progress-bar .rc-slider-handle:focus {
    opacity: 1;
}

.react-jinke-music-player-main .music-player-panel .panel-content .progress-bar-content .audio-main .current-time, .react-jinke-music-player-main .music-player-panel .panel-content .progress-bar-content .audio-main .duration {
    /*display: none;*/
    font-size: 0.675rem;
    color: rgba(255, 255, 255, 0.8)
}

.react-jinke-music-player-main .music-player-panel .panel-content .progress-bar-content .audio-main .current-time::after {
    content: "/";
    display: ruby;
    margin: 0 3px 0 3px;
}

.current-time {
    flex-basis: unset !important;
}

.react-jinke-music-player-main .music-player-panel .panel-content .progress-bar-content .audio-main {
    justify-content: left;
}

.react-jinke-music-player-main .music-player-panel .panel-content .progress-bar-content .audio-title {
    margin-top: 2.5px;
}

.react-jinke-music-player-main .music-player-panel .panel-content .img-content {
    margin-top: 2.5px;
    height: 60px;
    width: 60px;
}

.sound-operation > div:nth-child(4) {
    transform: translateX(-50%) translateY(5%) !important;
}

.sound-operation {
    padding: 4px 0;
}

.react-jinke-music-player-main .music-player-panel {
    background-color: var(--main-background);
    box-shadow: 0 0 8px rgba(0, 0, 0, 0.25);
    height: 85px;
}

.audio-lists-panel {
    background-color: var(--main-background);
    border-radius: .625rem;
    bottom: 6rem;
    box-shadow: 0px 5px 5px -3px rgba(0,0,0,0.2),0px 8px 10px 1px rgba(0,0,0,0.14),0px 3px 14px 2px rgba(0,0,0,0.12);
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
    padding: 4px 20px;
    border-radius: .5rem;
    margin: 3px;
}

/*.react-jinke-music-player-main .music-player-panel .panel-content .img-content {
    border-radius: .5rem;
}

.react-jinke-music-player-main .music-player-panel .panel-content .img-rotate {
    animation: none;
}*/

.NDAudioPlayer-player-6 .music-player-panel .panel-content div.img-rotate {
  background-size: cover !important;
}


.react-jinke-music-player-main .music-player-panel .panel-content .img-content {
    box-shadow:0px 5px 5px -3px rgba(0,0,0,0.2),0px 8px 10px 1px rgba(0,0,0,0.14),0px 3px 14px 2px rgba(0,0,0,0.12);
}

.react-jinke-music-player-main .music-player-lyric {
    color: var(--darker);
}

.react-jinke-music-player-main .lyric-btn-active, .react-jinke-music-player-main .lyric-btn-active svg {
    color: var(--darker) !important;
}

.player-content > span:nth-child(1) {
    margin-right: 20px !important;
    margin-left: 20px !important;
}

.player-content button:nth-child(2) {
    margin-right: 10px !important;
}

.audio-lists-panel-header {
    border-bottom:1px solid rgba(0, 0, 0, 0.25);
    box-shadow:none;
}

.audio-lists-panel-content .audio-item.playing, .audio-lists-panel-content .audio-item.playing svg {
    color: var(--main)
}

.audio-lists-panel-content .audio-item:active .group:not([class=".player-delete"]) svg, .audio-lists-panel-content .audio-item:hover .group:not([class=".player-delete"]) svg {
    color: var(--main)
}

.audio-lists-panel-content .audio-item .player-icons {
    scale: 75%;
}

.audio-lists-panel-content .audio-item:active,
.audio-lists-panel-content .audio-item:hover {
    background-color:var(--lighter-background);
}

/* Mobile */

.react-jinke-music-player-mobile-cover {
    border: none;
    /*border-radius: 2.5rem;*/
}

.react-jinke-music-player .music-player-controller {
    border: none;
    box-shadow: 0px 5px 5px -3px rgba(0,0,0,0.2),0px 8px 10px 1px rgba(0,0,0,0.14),0px 3px 14px 2px rgba(0,0,0,0.12);
    color: var(--main);
}

.react-jinke-music-player .music-player-controller .music-player-controller-setting {
    color: var(--darker-background-highlight);
    opacity: 0.3;
}

.react-jinke-music-player-mobile-progress .rc-slider-handle, .react-jinke-music-player-mobile-progress .rc-slider-track {
    background-color: var(--main);
}

`
