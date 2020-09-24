import React, { useCallback, useMemo } from 'react'
import ReactGA from 'react-ga'
import { useDispatch, useSelector } from 'react-redux'
import { Link } from 'react-router-dom'
import { useAuthState, useDataProvider, useTranslate } from 'react-admin'
import ReactJkMusicPlayer from 'react-jinke-music-player'
import 'react-jinke-music-player/assets/index.css'
import subsonic from '../subsonic'
import {
  scrobble,
  syncQueue,
  currentPlaying,
  setVolume,
  clearQueue,
} from './queue'
import themes from '../themes'
import { makeStyles } from '@material-ui/core/styles'
import config from '../config'

const useStyle = makeStyles((theme) => ({
  audioTitle: {
    textDecoration: 'none',
    color: theme.palette.primary.light,
  },
}))

const Player = () => {
  const classes = useStyle()
  const translate = useTranslate()
  const currentTheme = useSelector((state) => state.theme)
  const theme = themes[currentTheme] || themes.DarkTheme
  const playerTheme = (theme.player && theme.player.theme) || 'dark'

  const audioTitle = useCallback(
    (audioInfo) => (
      <Link
        to={`/album/${audioInfo.albumId}/show`}
        className={classes.audioTitle}
      >
        {audioInfo.name ? `${audioInfo.name} - ${audioInfo.singer}` : ''}
      </Link>
    ),
    [classes.audioTitle]
  )

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
      audioTitle: audioTitle,
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

  const dataProvider = useDataProvider()
  const dispatch = useDispatch()
  const queue = useSelector((state) => state.queue)
  const { authenticated } = useAuthState()

  const options = useMemo(() => {
    return {
      ...defaultOptions,
      clearPriorAudioLists: queue.clear,
      autoPlay: queue.clear || queue.playIndex === 0,
      playIndex: queue.playIndex,
      audioLists: queue.queue.map((item) => item),
      defaultVolume: queue.volume,
    }
  }, [queue.clear, queue.queue, queue.volume, queue.playIndex, defaultOptions])

  const OnAudioListsChange = useCallback(
    (currentPlayIndex, audioLists) => {
      dispatch(syncQueue(currentPlayIndex, audioLists))
    },
    [dispatch]
  )

  const OnAudioProgress = useCallback(
    (info) => {
      if (info.ended) {
        document.title = 'Navidrome'
      }
      const progress = (info.currentTime / info.duration) * 100
      if (isNaN(info.duration) || progress < 90) {
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
    (volume) => dispatch(setVolume(volume)),
    [dispatch]
  )

  const OnAudioPlay = useCallback(
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
      }
    },
    [dispatch]
  )

  const onAudioPause = useCallback(
    (info) => {
      dispatch(currentPlaying(info))
    },
    [dispatch]
  )

  const onAudioEnded = useCallback(
    (currentPlayId, audioLists, info) => {
      dispatch(currentPlaying(info))
      dataProvider.getOne('keepalive', { id: info.trackId })
    },
    [dispatch, dataProvider]
  )

  const onBeforeDestroy = useCallback(() => {
    return new Promise((resolve, reject) => {
      dispatch(clearQueue())
      reject()
    })
  }, [dispatch])

  if (authenticated && options.audioLists.length > 0) {
    return (
      <ReactJkMusicPlayer
        {...options}
        quietUpdate
        onAudioListsChange={OnAudioListsChange}
        onAudioProgress={OnAudioProgress}
        onAudioPlay={OnAudioPlay}
        onAudioPause={onAudioPause}
        onAudioEnded={onAudioEnded}
        onAudioVolumeChange={onAudioVolumeChange}
        onBeforeDestroy={onBeforeDestroy}
      />
    )
  }
  document.title = 'Navidrome'
  return null
}

export default Player
