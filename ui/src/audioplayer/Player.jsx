import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useInterval } from '../common'
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
  refreshQueue,
  setPlayMode,
  setTranscodingProfile,
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
import { detectBrowserProfile, decisionService } from '../transcode'
import {
  getPreferredLyricLanguage,
  hasStructuredLyricContent,
  selectLyricLayers,
  structuredLyricToLrc,
} from './lyrics'
import {
  resolveLyricsOverlayState,
  togglePronunciationPreference,
} from './lyricsOverlayState'
import KaraokeLyricsOverlay from './KaraokeLyricsOverlay'
import MobileKaraokeLyricsPortal from './MobileKaraokeLyricsPortal'

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
  const [currentTrackId, setCurrentTrackId] = useState(null)
  const [heartbeatTrackId, setHeartbeatTrackId] = useState(null)
  const lastPositionMsRef = useRef(0)
  const currentTrackIdRef = useRef(null)
  const stoppedRef = useRef(false)
  const [audioInstance, setAudioInstance] = useState(null)
  const isDesktop = useMediaQuery('(min-width:810px)')
  const isMobilePlayer =
    /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(
      navigator.userAgent,
    )

  const { authenticated } = useAuthState()

  // Keep a ref to playerState so the mount effect can read the latest value
  // without re-triggering on every queue/position change
  const playerStateRef = useRef(playerState)
  playerStateRef.current = playerState

  currentTrackIdRef.current = currentTrackId

  useInterval(
    () => {
      if (heartbeatTrackId && !stoppedRef.current) {
        subsonic.reportPlayback(
          heartbeatTrackId,
          lastPositionMsRef.current,
          'playing',
        )
      }
    },
    heartbeatTrackId ? config.playbackReportIntervalMs : null,
  )

  // Detect browser codec profile and eagerly resolve transcode URLs for the
  // persisted queue once on mount (e.g. after a browser refresh)
  useEffect(() => {
    const profile = detectBrowserProfile()
    decisionService.setProfile(profile)
    dispatch(setTranscodingProfile(profile))

    const state = playerStateRef.current
    const currentIdx = state.savedPlayIndex || 0
    const trackIds = state.queue
      .slice(currentIdx, currentIdx + 4)
      .filter((item) => !item.isRadio && item.trackId)
      .map((item) => item.trackId)

    if (trackIds.length === 0) {
      dispatch(refreshQueue())
      return
    }

    Promise.allSettled(
      trackIds.map((id) =>
        decisionService.resolveStreamUrl(id).then((url) => [id, url]),
      ),
    ).then((results) => {
      const resolvedUrls = {}
      results.forEach((r) => {
        if (r.status === 'fulfilled') {
          resolvedUrls[r.value[0]] = r.value[1]
        }
      })
      dispatch(refreshQueue(resolvedUrls))
    })
  }, [dispatch])

  // Pre-fetch transcode decisions for next 2-3 songs when queue or position changes
  useEffect(() => {
    if (!playerState.queue.length) return

    const currentIdx = playerState.savedPlayIndex || 0
    const nextSongIds = playerState.queue
      .slice(currentIdx + 1, currentIdx + 4)
      .filter((item) => !item.isRadio)
      .map((item) => item.trackId)

    if (nextSongIds.length > 0) {
      decisionService.prefetchDecisions(nextSongIds)
    }
  }, [playerState.queue, playerState.savedPlayIndex])

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
  const [karaokeVisiblePreference, setKaraokeVisiblePreference] =
    useState(false)
  const [selectedLyricLayers, setSelectedLyricLayers] =
    useState(emptyLyricLayers)
  const [translationPreference, setTranslationPreference] = useState(false)
  const [pronunciationPreference, setPronunciationPreference] = useState(null)
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
  const { karaokeVisible, showTranslation, showPronunciation } =
    resolveLyricsOverlayState({
      karaokeVisiblePreference,
      translationPreference,
      pronunciationPreference,
      hasKaraokeLyric,
      hasTranslationLyric,
      hasPronunciationLyric,
    })
  const useInlineMobileLyrics = karaokeVisible && !isDesktop

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
      if (playerState.current?.uuid && audioInstance && !audioInstance.paused) {
        e.preventDefault()
        e.returnValue = ''
      }
    }

    const handlePageHide = () => {
      if (currentTrackIdRef.current && !playerState.current?.isRadio) {
        stoppedRef.current = true
        try {
          subsonic.reportPlaybackKeepalive(
            currentTrackIdRef.current,
            lastPositionMsRef.current,
            'stopped',
          )
        } catch {
          // fetch/sendBeacon may throw; ignore
        }
      }
    }

    window.addEventListener('beforeunload', handleBeforeUnload)
    window.addEventListener('pagehide', handlePageHide)
    return () => {
      window.removeEventListener('beforeunload', handleBeforeUnload)
      window.removeEventListener('pagehide', handlePageHide)
    }
  }, [playerState, audioInstance])

  useEffect(() => {
    if (!currentTrackId || currentTrackIsRadio) {
      setSelectedLyricLayers(emptyLyricLayers)
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
        // Do not cache network/request failures as empty lyrics, so we can retry.
        lyricCacheRef.current.delete(currentTrackId)
      })
  }, [dispatch, currentTrackId, currentTrackIsRadio, applyLyricToRuntimePlayer])

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
      autoPlay:
        playerState.queue.length > 0 &&
        playerState.autoPlay !== false &&
        (playerState.clear || playerState.playIndex === 0),
      clearPriorAudioLists: playerState.clear,
      extendsContent: (
        <PlayerToolbar
          id={current.trackId}
          isRadio={current.isRadio}
          onToggleLyrics={() =>
            setKaraokeVisiblePreference((visible) => !visible)
          }
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

  const onAudioProgress = useCallback((info) => {
    if (info.ended) {
      document.title = 'Navidrome'
    }
    if (!info.isRadio && info.currentTime != null) {
      lastPositionMsRef.current = Math.floor(info.currentTime * 1000)
    }
  }, [])

  const onAudioSeeked = useCallback(
    (info) => {
      if (!info.isRadio && currentTrackId) {
        const posMs = Math.floor(info.currentTime * 1000)
        lastPositionMsRef.current = posMs
        const state = audioInstance?.paused ? 'paused' : 'playing'
        subsonic.reportPlayback(currentTrackId, posMs, state)
      }
    },
    [currentTrackId, audioInstance],
  )

  const onAudioVolumeChange = useCallback(
    // sqrt to compensate for the logarithmic volume
    (volume) => dispatch(setVolume(Math.sqrt(volume))),
    [dispatch],
  )

  const onAudioPlay = useCallback(
    (info) => {
      if (context && context.state !== 'running') {
        context.resume()
      }

      dispatch(currentPlaying(info))
      if (info.duration) {
        const song = info.song
        document.title = `${song.title} - ${song.artist} - Navidrome`
        if (!info.isRadio) {
          const posMs = Math.floor(info.currentTime * 1000)
          lastPositionMsRef.current = posMs
          const isNewTrack = info.trackId !== currentTrackId
          if (isNewTrack) {
            subsonic
              .reportPlayback(info.trackId, posMs, 'starting')
              .then(() =>
                subsonic.reportPlayback(info.trackId, posMs, 'playing'),
              )
            setCurrentTrackId(info.trackId)
          } else {
            subsonic.reportPlayback(info.trackId, posMs, 'playing')
          }
          setHeartbeatTrackId(info.trackId)
        }
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
    [context, dispatch, showNotifications, currentTrackId],
  )

  const onAudioPlayTrackChange = useCallback(() => {
    if (currentTrackId) {
      subsonic.reportPlayback(
        currentTrackId,
        lastPositionMsRef.current,
        'stopped',
      )
    }
    setHeartbeatTrackId(null)
    setCurrentTrackId(null)
  }, [currentTrackId])

  const onAudioPause = useCallback(
    (info) => {
      dispatch(currentPlaying(info))
      if (!info.isRadio && currentTrackId) {
        const posMs = Math.floor(info.currentTime * 1000)
        lastPositionMsRef.current = posMs
        subsonic.reportPlayback(currentTrackId, posMs, 'paused')
      }
      setHeartbeatTrackId(null)
    },
    [dispatch, currentTrackId],
  )

  const onAudioEnded = useCallback(
    (currentPlayId, audioLists, info) => {
      if (currentTrackId && !info.isRadio) {
        const posMs = Math.floor((info.duration || 0) * 1000)
        subsonic.reportPlayback(currentTrackId, posMs, 'stopped')
      }
      setHeartbeatTrackId(null)
      setCurrentTrackId(null)
      dispatch(currentPlaying(info))
      dataProvider
        .getOne('keepalive', { id: info.trackId })
        // eslint-disable-next-line no-console
        .catch((e) => console.log('Keepalive error:', e))
    },
    [dispatch, dataProvider, currentTrackId],
  )

  const onCoverClick = useCallback((mode, audioLists, audioInfo) => {
    if (!isDesktop && karaokeVisible) {
      return
    }
    if (mode === 'full' && audioInfo?.song?.albumId) {
      window.location.href = `#/album/${audioInfo.song.albumId}/show`
    }
  }, [isDesktop, karaokeVisible])

  const onAudioError = useCallback(
    (error, currentPlayId, audioLists, audioInfo) => {
      // Invalidate all cached decisions — token may be stale
      decisionService.invalidateAll()

      // Pre-fetch decisions for upcoming songs with fresh tokens
      const currentIdx = playerState.queue.findIndex(
        (item) => item.uuid === currentPlayId,
      )
      if (currentIdx >= 0) {
        const nextSongIds = playerState.queue
          .slice(currentIdx + 1, currentIdx + 4)
          .filter((item) => !item.isRadio)
          .map((item) => item.trackId)
        if (nextSongIds.length > 0) {
          decisionService.prefetchDecisions(nextSongIds)
        }
      }
    },
    [playerState.queue],
  )

  const onBeforeDestroy = useCallback(() => {
    return new Promise((resolve, reject) => {
      if (currentTrackId && !playerStateRef.current?.current?.isRadio) {
        subsonic.reportPlayback(
          currentTrackId,
          lastPositionMsRef.current,
          'stopped',
        )
      }
      setHeartbeatTrackId(null)
      setCurrentTrackId(null)
      dispatch(clearQueue())
      reject()
    })
  }, [dispatch, currentTrackId])

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
        onAudioSeeked={onAudioSeeked}
        onAudioPlay={onAudioPlay}
        onAudioPlayTrackChange={onAudioPlayTrackChange}
        onAudioPause={onAudioPause}
        onPlayModeChange={(mode) => dispatch(setPlayMode(mode))}
        onAudioEnded={onAudioEnded}
        onCoverClick={onCoverClick}
        onAudioError={onAudioError}
        onBeforeDestroy={onBeforeDestroy}
        getAudioInstance={setAudioInstance}
      />
      {isDesktop && (
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
            setTranslationPreference((previous) =>
              hasTranslationLyric ? !previous : false,
            )
          }
          onTogglePronunciation={() =>
            setPronunciationPreference((previous) =>
              togglePronunciationPreference(previous, hasPronunciationLyric),
            )
          }
          audioInstance={audioInstance}
          onClose={() => setKaraokeVisiblePreference(false)}
        />
      )}
      <MobileKaraokeLyricsPortal active={useInlineMobileLyrics}>
        <KaraokeLyricsOverlay
          visible={useInlineMobileLyrics}
          inline={true}
          mainLyric={selectedLyricLayers.main}
          translationLyric={selectedLyricLayers.translation}
          pronunciationLyric={selectedLyricLayers.pronunciation}
          showTranslation={showTranslation}
          showPronunciation={showPronunciation}
          translationEnabled={hasTranslationLyric}
          pronunciationEnabled={hasPronunciationLyric}
          onToggleTranslation={() =>
            setTranslationPreference((previous) =>
              hasTranslationLyric ? !previous : false,
            )
          }
          onTogglePronunciation={() =>
            setPronunciationPreference((previous) =>
              togglePronunciationPreference(previous, hasPronunciationLyric),
            )
          }
          audioInstance={audioInstance}
          onClose={() => setKaraokeVisiblePreference(false)}
        />
      </MobileKaraokeLyricsPortal>
      <GlobalHotKeys handlers={handlers} keyMap={keyMap} allowChanges />
    </ThemeProvider>
  )
}

export { Player }
