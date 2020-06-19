import React from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { Link } from 'react-router-dom'
import { useAuthState, useDataProvider, useTranslate } from 'react-admin'
import ReactJkMusicPlayer from 'react-jinke-music-player'
import 'react-jinke-music-player/assets/index.css'
import subsonic from '../subsonic'
import { scrobble, syncQueue, currentPlaying } from './queue'
import themes from '../themes'
import { makeStyles } from '@material-ui/core/styles'

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

  const audioTitle = (audioInfo) => (
    <Link
      to={`/album/${audioInfo.albumId}/show`}
      className={classes.audioTitle}
    >
      {`${audioInfo.name} - ${audioInfo.singer}`}
    </Link>
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
    showDestroy: false,
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

  const addQueueToOptions = (queue) => {
    return {
      ...defaultOptions,
      autoPlay: queue.playing,
      clearPriorAudioLists: queue.clear,
      audioLists: queue.queue.map((item) => item),
    }
  }

  const dataProvider = useDataProvider()
  const dispatch = useDispatch()
  const queue = useSelector((state) => state.queue)
  const options = addQueueToOptions(queue)
  const { authenticated } = useAuthState()

  const OnAudioListsChange = (currentPlayIndex, audioLists) => {
    dispatch(syncQueue(currentPlayIndex, audioLists))
  }

  const OnAudioProgress = (info) => {
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
  }

  const OnAudioPlay = (info) => {
    dispatch(currentPlaying(info))
    if (info.duration) {
      document.title = `${info.name} - ${info.singer} - Navidrome`
      dispatch(scrobble(info.trackId, false))
      subsonic.scrobble(info.trackId, false)
    }
  }

  const onAudioPause = (info) => {
    dispatch(currentPlaying(info))
  }

  const onAudioEnded = (currentPlayId, audioLists, info) => {
    dispatch(currentPlaying(info))
    dataProvider.getOne('keepalive', { id: info.trackId })
  }

  if (authenticated && options.audioLists.length > 0) {
    return (
      <ReactJkMusicPlayer
        {...options}
        onAudioListsChange={OnAudioListsChange}
        onAudioProgress={OnAudioProgress}
        onAudioPlay={OnAudioPlay}
        onAudioPause={onAudioPause}
        onAudioEnded={onAudioEnded}
      />
    )
  }
  document.title = 'Navidrome'
  return null
}

export default Player
