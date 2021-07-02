import React, { useCallback, useMemo, useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { createMuiTheme, ThemeProvider } from '@material-ui/core/styles'
import { useMediaQuery } from '@material-ui/core'
import { useAuthState, useDataProvider, useTranslate } from 'react-admin'
import ReactGA from 'react-ga'
import { GlobalHotKeys } from 'react-hotkeys'
import ReactJkMusicPlayer from 'react-jinke-music-player'
import 'react-jinke-music-player/assets/index.css'
import useCurrentTheme from '../themes/useCurrentTheme'
import config from '../config'
import useStyle from './styles'
import AudioTitle from './AudioTitle'
import { clearQueue, currentPlaying, setVolume, syncQueue } from '../actions'
import PlayerToolbar from './PlayerToolbar'
import { sendNotification } from '../utils'
import subsonic from '../subsonic'
import locale from './locale'
import { keyMap } from '../hotkeys'
import keyHandlers from './keyHandlers'

const Player = () => {
  const theme = useCurrentTheme()
  const translate = useTranslate()
  const playerTheme = theme.player?.theme || 'dark'
  const dataProvider = useDataProvider()
  const playerState = useSelector((state) => state.player)
  const dispatch = useDispatch()
  const [startTime, setStartTime] = useState(null)
  const [scrobbled, setScrobbled] = useState(false)
  const [audioInstance, setAudioInstance] = useState(null)
  const isDesktop = useMediaQuery('(min-width:810px)')
  const { authenticated } = useAuthState()
  const visible = authenticated && playerState.queue.length > 0
  const classes = useStyle({
    visible,
    enableCoverAnimation: config.enableCoverAnimation,
  })
  const showNotifications = useSelector(
    (state) => state.settings.notifications || false
  )

  const defaultOptions = useMemo(
    () => ({
      theme: playerTheme,
      bounds: 'body',
      mode: 'full',
      loadAudioErrorPlayNext: false,
      clearPriorAudioLists: false,
      showDestroy: true,
      showDownload: false,
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
        <AudioTitle audioInfo={audioInfo} isMobile={isMobile} />
      ),
      locale: locale(translate),
    }),
    [isDesktop, playerTheme, translate]
  )

  const options = useMemo(() => {
    const current = playerState.current || {}
    return {
      ...defaultOptions,
      audioLists: playerState.queue.map((item) => item),
      playIndex: playerState.playIndex,
      clearPriorAudioLists: playerState.clear,
      extendsContent: <PlayerToolbar id={current.trackId} />,
      defaultVolume: playerState.volume,
    }
  }, [playerState, defaultOptions])

  const onAudioListsChange = useCallback(
    (currentPlayIndex, audioLists) =>
      dispatch(syncQueue(currentPlayIndex, audioLists)),
    [dispatch]
  )

  const onAudioProgress = useCallback(
    (info) => {
      if (info.ended) {
        document.title = 'Navidrome'
      }

      const progress = (info.currentTime / info.duration) * 100
      if (isNaN(info.duration) || (progress < 50 && info.currentTime < 240)) {
        return
      }

      if (!scrobbled) {
        subsonic.scrobble(info.trackId, true, startTime)
        setScrobbled(true)
      }
    },
    [startTime, scrobbled]
  )

  const onAudioVolumeChange = useCallback(
    // sqrt to compensate for the logarithmic volume
    (volume) => dispatch(setVolume(Math.sqrt(volume))),
    [dispatch]
  )

  const onAudioPlay = useCallback(
    (info) => {
      dispatch(currentPlaying(info))
      setStartTime(Date.now())
      if (info.duration) {
        const song = info.song
        document.title = `${song.title} - ${song.artist} - Navidrome`
        subsonic.nowPlaying(info.trackId)
        setScrobbled(false)
        if (config.gaTrackingId) {
          ReactGA.event({
            category: 'Player',
            action: 'Play song',
            label: `${song.title} - ${song.artist}`,
          })
        }
        if (showNotifications) {
          sendNotification(
            song.title,
            `${song.artist} - ${song.album}`,
            info.cover
          )
        }
      }
    },
    [dispatch, showNotifications]
  )

  const onAudioPause = useCallback(
    (info) => dispatch(currentPlaying(info)),
    [dispatch]
  )

  const onAudioEnded = useCallback(
    (currentPlayId, audioLists, info) => {
      dispatch(currentPlaying(info))
      dataProvider
        .getOne('keepalive', { id: info.trackId })
        .catch((e) => console.log('Keepalive error:', e))
    },
    [dispatch, dataProvider]
  )

  const onCoverClick = useCallback((mode, audioLists, audioInfo) => {
    if (mode === 'full' && audioInfo?.song?.albumId) {
      window.location.href = `#/album/${audioInfo.song.albumId}/show`
    }
  }, [])

  const onBeforeDestroy = useCallback(() => {
    return new Promise((resolve, reject) => {
      dispatch(clearQueue())
      reject()
    })
  }, [dispatch])

  if (!visible) {
    document.title = 'Navidrome'
  }

  const handlers = useMemo(
    () => keyHandlers(audioInstance, playerState),
    [audioInstance, playerState]
  )

  return (
    <ThemeProvider theme={createMuiTheme(theme)}>
      <ReactJkMusicPlayer
        {...options}
        className={classes.player}
        onAudioListsChange={onAudioListsChange}
        onAudioVolumeChange={onAudioVolumeChange}
        onAudioProgress={onAudioProgress}
        onAudioPlay={onAudioPlay}
        onAudioPause={onAudioPause}
        onAudioEnded={onAudioEnded}
        onCoverClick={onCoverClick}
        onBeforeDestroy={onBeforeDestroy}
        getAudioInstance={setAudioInstance}
      />
      <GlobalHotKeys handlers={handlers} keyMap={keyMap} allowChanges />
    </ThemeProvider>
  )
}

export { Player }
