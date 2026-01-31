import React, { useCallback, useMemo } from 'react'
import { useSelector } from 'react-redux'
import { useMediaQuery } from '@material-ui/core'
import { ThemeProvider } from '@material-ui/core/styles'
import {
  createMuiTheme,
  useAuthState,
  useDataProvider,
  useTranslate,
} from 'react-admin'
import ReactGA from 'react-ga'
import { GlobalHotKeys } from 'react-hotkeys'
import ReactJkMusicPlayer from 'navidrome-music-player'
import 'navidrome-music-player/assets/index.css'
import useCurrentTheme from '../themes/useCurrentTheme'
import config from '../config'
import useStyle from './styles'
import AudioTitle from './AudioTitle'
import PlayerToolbar from './PlayerToolbar'
import { sendNotification } from '../utils'
import locale from './locale'
import { keyMap } from '../hotkeys'
import keyHandlers from './keyHandlers'
import { useScrobbling } from './hooks/useScrobbling'
import { useReplayGain } from './hooks/useReplayGain'
import { usePreloading } from './hooks/usePreloading'
import { usePlayerState } from './hooks/usePlayerState'
import { useAudioInstance } from './hooks/useAudioInstance'

/**
 * Player component for Navidrome music streaming application.
 * Renders an audio player with scrobbling, replay gain, preloading, and other features.
 *
 * @returns {JSX.Element} The rendered Player component.
 */
const Player = () => {
  const theme = useCurrentTheme()
  const translate = useTranslate()
  const playerTheme = theme.player?.theme || 'dark'
  const dataProvider = useDataProvider()
  const isDesktop = useMediaQuery('(min-width:810px)')
  const isMobilePlayer =
    /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(
      navigator.userAgent,
    )

  const { authenticated } = useAuthState()
  const showNotifications = useSelector(
    (state) => state.settings.notifications || false,
  )
  const gainInfo = useSelector((state) => state.replayGain)

  // Custom hooks for separated concerns
  const {
    playerState,
    dispatch,
    dispatchCurrentPlaying,
    dispatchSetPlayMode,
    dispatchSetVolume,
    dispatchSyncQueue,
    dispatchClearQueue,
  } = usePlayerState()

  const {
    startTime,
    setStartTime,
    scrobbled,
    onAudioProgress,
    onAudioPlayTrackChange,
    onAudioEnded,
  } = useScrobbling(playerState, dispatch, dataProvider)

  const { preloaded, preloadNextSong, resetPreloading } =
    usePreloading(playerState)

  const { audioInstance, setAudioInstance, onAudioPlay } =
    useAudioInstance(isMobilePlayer)

  const { context } = useReplayGain(audioInstance, playerState, gainInfo)

  const visible = authenticated && playerState.queue.length > 0
  const isRadio = playerState.current?.isRadio || false
  const classes = useStyle({
    isRadio,
    visible,
    enableCoverAnimation: config.enableCoverAnimation,
  })

  useEffect(() => {
    const handleBeforeUnload = (e) => {
      // Check there's a current track and is actually playing/not paused
      if (playerState.current?.uuid && audioInstance && !audioInstance.paused) {
        e.preventDefault()
        e.returnValue = '' // Chrome requires returnValue to be set
      }
    }

    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => window.removeEventListener('beforeunload', handleBeforeUnload)
  }, [playerState, audioInstance])

  const defaultOptions = useMemo(
    () => ({
      theme: playerTheme,
      bounds: 'body',
      playMode: playerState.mode,
      mode: 'full',
      loadAudioErrorPlayNext: false,
      autoPlayInitLoadPlayList: true,
      clearPriorAudioLists: false,
      showDestroy: true,
      showDownload: false,
      showLyric: true,
      showReload: false,
      toggleMode: !isDesktop,
      glassBg: false,
      showThemeSwitch: false,
      showMediaSession: true,
      restartCurrentOnPrev: true,
      quietUpdate: true,
      defaultPosition: {
        top: 300,
        left: 120,
      },
      volumeFade: { fadeIn: 200, fadeOut: 200 },
      renderAudioTitle: (audioInfo, isMobile) => (
        <AudioTitle
          audioInfo={audioInfo}
          gainInfo={gainInfo}
          isMobile={isMobile}
        />
      ),
      locale: locale(translate),
      sortableOptions: { delay: 200, delayOnTouchOnly: true },
    }),
    [playerTheme, playerState.mode, isDesktop, gainInfo, translate],
  )

  // Memoize expensive computations
  const audioLists = useMemo(
    () => playerState.queue.map((item) => item),
    [playerState.queue],
  )

  const currentTrack = playerState.current || {}

  const options = useMemo(() => {
    return {
      ...defaultOptions,
      audioLists,
      playIndex: playerState.playIndex,
      autoPlay: playerState.clear || playerState.playIndex === 0,
      clearPriorAudioLists: playerState.clear,
      extendsContent: (
        <PlayerToolbar
          id={currentTrack.trackId}
          isRadio={currentTrack.isRadio}
        />
      ),
      defaultVolume: isMobilePlayer ? 1 : playerState.volume,
      showMediaSession: !currentTrack.isRadio,
    }
  }, [
    defaultOptions,
    audioLists,
    playerState.playIndex,
    playerState.clear,
    playerState.volume,
    isMobilePlayer,
    currentTrack.trackId,
    currentTrack.isRadio,
  ])

  const onAudioListsChange = useCallback(
    (_, audioLists, audioInfo) => dispatchSyncQueue(audioInfo, audioLists),
    [dispatchSyncQueue],
  )

  const onAudioVolumeChange = useCallback(
    // sqrt to compensate for the logarithmic volume
    (volume) => dispatchSetVolume(volume),
    [dispatchSetVolume],
  )

  const handleAudioPlay = useCallback(
    (info) => {
      onAudioPlay(
        context,
        info,
        (info) => dispatchCurrentPlaying(info),
        showNotifications,
        sendNotification,
        startTime,
        setStartTime,
        resetPreloading,
        config,
        ReactGA,
      )
    },
    [
      onAudioPlay,
      context,
      dispatchCurrentPlaying,
      showNotifications,
      startTime,
      setStartTime,
      resetPreloading,
    ],
  )

  const onAudioPause = useCallback(
    (info) => dispatchCurrentPlaying(info),
    [dispatchCurrentPlaying],
  )

  const onCoverClick = useCallback((mode, audioLists, audioInfo) => {
    if (mode === 'full' && audioInfo?.song?.albumId) {
      window.location.href = `#/album/${audioInfo.song.albumId}/show`
    }
  }, [])

  const onBeforeDestroy = useCallback(() => {
    return new Promise((resolve, reject) => {
      dispatchClearQueue()
      reject()
    })
  }, [dispatchClearQueue])

  if (!visible) {
    document.title = 'Navidrome'
  }

  const handlers = useMemo(
    () => keyHandlers(audioInstance, playerState),
    [audioInstance, playerState],
  )

  return (
    <ThemeProvider theme={createMuiTheme(theme)}>
      <div role="region" aria-label="Audio Player" aria-live="polite">
        <ReactJkMusicPlayer
          {...options}
          className={classes.player}
          onAudioListsChange={onAudioListsChange}
          onAudioVolumeChange={onAudioVolumeChange}
          onAudioProgress={onAudioProgress}
          onAudioPlay={handleAudioPlay}
          onAudioPlayTrackChange={onAudioPlayTrackChange}
          onAudioPause={onAudioPause}
          onPlayModeChange={dispatchSetPlayMode}
          onAudioEnded={onAudioEnded}
          onCoverClick={onCoverClick}
          onBeforeDestroy={onBeforeDestroy}
          getAudioInstance={setAudioInstance}
          aria-label="Music Player"
        />
        <GlobalHotKeys
          handlers={handlers}
          keyMap={keyMap}
          allowChanges
          aria-hidden="true"
        />
      </div>
    </ThemeProvider>
  )
}

export { Player }
