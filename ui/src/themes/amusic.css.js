const stylesheet = `
.react-jinke-music-player-main svg:active, .react-jinke-music-player-main svg:hover {
    color: #D60017
}
.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle, 
.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-track {
    background-color: #ff4e6b
}
.react-jinke-music-player-main ::-webkit-scrollbar-thumb,
.react-jinke-music-player-mobile-progress .rc-slider-handle, 
.react-jinke-music-player-mobile-progress .rc-slider-track {
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
.audio-lists-panel-content .audio-item:hover,
.audio-lists-panel-content .audio-item:hover svg
.audio-lists-panel-content .audio-item:active .group:not([class=".player-delete"]) svg, .audio-lists-panel-content .audio-item:hover .group:not([class=".player-delete"]) svg{
    color: #D60017
}
.react-jinke-music-player-main .audio-item.playing .player-singer {
    color: #ff4e6b !important
}
.react-jinke-music-player-main .lyric-btn,
.react-jinke-music-player-main .lyric-btn-active svg{
    color: #ff4e6b !important
}
.react-jinke-music-player-main .lyric-btn-active {
    color: #D60017 !important
}
.react-jinke-music-player-main .loading svg {
    color: #ff4e6b !important
}
.react-jinke-music-player .music-player-controller .music-player-controller-setting{
    background: #ff4e6b4d
}
.react-jinke-music-player-main .music-player-lyric{
    color: #ff4e6b !important;
	text-shadow: -1px -1px 0 #000, 1px -1px 0 #000, -1px 1px 0 #000, 1px 1px 0 #000
}
.react-jinke-music-player-main .music-player-panel,
.react-jinke-music-player-mobile,
.ril__outer{
    background-color: #1a1a1a;
	border: 1px solid #fff1;
}
.ril__toolbarItem{
	font-size: 100%;
	color: #eee
}
.audio-lists-panel,
.ril__toolbar{
    background-color: #1f1f1f;
	border: 1px solid #fff1;
	border-radius: 6px 6px 0 0;
}
.react-jinke-music-player-main .music-player-panel .panel-content .img-rotate,
.react-jinke-music-player-mobile .react-jinke-music-player-mobile-cover img.cover,
.react-jinke-music-player-mobile-cover {
    border-radius: 6px !important;
	animation-duration: 0s !important
}
.react-jinke-music-player-main .music-player-panel .panel-content .img-content{
	width: 60px;
	height: 60px
}
.react-jinke-music-player-main .songTitle{
    color: #eee
}
.react-jinke-music-player .music-player-controller{
    color: #ff4e6b
}
.audio-lists-panel-mobile .audio-item:not(.audio-lists-panel-sortable-highlight-bg){
	background: unset
}
.lastfm-icon, 
.musicbrainz-icon{
	color: #eee
}
`
export default stylesheet
