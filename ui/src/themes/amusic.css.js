const stylesheet = `
.react-jinke-music-player-main svg:active, .react-jinke-music-player-main svg:hover {
    color: #ff0436
}
.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle, 
.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-track {
    background-color: #ff4e6b
}
.react-jinke-music-player-main ::-webkit-scrollbar-thumb {
    background-color: #ff4e6b
}
.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle:active {
    box-shadow: 0 0 2px #ff4e6b
}
.audio-lists-panel-content .audio-item.playing,
.react-jinke-music-player-main .audio-item.playing svg,
.react-jinke-music-player-main .group player-delete {
    color: #ff4e6b
}
.audio-lists-panel-content .audio-item:hover,{
    color: #ff0436
}
.react-jinke-music-player-main .audio-item.playing .player-singer {
    color: #ff4e6b !important
}
.react-jinke-music-player-main .lyric-btn,
.react-jinke-music-player-main .lyric-btn-active svg{
    color: #ff4e6b !important
}
.react-jinke-music-player-main .lyric-btn-active {
    color: #ff0436 !important
}
.react-jinke-music-player-main .loading svg {
    color: #ff4e6b !important
}
.react-jinke-music-player-main .music-player-lyric{
    color: #ff4e6b !important;
	text-shadow: -1px -1px 0 #000, 1px -1px 0 #000, -1px 1px 0 #000, 1px 1px 0 #000
}
.react-jinke-music-player-main .music-player-panel{
    background-color: #1f1f1f;
	border: 1px solid rgba(255, 255, 255, 0.12);
}
.audio-lists-panel{
    background-color: #1f1f1f;
	border: 1px solid rgba(255, 255, 255, 0.12);
	border-radius: 6px 6px 0 0;
}
.react-jinke-music-player-main .music-player-panel .panel-content div.img-rotate{
    border-radius: 6px;
	animation-duration: 0s !important
}
.react-jinke-music-player-main .songTitle{
    color: #ddd
}
`
export default stylesheet
