const stylesheet = `
html { 
    scrollbar-width: none; 
}
body {
    -ms-overflow-style: none;
    font-family: monospace;
}
body::-webkit-scrollbar, body::-webkit-scrollbar-button { 
    display: none; 
}
.react-jinke-music-player-main .music-player-panel {
    background-color: white!important;
    box-shadow: none;
    font-family: monospace;
    color: black;
    border-top: 1px solid black;
}
.react-jinke-music-player-main .music-player-panel .panel-content div.img-content {
    animation: none;
    box-shadow: none;
    border-radius: 5px;
}
.react-jinke-music-player-main .music-player-panel .panel-content .progress-bar-content {
    flex: 0 0 auto;
    width: calc(50% - 150px);
    margin-left: 10px;
    padding: 0;
}
section.audio-main {
    position: absolute;
    width: calc(100% - 131px)!important;
    bottom: 0;
    margin-bottom: 10px;
}
span.audio-title {
    margin-bottom: 20px;
}
span.audio-title .songTitle {
    color: black!important;
}
.react-jinke-music-player-main .music-player-panel .panel-content .player-content {
    flex: 1;
    margin-bottom: 20px;
    padding-left: 0;
}
div.player-content > span:first-child {
    flex: 1!important;
    justify-content: flex-start!important;
}
div.player-content > span:first-child svg {
    width: 50px;
}
.react-jinke-music-player-main .music-player-panel .panel-content .player-content > .group {
    flex: 0;
}
.play-sounds svg, .loop-btn svg, .audio-lists-btn svg, .destroy-btn {
    margin-left: 0!important;
}
.play-sounds svg, .loop-btn svg, .audio-lists-btn svg, .destroy-btn svg {
    width: 20px;
}
.react-jinke-music-player-main .music-player-panel .panel-content .player-content .audio-lists-btn {
    padding: 0;
}
.react-jinke-music-player-main .music-player-panel .panel-content .player-content .audio-lists-btn .audio-lists-icon svg {
    height: .75em;
}
.react-jinke-music-player-main .music-player-panel .panel-content .progress-bar-content .audio-main .current-time, .react-jinke-music-player-main .music-player-panel .panel-content .progress-bar-content .audio-main .duration {
    flex-basis: 0;
}
.progress-bar > div:nth-child(2) > div:nth-child(4) {
    transform: translateX(-50%) translateY(5%) !important;
}
.progress-load-bar {
	display: none;
}
.sound-operation > div:nth-child(4) {
    transform: translateX(-50%) translateY(5%) !important;
}
.react-jinke-music-player-main .music-player-panel .panel-content .player-content .play-sounds .sound-operation {
    width: 60px;
}
.rc-slider {
    border-radius: 0px;
    border: 1px solid black;
    padding: 3px 0!important;
}
.rc-slider .rc-slider-handle {
    box-shadow: none!important;
    border-radius: 0px;
    background-color: black!important;
    border: hidden!important;
}
.rc-slider[style*="left: 0%"] {
	transform: translateX(0) !important;
}
.rc-slider .rc-slider-track {
    display: none;
}
.react-jinke-music-player-main .rc-slider-rail, .react-jinke-music-player-main.light-theme .rc-slider-rail {
    background-color: white!important;
}
.react-jinke-music-player-main .music-player-panel .panel-content .player-content .play-sounds .sounds-icon {
    margin-right: 10px;
}
.lyric-btn {
    display: none!important;
}
button[data-testid="save-queue-button"] {
    display: none!important;
}
.react-jinke-music-player-main .music-player-panel .panel-content .player-content .audio-lists-btn {
    box-shadow: 0 0 0 0;
    margin: 0;
    margin-left: -8px;
    margin-right: -5px;
}
.audio-lists-btn:hover span,
.audio-lists-btn:hover svg {
    color: #a8fe40!important;
}
.react-jinke-music-player-main.light-theme .audio-lists-btn {
    background-color: white!important;
}
.react-jinke-music-player-main .music-player-panel .panel-content .player-content .audio-lists-btn .audio-lists-num {
    color: grey;
    margin-left: 5px;
    font-size: .7rem;
}
.react-jinke-music-player-main .music-player-panel .panel-content .player-content .hide-panel {
    margin-left: 2px;
}
.react-jinke-music-player-main .music-player-panel .panel-content .player-content .hide-panel svg {
    stroke-width: 15px;
    stroke: #fff;
    height: .8em;
}
@media screen and (max-width: 810px) {
    .react-jinke-music-player-main .music-player-panel .panel-content .player-content .play-sounds .sounds-icon {
        margin-left: 5px;
        margin-right: 0;
    }
    .react-jinke-music-player-main .music-player-panel .panel-content .player-content .loop-btn {
        margin-left: 5px;
    }
    .play-sounds svg, .loop-btn svg, .audio-lists-btn svg, .destroy-btn {
        margin-left: -3px!important;
    }
}
.panel-content li {
    flex-grow: 0;
}
.react-jinke-music-player .music-player-controller,
.react-jinke-music-player-main.light-theme .music-player-controller {
    border-radius: 5px;
    box-shadow: none;
}
.react-jinke-music-player .music-player-controller:hover,
.react-jinke-music-player .music-player-controller:has(+ .destroy-btn:hover) {
    border: 1px solid black;
}
.react-jinke-music-player .music-player-controller .controller-title,
.react-jinke-music-player .music-player-controller .music-player-controller-setting {
    display: none;
}
.react-jinke-music-player .music-player-controller.music-player-playing:before {
    animation: none;
    border: none;
}
@media screen and (max-width:767px) {
    .react-jinke-music-player .music-player .destroy-btn {
        right: 0;
    }
    .react-jinke-music-player-main .destroy-btn svg {
        font-size: 10px;
    }
}
.react-jinke-music-player-main svg {
    transition: none;
}
.react-jinke-music-player-main svg, .react-jinke-music-player-main.light-theme svg {
    color: black;
}
.react-jinke-music-player-main svg:active, .react-jinke-music-player-main svg:hover,
.react-jinke-music-player-main.light-theme svg:active, .react-jinke-music-player-main.light-theme svg:hover {
    color: #a8fe40;
}
.react-jinke-music-player-main .play-mode-title {
    font-family: monospace;
    background-color: white;
    color: black;
}
.react-jinke-music-player-mobile,
.react-jinke-music-player-main.light-theme .react-jinke-music-player-mobile {
    font-family: monospace;
    background-color: rgba(255, 255, 255, .9);
    color: black!important;
    justify-content: center;
    padding: 50px;
}
.react-jinke-music-player-mobile:before {
    content: " ";
    display: block;
    position: absolute;
    margin-left: auto;
    margin-right: auto;
    left: 0;
    right: 0;
    text-align: center;
    width: 90%;
    height: 700px;
    background-color: white;
    border: 1px solid black;
    z-index: -1;
	border-radius: 4px;
}
.react-jinke-music-player-mobile-header {
    align-items: start;
    margin-bottom: 4rem;
    justify-content: start;
}
.react-jinke-music-player-mobile-header-title {
    text-align: left;
    padding: 0;
}
.react-jinke-music-player-mobile-header-right {
    color: black;
}
.react-jinke-music-player-mobile > .group {
    flex: 0;
}
.react-jinke-music-player-mobile-cover,
.react-jinke-music-player-main.light-theme .react-jinke-music-player-mobile-cover {
    border-radius: 5px;
    box-shadow: none;
    animation: none;
    border: 1px solid black;
    margin: 0 auto 4rem auto;
    width: auto;
    height: auto;
}
.react-jinke-music-player-mobile-cover .cover {
    animation: none;
}
.react-jinke-music-player-mobile-progress .current-time {
    /* margin-right: 17px; */
}
.react-jinke-music-player-mobile-progress .current-time, .react-jinke-music-player-mobile-progress .duration {
    color: black!important;
}
.react-jinke-music-player-mobile-progress .rc-slider {
    height: 24px;
}
.react-jinke-music-player-mobile-progress .rc-slider-handle {
    border: 2px solid black;
    margin-top: -4px;
    height: 24px;
    width: 24px;
}
.react-jinke-music-player-mobile-toggle {
    margin-bottom: 1rem;
    padding: 2rem 0;
}
.react-jinke-music-player-mobile-operation .items .item svg {
    color: black!important;
    font-size: 2rem;
    width: 2rem;
}
.react-jinke-music-player-mobile-operation .items .item svg:hover,
.react-jinke-music-player-mobile-operation .items .item button:hover svg {
    color: #a8fe40!important;
}
.react-jinke-music-player-mobile-operation .items .item .MuiIconButton-root:hover {
    background-color: rgba(0, 0, 0, 0.0);
}
.react-jinke-music-player-mobile-operation .MuiButtonBase-root.Mui-disabled {
    cursor: pointer;
    pointer-events: auto;
}
.react-jinke-music-player-mobile-operation .items li:nth-child(5) svg {
    font-size: 1.4rem;
}
.react-jinke-music-player-mobile-operation .items li:nth-child(5) svg g path:nth-child(2) {
    stroke-width: .4px;
}
.react-jinke-music-player-mobile-operation .items li:nth-child(2), 
.react-jinke-music-player-mobile-operation .items li:nth-child(3) {
    display: none;
}
.react-jinke-music-player-mobile-play-model-tip {
   display: none;
}
.audio-lists-panel {
    overflow-y: scroll;
    scrollbar-width: none;
    border-radius: .625rem;
    bottom: 6.25rem;
}
.react-jinke-music-player-main.light-theme .audio-lists-panel {
    font-family: monospace;
    box-shadow: none;
    border: 1px solid black;
}
.react-jinke-music-player-main.light-theme .audio-lists-panel-header {
    text-shadow: none;
    border-bottom: 1px solid black;
}
.audio-lists-panel-header-line {
    width: 0;
}
.audio-lists-panel-header-close-btn:hover svg {
    animation: none;
}
.audio-lists-panel-content .audio-item,
.react-jinke-music-player-main.light-theme .audio-item {
    border-radius: 0px;
    margin: 0;
    border-bottom: none;
    box-shadow: none;
    transition: none;
}
.react-jinke-music-player-main.light-theme .audio-lists-panel .audio-lists-panel-content .audio-item:nth-child(2n+1) {
    background-color: white!important;
}
.react-jinke-music-player-main.light-theme .audio-lists-panel .audio-lists-panel-content .audio-item:nth-child(2n+1):hover {
    background-color: #fafafa!important;
}
.audio-lists-panel-content .audio-item .player-singer {
    width: unset;
    padding-right: 20px;
}
.audio-lists-panel-content .audio-item .player-delete:hover svg {
    color: #a8fe40!important;
    animation: none;
}
.react-jinke-music-player-main .audio-lists-panel-content .audio-item:active .group:not([class=".player-delete"]) svg, 
.react-jinke-music-player-main .audio-lists-panel-content .audio-item:hover .group:not([class=".player-delete"]) svg,
.react-jinke-music-player-main.light-theme .audio-item:active svg, 
.react-jinke-music-player-main.light-theme .audio-item:hover svg {
    color: black;
}
.audio-lists-panel-content .audio-item .player-delete {
    justify-content: center;
    width: 25px;
}
.audio-lists-panel-content .audio-item .player-delete svg {
    font-size: 20px;
}
.react-jinke-music-player-main.light-theme .audio-lists-panel .audio-item.playing, .react-jinke-music-player-main.light-theme .audio-lists-panel .audio-item.playing svg {
    color: #a8fe40!important;
}
.audio-lists-panel-content .audio-item .player-name,
.audio-lists-panel-content .audio-item .player-singer,
.react-jinke-music-player-main.light-theme .audio-item.playing .player-singer,
.react-jinke-music-player-main.light-theme .audio-lists-panel .audio-item.playing, .react-jinke-music-player-main.light-theme .audio-lists-panel .audio-item.playing .player-delete svg {
    color: black!important;
}
.audio-lists-panel-mobile {
    height: 750px !important;
    top: calc(100vh / 2 - 375px) !important; 
    width: 91% !important;
    margin: 0 auto;
} 
.audio-lists-panel-mobile .audio-lists-panel-content {
    height: auto!important;
}
.audio-lists-panel-content {
    scrollbar-width: none;
}
@keyframes fromOut {
  0% {
    transform:scale(1) translateZ(0)
  }
  to {
    transform:scale(1) translate3d(0,150%,0);
  }
}
`
export default stylesheet
