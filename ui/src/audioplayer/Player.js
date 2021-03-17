import React, { useCallback, useMemo } from 'react'
import ReactGA from 'react-ga'
import { useDispatch, useSelector } from 'react-redux'
import { Link } from 'react-router-dom'
import { useAuthState, useDataProvider, useTranslate } from 'react-admin'
import ReactJkMusicPlayer from 'react-jinke-music-player'
import 'react-jinke-music-player/assets/index.css'
import { makeStyles } from '@material-ui/core/styles'
import { GlobalHotKeys } from 'react-hotkeys'
import subsonic from '../subsonic'
import {
  scrobble,
  syncQueue,
  currentPlaying,
  setVolume,
  clearQueue,
} from '../actions'
import themes from '../themes'
import config from '../config'
import PlayerToolbar from './PlayerToolbar'
import { sendNotification, baseUrl } from '../utils'
import { resolution } from '../audioplayer/resolutionMap.js'
import { useTheme } from '@material-ui/core/styles'
import { keyMap } from '../hotkeys'

const useStyle = makeStyles((theme) => ({
  audioTitle: {
    textDecoration: 'none',
    color: theme.palette.primary.light,
    '&.songTitle': {
      fontWeight: 'bold',
    },
  },
  player: {
    display: (props) => (props.visible ? 'block' : 'none'),
  },
}))

let audioInstance = null

const AudioTitle = React.memo(({ audioInfo, isMobile, className }) => {
  if (!audioInfo.name) {
    return ''
  }

  return (
    <Link to={`/album/${audioInfo.albumId}/show`} className={className}>
      <span className={`${className} songTitle`}>{audioInfo.name}</span>
      {!isMobile && (
        <>
          <br />
          <span className={`${className} songInfo`}>
            {`${audioInfo.singer} - ${audioInfo.album}`}
          </span>
        </>
      )}
    </Link>
  )
})

const Player = () => {
  const translate = useTranslate()
  const currentTheme = useSelector((state) => state.theme)
  const theme = themes[currentTheme] || themes.DarkTheme
  const playerTheme = (theme.player && theme.player.theme) || 'dark'
  const dataProvider = useDataProvider()
  const dispatch = useDispatch()
  const queue = useSelector((state) => state.queue)
  const current = queue.current || {}
  const { authenticated } = useAuthState()
  const materialTheme = useTheme()
  const showNotifications = useSelector(
    (state) => state.settings.notifications || false
  )

  const breakpoint = materialTheme.breakpoints.width(
    resolution[useSelector((state) => state.settings.resolution)]
  )
  const visible = authenticated && queue.queue.length > 0
  const classes = useStyle({ visible })

  const nextSong = useCallback(() => {
    const idx = queue.queue.findIndex(
      (item) => item.uuid === queue.current.uuid
    )
    return idx !== null ? queue.queue[idx + 1] : null
  }, [queue])

  const prevSong = useCallback(() => {
    const idx = queue.queue.findIndex(
      (item) => item.uuid === queue.current.uuid
    )
    return idx !== null ? queue.queue[idx - 1] : null
  }, [queue])

  const keyHandlers = {
    TOGGLE_PLAY: (e) => {
      e.preventDefault()
      audioInstance && audioInstance.togglePlay()
    },
    VOL_UP: () =>
      (audioInstance.volume = Math.min(1, audioInstance.volume + 0.1)),
    VOL_DOWN: () =>
      (audioInstance.volume = Math.max(0, audioInstance.volume - 0.1)),
    PREV_SONG: useCallback(
      (e) => {
        if (!e.metaKey && prevSong()) audioInstance && audioInstance.playPrev()
      },
      [prevSong]
    ),
    NEXT_SONG: useCallback(
      (e) => {
        if (!e.metaKey && nextSong()) audioInstance && audioInstance.playNext()
      },
      [nextSong]
    ),
  }

  const defaultOptions = {
    theme: playerTheme,
    bounds: 'body',
    mode: 'full',
    autoPlay: false,
    preload: true,
    autoPlayInitLoadPlayList: true,
    loadAudioErrorPlayNext: false,
    clearPriorAudioLists: false,
    showDestroy: true,
    showDownload: false,
    showReload: false,
    glassBg: false,
    showThemeSwitch: false,
    showMediaSession: true,
    defaultPosition: {
      top: 300,
      left: 120,
    },
    volumeFade: { fadeIn: 200, fadeOut: 200 },
    renderAudioTitle: (audioInfo, isMobile) => (
      <AudioTitle
        audioInfo={audioInfo}
        isMobile={isMobile}
        className={classes.audioTitle}
      />
    ),
    locale: {
      playListsText: translate('player.playListsText'),
      openText: translate('player.openText'),
      closeText: translate('player.closeText'),
      notContentText: translate('player.notContentText'),
      clickToPlayText: translate('player.clickToPlayText'),
      clickToPauseText: translate('player.clickToPauseText'),
      nextTrackText: translate('player.nextTrackText'),
      previousTrackText: translate('player.previousTrackText'),
      reloadText: translate('player.reloadText'),
      volumeText: translate('player.volumeText'),
      toggleLyricText: translate('player.toggleLyricText'),
      toggleMiniModeText: translate('player.toggleMiniModeText'),
      destroyText: translate('player.destroyText'),
      downloadText: translate('player.downloadText'),
      removeAudioListsText: translate('player.removeAudioListsText'),
      clickToDeleteText: (name) =>
        translate('player.clickToDeleteText', { name }),
      emptyLyricText: translate('player.emptyLyricText'),
      playModeText: {
        order: translate('player.playModeText.order'),
        orderLoop: translate('player.playModeText.orderLoop'),
        singleLoop: translate('player.playModeText.singleLoop'),
        shufflePlay: translate('player.playModeText.shufflePlay'),
      },
    },
  }

  const options = useMemo(() => {
    return {
      ...defaultOptions,
      clearPriorAudioLists: queue.clear,
      autoPlay: queue.clear || queue.playIndex === 0,
      playIndex: queue.playIndex,
      audioLists: queue.queue.map((item) => item),
      extendsContent: <PlayerToolbar id={current.trackId} />,
      defaultVolume: queue.volume,
    }
  }, [
    queue.clear,
    queue.queue,
    queue.volume,
    queue.playIndex,
    current,
    defaultOptions,
  ])

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

      // See https://www.last.fm/api/scrobbling#when-is-a-scrobble-a-scrobble
      const progress = (info.currentTime / info.duration) * 100
      if (
        isNaN(info.duration) ||
        info.duration < 30 ||
        (progress < 50 && info.currentTime < 240)
      ) {
        return
      }

      const item = queue.queue.find((item) => item.trackId === info.trackId)
      if (item && !item.scrobbled) {
        dispatch(scrobble(info.trackId, true))
        subsonic.scrobble(info.trackId, true)
      }
    },
    [dispatch, queue.queue]
  )

  const onAudioVolumeChange = useCallback(
    // sqrt to compensate for the logarithmic volume
    (volume) => dispatch(setVolume(Math.sqrt(volume))),
    [dispatch]
  )

  const onAudioPlay = useCallback(
    (info) => {
      dispatch(currentPlaying(info))
      if (info.duration) {
        document.title = `${info.name} - ${info.singer} - Navidrome`
        dispatch(scrobble(info.trackId, false))
        subsonic.scrobble(info.trackId, false)
        if (config.gaTrackingId) {
          ReactGA.event({
            category: 'Player',
            action: 'Play song',
            label: `${info.name} - ${info.singer}`,
          })
        }
        if (showNotifications) {
          sendNotification(
            info.name,
            `${info.singer} - ${info.album}`,
            baseUrl(info.cover)
          )
        }
      }
    },
    [dispatch, showNotifications]
  )

  const onAudioPause = useCallback((info) => dispatch(currentPlaying(info)), [
    dispatch,
  ])

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
    if (mode === 'full') {
      window.location.href = `#/album/${audioInfo.albumId}/show`
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

  return (
    <>
      <ReactJkMusicPlayer
        {...options}
        quietUpdate
        className={classes.player}
        onAudioListsChange={onAudioListsChange}
        onAudioProgress={onAudioProgress}
        onAudioPlay={onAudioPlay}
        onAudioPause={onAudioPause}
        onAudioEnded={onAudioEnded}
        onAudioVolumeChange={onAudioVolumeChange}
        onCoverClick={onCoverClick}
        onBeforeDestroy={onBeforeDestroy}
        getAudioInstance={(instance) => {
          audioInstance = instance
        }}
        mobileMediaQuery={`(max-width:${breakpoint}px)`}
      />
      <GlobalHotKeys handlers={keyHandlers} keyMap={keyMap} allowChanges />
    </>
  )
}

export { Player }
