import React from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { fetchUtils, useAuthState, useDataProvider } from 'react-admin'
import ReactJkMusicPlayer from 'react-jinke-music-player'
import 'react-jinke-music-player/assets/index.css'
import { markScrobbled, syncQueue } from './queue'

const defaultOptions = {
  bounds: 'body',
  mode: 'full',
  autoPlay: true,
  preload: true,
  autoPlayInitLoadPlayList: true,
  clearPriorAudioLists: false,
  showDownload: false,
  showReload: false,
  glassBg: false,
  showThemeSwitch: false,
  playModeText: {
    order: 'order',
    orderLoop: 'orderLoop',
    singleLoop: 'singleLoop',
    shufflePlay: 'shufflePlay'
  },
  defaultPosition: {
    top: 300,
    left: 120
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

const Player = () => {
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
    const item = queue.queue.find((item) => item.id === info.id)
    if (item && !item.scrobbled) {
      dispatch(markScrobbled(info.id, true))
      fetchUtils.fetchJson(
        `/rest/scrobble?u=admin&p=enc:73756e6461&f=json&v=1.8.0&c=NavidromeUI&id=${info.id}&submission=true`
      )
    }
  }

  const OnAudioPlay = (info) => {
    console.log('AUDIOPLAY: ', info)
    if (info.duration) {
      dispatch(markScrobbled(info.id, false))
      fetchUtils.fetchJson(
        `/rest/scrobble?u=admin&p=enc:73756e6461&f=json&v=1.8.0&c=NavidromeUI&id=${info.id}&submission=false`
      )
      //
      dataProvider.getOne('keepalive', { id: info.id })
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
