import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'
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
import {
  clearQueue,
  currentPlaying,
  setPlayMode,
  updateQueueLyric,
  setVolume,
  syncQueue,
} from '../actions'
import PlayerToolbar from './PlayerToolbar'
import { sendNotification } from '../utils'
import subsonic from '../subsonic'
import locale from './locale'
import { keyMap } from '../hotkeys'
import keyHandlers from './keyHandlers'
import { calculateGain } from '../utils/calculateReplayGain'
import {
  getPreferredLyricLanguage,
  hasStructuredLyricContent,
  selectLyricLayers,
  structuredLyricToLrc,
} from './lyrics'
import KaraokeLyricsOverlay from './KaraokeLyricsOverlay'

const emptyLyricLayers = {
  main: null,
  translation: null,
  pronunciation: null,
}

const normalizeLyricLayers = (layers) => ({
  main: layers?.main || null,
  translation: layers?.translation || null,
  pronunciation: layers?.pronunciation || null,
})

const Player = () => {
  const theme = useCurrentTheme()
  const translate = useTranslate()
  const playerTheme = theme.player?.theme || 'dark'
  const dataProvider = useDataProvider()
  const playerState = useSelector((state) => state.player)
  const dispatch = useDispatch()
  const [startTime, setStartTime] = useState(null)
  const [scrobbled, setScrobbled] = useState(false)
  const [preloaded, setPreload] = useState(false)
  const [audioInstance, setAudioInstance] = useState(null)
  const isDesktop = useMediaQuery('(min-width:810px)')
  const isMobilePlayer =
    /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(
      navigator.userAgent,
    )

  const { authenticated } = useAuthState()
  const visible = authenticated && playerState.queue.length > 0
  const isRadio = playerState.current?.isRadio || false
  const classes = useStyle({
    isRadio,
    visible,
    enableCoverAnimation: config.enableCoverAnimation,
  })
  const showNotifications = useSelector(
    (state) => state.settings.notifications || false,
  )
  const gainInfo = useSelector((state) => state.replayGain)
  const [context, setContext] = useState(null)
  const [gainNode, setGainNode] = useState(null)
  const lyricCacheRef = useRef(new Map())
  const lyricRequestIdRef = useRef(0)
  const playerRef = useRef(null)
  const [karaokeVisible, setKaraokeVisible] = useState(false)
  const [selectedLyricLayers, setSelectedLyricLayers] =
    useState(emptyLyricLayers)
  const [showTranslation, setShowTranslation] = useState(false)
  const [showPronunciation, setShowPronunciation] = useState(false)
  const currentTrackId = playerState.current?.trackId
  const currentTrackIsRadio = playerState.current?.isRadio
  const selectedStructuredLyric = selectedLyricLayers.main
  const hasKaraokeLyric = hasStructuredLyricContent(selectedStructuredLyric)
  const hasTranslationLyric = hasStructuredLyricContent(
    selectedLyricLayers.translation,
  )
  const hasPronunciationLyric = hasStructuredLyricContent(
    selectedLyricLayers.pronunciation,
  )

  const applyLyricToRuntimePlayer = useCallback((trackId, lyric) => {
    if (!trackId) {
      return
    }

    const player = playerRef.current
    if (!player || typeof player.setState !== 'function') {
      return
    }

    player.setState((prevState) => {
      const prevLists = Array.isArray(prevState.audioLists)
        ? prevState.audioLists
        : []
      let changed = false
      const audioLists = prevLists.map((item) => {
        if (item.trackId !== trackId) {
          return item
        }
        if (item.lyric === lyric) {
          return item
        }
        changed = true
        return {
          ...item,
          lyric,
        }
      })

      const currentItem = audioLists.find(
        (item) => item.musicSrc === prevState.musicSrc,
      )
      const currentLyric =
        typeof currentItem?.lyric === 'string'
          ? currentItem.lyric
          : prevState.lyric

      if (!changed && currentLyric === prevState.lyric) {
        return null
      }

      return {
        audioLists,
        lyric: currentLyric,
      }
    })
  }, [])

  useEffect(() => {
    if (
      context === null &&
      audioInstance &&
      config.enableReplayGain &&
      'AudioContext' in window &&
      (gainInfo.gainMode === 'album' || gainInfo.gainMode === 'track')
    ) {
      const ctx = new AudioContext()
      // we need this to support radios in firefox
      audioInstance.crossOrigin = 'anonymous'
      const source = ctx.createMediaElementSource(audioInstance)
      const gain = ctx.createGain()

      source.connect(gain)
      gain.connect(ctx.destination)

      setContext(ctx)
      setGainNode(gain)
    }
  }, [audioInstance, context, gainInfo.gainMode])

  useEffect(() => {
    if (gainNode) {
      const current = playerState.current || {}
      const song = current.song || {}

      const numericGain = calculateGain(gainInfo, song)
      gainNode.gain.setValueAtTime(numericGain, context.currentTime)
    }
  }, [audioInstance, context, gainNode, playerState, gainInfo])

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

  useEffect(() => {
    if (!currentTrackId || currentTrackIsRadio) {
      setSelectedLyricLayers(emptyLyricLayers)
      setShowTranslation(false)
      setShowPronunciation(false)
      setKaraokeVisible(false)
      return
    }

    const cached = lyricCacheRef.current.get(currentTrackId)
    let layers = emptyLyricLayers
    if (cached && typeof cached !== 'string') {
      if (cached.layers) {
        layers = normalizeLyricLayers(cached.layers)
      } else if (cached.structuredLyric) {
        layers = normalizeLyricLayers({
          main: cached.structuredLyric,
        })
      }
    }
    setSelectedLyricLayers(layers)
    setShowTranslation(false)
    setShowPronunciation(hasStructuredLyricContent(layers.pronunciation))
  }, [currentTrackId, currentTrackIsRadio])

  useEffect(() => {
    lyricRequestIdRef.current += 1
    const requestId = lyricRequestIdRef.current

    if (!currentTrackId || currentTrackIsRadio) {
      return
    }

    const cached = lyricCacheRef.current.get(currentTrackId)
    if (cached !== undefined) {
      const cachedLyric =
        typeof cached === 'string' ? cached : cached?.lrc || ''
      const cachedLayers =
        typeof cached === 'string'
          ? emptyLyricLayers
          : cached?.layers
            ? normalizeLyricLayers(cached.layers)
            : normalizeLyricLayers({ main: cached?.structuredLyric })

      setSelectedLyricLayers(cachedLayers)
      setShowTranslation(false)
      setShowPronunciation(
        hasStructuredLyricContent(cachedLayers.pronunciation),
      )
      if (cachedLyric) {
        dispatch(updateQueueLyric(currentTrackId, cachedLyric))
        applyLyricToRuntimePlayer(currentTrackId, cachedLyric)
      }
      return
    }

    subsonic
      .getLyricsBySongId(currentTrackId)
      .then((resp) => {
        if (lyricRequestIdRef.current !== requestId) {
          return
        }

        const structuredLyrics =
          resp?.json?.['subsonic-response']?.lyricsList?.structuredLyrics || []
        const layers = selectLyricLayers(
          structuredLyrics,
          getPreferredLyricLanguage(),
        )
        const lyric = layers.main ? structuredLyricToLrc(layers.main) : ''
        lyricCacheRef.current.set(currentTrackId, {
          lrc: lyric,
          layers,
        })
        setSelectedLyricLayers(layers)
        setShowTranslation(false)
        setShowPronunciation(hasStructuredLyricContent(layers.pronunciation))

        if (lyric !== '') {
          dispatch(updateQueueLyric(currentTrackId, lyric))
          applyLyricToRuntimePlayer(currentTrackId, lyric)
        }
      })
      .catch(() => {
        if (lyricRequestIdRef.current !== requestId) {
          return
        }
        setSelectedLyricLayers(emptyLyricLayers)
        setShowTranslation(false)
        setShowPronunciation(false)
        // Do not cache network/request failures as empty lyrics, so we can retry.
        lyricCacheRef.current.delete(currentTrackId)
      })
  }, [dispatch, currentTrackId, currentTrackIsRadio, applyLyricToRuntimePlayer])

  useEffect(() => {
    if (!hasKaraokeLyric && karaokeVisible) {
      setKaraokeVisible(false)
    }
  }, [hasKaraokeLyric, karaokeVisible])

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
      showLyric: false,
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
    [gainInfo, isDesktop, playerTheme, translate, playerState.mode],
  )

  const options = useMemo(() => {
    const current = playerState.current || {}
    return {
      ...defaultOptions,
      audioLists: playerState.queue.map((item) => item),
      playIndex: playerState.playIndex,
      autoPlay: playerState.clear || playerState.playIndex === 0,
      clearPriorAudioLists: playerState.clear,
      extendsContent: (
        <PlayerToolbar
          id={current.trackId}
          isRadio={current.isRadio}
          onToggleLyrics={() => setKaraokeVisible((visible) => !visible)}
          lyricsActive={karaokeVisible}
          lyricsDisabled={!hasKaraokeLyric}
        />
      ),
      defaultVolume: isMobilePlayer ? 1 : playerState.volume,
      showMediaSession: !current.isRadio,
    }
  }, [
    playerState,
    defaultOptions,
    isMobilePlayer,
    karaokeVisible,
    hasKaraokeLyric,
  ])

  const onAudioListsChange = useCallback(
    (_, audioLists, audioInfo) => dispatch(syncQueue(audioInfo, audioLists)),
    [dispatch],
  )

  const nextSong = useCallback(() => {
    const idx = playerState.queue.findIndex(
      (item) => item.uuid === playerState.current.uuid,
    )
    return idx !== null ? playerState.queue[idx + 1] : null
  }, [playerState])

  const onAudioProgress = useCallback(
    (info) => {
      if (info.ended) {
        document.title = 'Navidrome'
      }

      const progress = (info.currentTime / info.duration) * 100
      if (isNaN(info.duration) || (progress < 50 && info.currentTime < 240)) {
        return
      }

      if (info.isRadio) {
        return
      }

      if (!preloaded) {
        const next = nextSong()
        if (next != null) {
          const audio = new Audio()
          audio.src = next.musicSrc
        }
        setPreload(true)
        return
      }

      if (!scrobbled) {
        info.trackId && subsonic.scrobble(info.trackId, startTime)
        setScrobbled(true)
      }
    },
    [startTime, scrobbled, nextSong, preloaded],
  )

  const onAudioVolumeChange = useCallback(
    // sqrt to compensate for the logarithmic volume
    (volume) => dispatch(setVolume(Math.sqrt(volume))),
    [dispatch],
  )

  const onAudioPlay = useCallback(
    (info) => {
      // Do this to start the context; on chrome-based browsers, the context
      // will start paused since it is created prior to user interaction
      if (context && context.state !== 'running') {
        context.resume()
      }

      dispatch(currentPlaying(info))
      if (startTime === null) {
        setStartTime(Date.now())
      }
      if (info.duration) {
        const song = info.song
        document.title = `${song.title} - ${song.artist} - Navidrome`
        if (!info.isRadio) {
          const pos = startTime === null ? null : Math.floor(info.currentTime)
          subsonic.nowPlaying(info.trackId, pos)
        }
        setPreload(false)
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
            info.cover,
          )
        }
      }
    },
    [context, dispatch, showNotifications, startTime],
  )

  const onAudioPlayTrackChange = useCallback(() => {
    if (scrobbled) {
      setScrobbled(false)
    }
    if (startTime !== null) {
      setStartTime(null)
    }
  }, [scrobbled, startTime])

  const onAudioPause = useCallback(
    (info) => dispatch(currentPlaying(info)),
    [dispatch],
  )

  const onAudioEnded = useCallback(
    (currentPlayId, audioLists, info) => {
      setScrobbled(false)
      setStartTime(null)
      dispatch(currentPlaying(info))
      dataProvider
        .getOne('keepalive', { id: info.trackId })
        // eslint-disable-next-line no-console
        .catch((e) => console.log('Keepalive error:', e))
    },
    [dispatch, dataProvider],
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
    [audioInstance, playerState],
  )

  useEffect(() => {
    if (isMobilePlayer && audioInstance) {
      audioInstance.volume = 1
    }
  }, [isMobilePlayer, audioInstance])

  return (
    <ThemeProvider theme={createMuiTheme(theme)}>
      <ReactJkMusicPlayer
        ref={playerRef}
        {...options}
        className={classes.player}
        onAudioListsChange={onAudioListsChange}
        onAudioVolumeChange={onAudioVolumeChange}
        onAudioProgress={onAudioProgress}
        onAudioPlay={onAudioPlay}
        onAudioPlayTrackChange={onAudioPlayTrackChange}
        onAudioPause={onAudioPause}
        onPlayModeChange={(mode) => dispatch(setPlayMode(mode))}
        onAudioEnded={onAudioEnded}
        onCoverClick={onCoverClick}
        onBeforeDestroy={onBeforeDestroy}
        getAudioInstance={setAudioInstance}
      />
      <KaraokeLyricsOverlay
        visible={karaokeVisible}
        mainLyric={selectedLyricLayers.main}
        translationLyric={selectedLyricLayers.translation}
        pronunciationLyric={selectedLyricLayers.pronunciation}
        showTranslation={showTranslation}
        showPronunciation={showPronunciation}
        translationEnabled={hasTranslationLyric}
        pronunciationEnabled={hasPronunciationLyric}
        onToggleTranslation={() =>
          setShowTranslation((previous) =>
            hasTranslationLyric ? !previous : false,
          )
        }
        onTogglePronunciation={() =>
          setShowPronunciation((previous) =>
            hasPronunciationLyric ? !previous : false,
          )
        }
        audioInstance={audioInstance}
        onClose={() => setKaraokeVisible(false)}
      />
      <GlobalHotKeys handlers={handlers} keyMap={keyMap} allowChanges />
    </ThemeProvider>
  )
}

export { Player }
