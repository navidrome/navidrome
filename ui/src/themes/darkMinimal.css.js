export default `

.music-player-panel {
    background-color: #121212 !important;
}

.react-jinke-music-player-main .audio-lists-panel {
    background-color: #121212 !important;
    box-shadow: none !important;
    border: solid #FFFFFF1F !important;
    border-width: 1px 1px 0px 1px !important;
    border-radius: 10px 10px 0 0;
}

.react-jinke-music-player-main .music-player-panel .panel-content .progress-bar-content .audio-main .current-time {
    flex-basis: unset !important;
}

.progress-bar-content {
    flex-direction: row !important;
    align-items: center !important;
    border-right: solid #FFFFFF1F 1px !important;
    // padding-right: 0 !important;
}

.progress-bar-content .audio-title {
    width: auto !important;
    padding-right: 1rem !important;
}

.progress-bar-content .audio-main {
    margin-top: 0px !important;
}

.panel-content .player-content {
    padding-left: 0% !important;
}

/* --- growing play queue --- */
.audio-lists-panel {
    height: unset !important;
}
.audio-lists-panel-content {
    max-height: calc(100vh - 215px);
    height: unset !important;
}
`;
