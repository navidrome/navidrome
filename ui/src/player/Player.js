import React from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { useAuthState } from 'react-admin'
import ReactJkMusicPlayer from 'react-jinke-music-player'
import 'react-jinke-music-player/assets/index.css'
import { syncQueue } from './queue'

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
  const dispatch = useDispatch()
  const queue = useSelector((state) => state.queue)
  const options = addQueueToOptions(queue)
  const { authenticated } = useAuthState()

  const OnAudioListsChange = (currentPlayIndex, audioLists) => {
    dispatch(syncQueue(audioLists))
  }

  const OnAudioProgress = (info) => {
    const progress = (info.currentTime / info.duration) * 100
  }

  if (authenticated && options.audioLists.length > 0) {
    return (
      <ReactJkMusicPlayer
        {...options}
        onAudioListsChange={OnAudioListsChange}
        onAudioProgress={OnAudioProgress}
      />
    )
  }
  return <div />
}

export default Player
