import React, { useCallback, useMemo, useState } from 'react'
import ReactGA from 'react-ga'
import { useDispatch, useSelector } from 'react-redux'
import { Link } from 'react-router-dom'
import { useAuthState, useDataProvider, useTranslate } from 'react-admin'
import ReactJkMusicPlayer from 'react-jinke-music-player'
import 'react-jinke-music-player/assets/index.css'
import {
  createMuiTheme,
  makeStyles,
  ThemeProvider,
} from '@material-ui/core/styles'
import { useMediaQuery } from '@material-ui/core'
import { GlobalHotKeys } from 'react-hotkeys'
import clsx from 'clsx'
import subsonic from '../subsonic'
import {
  scrobble,
  syncQueue,
  currentPlaying,
  setVolume,
  clearQueue,
} from '../actions'
import config from '../config'
import PlayerToolbar from './PlayerToolbar'
import { sendNotification } from '../utils'
import { keyMap } from '../hotkeys'
import useCurrentTheme from '../themes/useCurrentTheme'
import { QualityInfo } from '../common'

const useStyle = makeStyles(
  (theme) => ({
    audioTitle: {
      textDecoration: 'none',
      color: theme.palette.primary.dark,
    },
    songTitle: {
      fontWeight: 'bold',
      '&:hover + $qualityInfo': {
        opacity: 1,
      },
    },
    songInfo: {
      display: 'block',
    },
    qualityInfo: {
      marginTop: '-4px',
      opacity: 0,
      transition: 'all 500ms ease-out',
    },
    player: {
      display: (props) => (props.visible ? 'block' : 'none'),
      '@media screen and (max-width:810px)': {
        '& .sound-operation': {
          display: 'none',
        },
      },
      '& .progress-bar-content': {
        display: 'flex',
        flexDirection: 'column',
      },
      '& .play-mode-title': {
        'pointer-events': 'none',
      },
    },
    artistAlbum: {
      marginTop: '2px',
    },
  }),
  { name: 'NDAudioPlayer' }
)

let audioInstance = null

const AudioTitle = React.memo(({ audioInfo, isMobile }) => {
  const classes = useStyle()
  const className = classes.audioTitle
  const isDesktop = useMediaQuery('(min-width:810px)')

  if (!audioInfo.name) {
    return ''
  }

  const qi = { suffix: audioInfo.suffix, bitRate: audioInfo.bitRate }

  return (
    <Link to={`/album/${audioInfo.albumId}/show`} className={className}>
      <span>
        <span className={clsx(classes.songTitle, 'songTitle')}>
          {audioInfo.name}
        </span>
        {isDesktop && (
          <QualityInfo record={qi} className={classes.qualityInfo} />
        )}
      </span>
      {!isMobile && (
        <div className={classes.artistAlbum}>
          <span className={clsx(classes.songInfo, 'songInfo')}>
            {`${audioInfo.singer} - ${audioInfo.album}`}
          </span>
        </div>
      )}
    </Link>
  )
})

const Player = () => {
  const translate = useTranslate()
  const theme = useCurrentTheme()
  const playerTheme = (theme.player && theme.player.theme) || 'dark'
  const dataProvider = useDataProvider()
  const dispatch = useDispatch()
  const queue = useSelector((state) => state.queue)
  const { authenticated } = useAuthState()
  const showNotifications = useSelector(
    (state) => state.settings.notifications || false
  )

  const visible = authenticated && queue.queue.length > 0
  const classes = useStyle({ visible })
  // Match the medium breakpoint defined in the material-ui theme
  // See https://material-ui.com/customization/breakpoints/#breakpoints
  const isDesktop = useMediaQuery('(min-width:810px)')
  const [startTime, setStartTime] = useState(null)

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

  const defaultOptions = useMemo(
    () => ({
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
      toggleMode: !isDesktop,
      glassBg: false,
      showThemeSwitch: false,
      showMediaSession: true,
      restartCurrentOnPrev: true,
      defaultPosition: {
        top: 300,
        left: 120,
      },
      volumeFade: { fadeIn: 200, fadeOut: 200 },
      renderAudioTitle: (audioInfo, isMobile) => (
        <AudioTitle audioInfo={audioInfo} isMobile={isMobile} />
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
    }),
    [isDesktop, playerTheme, translate]
  )

  const options = useMemo(() => {
    const current = queue.current || {}
    return {
      ...defaultOptions,
      clearPriorAudioLists: queue.clear,
      autoPlay: queue.clear || queue.playIndex === 0,
      playIndex: queue.playIndex,
      audioLists: queue.queue.map((item) => item),
      extendsContent: <PlayerToolbar id={current.trackId} />,
      defaultVolume: queue.volume,
    }
  }, [queue, defaultOptions])

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
      if (isNaN(info.duration) || (progress < 50 && info.currentTime < 240)) {
        return
      }

      const item = queue.queue.find((item) => item.trackId === info.trackId)
      if (item && !item.scrobbled) {
        dispatch(scrobble(info.trackId, true))
        subsonic.scrobble(info.trackId, true, startTime)
      }
    },
    [dispatch, queue.queue, startTime]
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
        document.title = `${info.name} - ${info.singer} - Navidrome`
        dispatch(scrobble(info.trackId, false))
        subsonic.nowPlaying(info.trackId)
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
    <ThemeProvider theme={createMuiTheme(theme)}>
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
      />
      <GlobalHotKeys handlers={keyHandlers} keyMap={keyMap} allowChanges />
    </ThemeProvider>
  )
}

export { Player }
