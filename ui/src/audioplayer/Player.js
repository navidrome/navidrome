import React from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { useAuthState, useDataProvider, useTranslate } from 'react-admin'
import ReactJkMusicPlayer from 'react-jinke-music-player'
import 'react-jinke-music-player/assets/index.css'
import subsonic from '../subsonic'
import { scrobbled, syncQueue } from './queue'
import themes from '../themes'

const Player = () => {
  const translate = useTranslate()
  const currentTheme = useSelector((state) => state.theme)
  const theme = themes[currentTheme] || themes.DarkTheme
  const playerTheme = (theme.player && theme.player.theme) || 'dark'

  const defaultOptions = {
    theme: playerTheme,
    bounds: 'body',
    mode: 'full',
    autoPlay: true,
    preload: true,
    autoPlayInitLoadPlayList: true,
    clearPriorAudioLists: false,
    showDestroy: false,
    showDownload: false,
    showReload: false,
    glassBg: false,
    showThemeSwitch: false,
    showMediaSession: true,
    defaultPosition: {
      top: 300,
      left: 120
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
      controllerTitle: translate('player.controllerTitle'),
      clickToDeleteText: (name) =>
        translate('player.clickToDeleteText', { name }),
      emptyLyricText: translate('player.emptyLyricText'),
      playModeText: {
        order: translate('player.playModeText.order'),
        orderLoop: translate('player.playModeText.orderLoop'),
        singleLoop: translate('player.playModeText.singleLoop'),
        shufflePlay: translate('player.playModeText.shufflePlay')
      }
    }
  }

  const addQueueToOptions = (queue) => {
    return {
      ...defaultOptions,
      autoPlay: true,
      clearPriorAudioLists: queue.clear,
      audioLists: queue.queue.map((item) => item)
    }
  }

  const dataProvider = useDataProvider()
  const dispatch = useDispatch()
  const queue = useSelector((state) => state.queue)
  const options = addQueueToOptions(queue)
  const { authenticated } = useAuthState()

  const OnAudioListsChange = (currentPlayIndex, audioLists) => {
    dispatch(syncQueue(audioLists))
  }

  const OnAudioProgress = (info) => {
    const progress = (info.currentTime / info.duration) * 100
    if (isNaN(info.duration) || progress < 90) {
      return
    }
    const item = queue.queue.find((item) => item.trackId === info.trackId)
    if (item && !item.scrobbled) {
      dispatch(scrobbled(info.trackId))
      subsonic.scrobble(info.trackId, true)
    }
  }

  const OnAudioPlay = (info) => {
    if (info.duration) {
      subsonic.scrobble(info.trackId, false)
      dataProvider.getOne('keepalive', { id: info.trackId })
    }
  }

  if (authenticated && options.audioLists.length > 0) {
    return (
      <ReactJkMusicPlayer
        {...options}
        onAudioListsChange={OnAudioListsChange}
        onAudioProgress={OnAudioProgress}
        onAudioPlay={OnAudioPlay}
      />
    )
  }
  return <div />
}

export default Player
