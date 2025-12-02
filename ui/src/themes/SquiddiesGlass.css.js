const stylesheet = `

.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-handle {
  background: #c231ab
}
.react-jinke-music-player-main .music-player-panel .panel-content .rc-slider-track,
.react-jinke-music-player-mobile-progress .rc-slider-track {
  background: linear-gradient(to left, #c231ab, #380eff)
}

.react-jinke-music-player-mobile {
  background-color: #171717 !important;
}

.react-jinke-music-player-mobile-progress .rc-slider-handle {
  background: #c231ab;
  height: 20px;
  width: 20px;
  margin-top: -9px;
}

.react-jinke-music-player-main ::-webkit-scrollbar-thumb {
  background-color: #c231ab;
}

.react-jinke-music-player-pause-icon {
  background-color: #c231ab;
  border-radius: 50%;
  outline: auto;
  color: white;
}
.react-jinke-music-player-main .music-player-panel .panel-content .player-content {
  z-index: 99999;
}
.react-jinke-music-player-main .music-player-panel .panel-content .player-content .play-btn svg  {
  border-radius: 50%;
  outline: auto;
  color: white;
}
.react-jinke-music-player-main .music-player-panel .panel-content .player-content .play-btn svg:hover  {
  background-color: #c231ab;
  border-radius: 50%;
  outline: auto;
  color: white;
}

.react-jinke-music-player-main svg:hover {
  color: #c231ab;
}

.react-jinke-music-player .music-player-controller {
  color: #c231ab;
  border: 1px solid #e14ac2;
}

.react-jinke-music-player .music-player-controller.music-player-playing:before {
  border: 1px solid rgba(194, 49, 171, 0.3);
}

.react-jinke-music-player .music-player .destroy-btn {
  background-color: #c2c1c2;
  top: -7px;
  border-radius: 50%;
  display: flex;
}

.react-jinke-music-player .music-player .destroy-btn svg {
  font-size: 20px;
}

@media screen and (max-width: 767px) {
  .react-jinke-music-player .music-player .destroy-btn {
    right: -12px;
  }
}

.react-jinke-music-player-mobile-header-right {
  right: 0;
  top: 0;
}

@media screen and (max-width: 767px) {
  .react-jinke-music-player-main svg {
    font-size: 32px;
  }
}

@keyframes gradientFlow {
  0% { background-position: 0% 50%; }
  50% { background-position: 100% 50%; }
  100% { background-position: 0% 50%; }
}

.RaBulkActionsToolbar .MuiButton-label {
  color: white;
}

a[aria-current="page"] {
  color: #c231ab !important;
  font-weight: bold;
}

a[aria-current="page"] .MuiListItemIcon-root {
  color: #c231ab !important;
}

.panel-content {
  position: relative;
  overflow: hidden;
  background: linear-gradient(90deg, #311f2f, #0a0912, #2f0c28);
  background-size: 300% 300%;
  animation: gradientFlow 10s ease-in-out infinite;
}

/* Equalizer bars */
.panel-content::before {
  content: "";
  position: absolute;
  inset: 0;
  background: repeating-linear-gradient(
    90deg,
    rgba(255, 255, 255, 0.05) 0px,
    rgba(255, 255, 255, 0.05) 2px,
    transparent 1px,
    transparent 3px
  );
  animation: equalizer 1.8s infinite ease-in-out;
  filter: blur(1px);
  opacity: 0.5;
}

@keyframes backgroundFlow {
  0% {
    background-position: 0% 50%;
  }
  50% {
    background-position: 100% 50%;
  }
  100% {
    background-position: 0% 50%;
  }
}

/* Vertical movement, equalizer type */
@keyframes equalizer {
  0%, 100% {
    transform: scaleY(1);
    opacity: 0.2;
  }
  25% {
    transform: scaleY(1.4);
    opacity: 0.9;
  }
  50% {
    transform: scaleY(0.7);
    opacity: 0.2;
  }
  75% {
    transform: scaleY(1.2);
    opacity: 0.8;
  }
}

@keyframes pulse {
  0% { opacity: 0.5; }
  100% { opacity: 1; }
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
`

export default stylesheet
