import clsx from 'clsx'
import IcecastMetadataPlayer from 'icecast-metadata-player'
import { useCallback, useEffect, useRef, useState } from 'react'

import Slider from 'rc-slider/lib/Slider'

import {
  AnimatePauseIcon,
  AnimatePlayIcon,
  CloseIcon,
  DeleteIcon,
  VolumeMuteIcon,
  VolumeUnmuteIcon,
} from 'navidrome-music-player/es/components/Icon'
import RadioTitle from './RadioTitle'
import { useDispatch } from 'react-redux'

import { clearQueue } from '../actions'
import { useMediaQuery } from '@material-ui/core'
import RadioPlayerMobile from './RadioPlayerMobile'
import subsonic from '../subsonic'
import config from '../config'

const DEFAULT_ICON = {
  pause: <AnimatePauseIcon />,
  play: <AnimatePlayIcon />,
  destroy: <CloseIcon />,
  close: <CloseIcon />,
  delete: <DeleteIcon size={24} />,
  volume: <VolumeUnmuteIcon size={26} />,
  mute: <VolumeMuteIcon size={26} />,
}

const MIN_TIME_BETWEEN_SCROBBLE_MS = 30 * 1000
const SCROBBLE_DELAY_MS = 4 * 60 * 1000

function parseMetadata(metadata) {
  const split = metadata.StreamTitle.split(' - ')
  const artist = split[0]
  const title = split.slice(1).join(' - ')

  return [artist, title]
}

const RadioPlayer = ({
  className,
  cover,
  icon = {},
  locale,
  homePageUrl,
  id,
  name,
  streamUrl,
  theme,
}) => {
  const dispatch = useDispatch()
  const audioRef = useRef()

  const [cast, setCast] = useState(null)
  const [currentStream, setCurrentStream] = useState(null)
  const [loading, setLoading] = useState(false)
  const [metadata, setMetadata] = useState({})
  const [playing, setPlaying] = useState(false)
  const [savedVolume, setSavedVolume] = useState(1)
  const [volume, setVolume] = useState(1)

  const isMobile = useMediaQuery(
    '(max-width: 768px) and (orientation : portrait)'
  )

  const Spin = () => <span className="loading group">{icon.loading}</span>

  const iconMap = { ...DEFAULT_ICON, ...icon, loading: <Spin /> }

  const mapListenToBar = (vol) => Math.sqrt(vol)
  const mapBarToListen = (vol) => vol ** 2

  useEffect(() => {
    const streamChanged = currentStream !== streamUrl

    if (cast && !streamChanged && cast.state !== 'stopped') {
      return
    }

    if (!config.enableProxy) {
      const node = audioRef.current

      if (node && streamChanged) {
        node.crossOrigin = 'anonymous'
        node.src = streamUrl

        node.play()
        setPlaying(true)

        return () => {
          node.src = ''
        }
      } else {
        return
      }
    }

    if (cast) {
      cast.stop()
      cast.detachAudioElement()
    }

    if (streamUrl) {
      const player = new IcecastMetadataPlayer(streamUrl, {
        onMetadata: setMetadata,
        onPlay: () => {
          setLoading(false)
          setPlaying(true)
        },
        onStop: () => {
          setPlaying(false)
        },
        icyDetectionTimeout: 10000,
        enableLogging: true,
        audioElement: audioRef.current,
        playbackMethod: 'mediasource',
      })

      player.id = Math.random()
      player.play()

      setCast(player)
      setLoading(true)
    } else {
      setCast(null)
    }

    setCurrentStream(streamUrl)
    setMetadata({})
  }, [audioRef, cast, currentStream, streamUrl])

  useEffect(() => {
    if (metadata.StreamTitle) {
      let scrobbled = false

      const currentUpdate = new Date()

      const [artist, title] = parseMetadata(metadata)

      if ('mediaSession' in navigator) {
        navigator.mediaSession.metadata = new MediaMetadata({
          album: name,
          artist,
          title,
        })
      }

      subsonic.scrobbleRadio(artist, title, false)

      const timeout = setTimeout(() => {
        scrobbled = true
        subsonic.scrobbleRadio(artist, title, true)
      }, SCROBBLE_DELAY_MS)

      return () => {
        const now = new Date()

        if (!scrobbled) {
          clearTimeout(timeout)

          if (now - currentUpdate > MIN_TIME_BETWEEN_SCROBBLE_MS) {
            const [priorArtist, priorTitle] = parseMetadata(metadata)
            subsonic.scrobbleRadio(priorArtist, priorTitle, true)
          }
        }
      }
    }
  }, [metadata, name])

  useEffect(() => {
    const audio = audioRef.current

    if (audio) {
      audio.onvolumechange = () => {
        const { volume } = audio
        setVolume(mapListenToBar(volume))
      }
    }
  }, [audioRef])

  const setAudioVolume = useCallback((volumeBarVal) => {
    if (audioRef.current) {
      audioRef.current.volume = mapBarToListen(volumeBarVal)

      setSavedVolume(volumeBarVal)
      setVolume(volumeBarVal)
    }
  }, [])

  const mute = useCallback(() => {
    const audio = audioRef.current

    if (audio) {
      setVolume(0)
      setSavedVolume(audio.volume)

      audio.volume = 0
    }
  }, [])

  const resetVolume = useCallback(() => {
    setAudioVolume(mapListenToBar(savedVolume || 0.1))
  }, [savedVolume, setAudioVolume])

  const togglePlay = useCallback(() => {
    const audio = audioRef.current

    if (audio) {
      if (audio.paused) {
        audio.play()
        setPlaying(true)
      } else {
        audio.pause()
        setPlaying(false)
      }
    }
  }, [])

  const coverClick = useCallback(() => {
    window.location.href = `#/radio?displayedFilters={}&filter={"name":"${name}"}`
  }, [name])

  const stopPlaying = useCallback(() => {
    audioRef.current.src = ''
    dispatch(clearQueue())
  }, [dispatch])

  return (
    <div
      className={clsx(
        'react-jinke-music-player-main',
        {
          'light-theme': theme === 'light',
          'dark-theme': theme === 'dark',
        },
        className
      )}
    >
      {isMobile && (
        <RadioPlayerMobile
          cover={cover}
          homePageUrl={homePageUrl}
          icon={iconMap}
          loading={loading}
          locale={locale}
          metadata={metadata}
          name={name}
          onClose={stopPlaying}
          onCoverClick={coverClick}
          onPlay={togglePlay}
          playing={playing}
        />
      )}
      {!isMobile && (
        <div className={clsx('music-player-panel', 'translate')}>
          <section className="panel-content">
            {cover && (
              <div
                className={clsx('img-content', 'img-rotate', {
                  'img-rotate-pause': !playing || !cover,
                })}
                style={{ backgroundImage: `url(${cover})` }}
                onClick={() => coverClick()}
              />
            )}
            <div className="progress-bar-content">
              <span className="audio-title" title={metadata.StreamTitle || ''}>
                <RadioTitle
                  homePageUrl={homePageUrl}
                  isMobile={false}
                  metadata={metadata}
                  name={name}
                />
              </span>
              <section className="audio-main"></section>
            </div>
            <div className="player-content">
              <span className="group">
                {loading ? (
                  <span
                    className="group loading-icon"
                    title={locale.loadingText}
                  >
                    {iconMap.loading}
                  </span>
                ) : (
                  <span
                    className="group play-btn"
                    onClick={togglePlay}
                    title={
                      playing ? locale.clickToPauseText : locale.clickToPlayText
                    }
                  >
                    {playing ? iconMap.pause : iconMap.play}
                  </span>
                )}
              </span>

              <span className="group play-sounds" title={locale.volumeText}>
                {volume === 0 ? (
                  <span className="sounds-icon" onClick={resetVolume}>
                    {iconMap.mute}
                  </span>
                ) : (
                  <span className="sounds-icon" onClick={mute}>
                    {iconMap.volume}
                  </span>
                )}
                <Slider
                  value={volume}
                  onChange={setAudioVolume}
                  className="sound-operation"
                  min={0}
                  max={1}
                  step={0.01}
                />
              </span>
              <span
                title={locale.destroyText}
                className="group destroy-btn"
                onClick={stopPlaying}
              >
                {iconMap.destroy}
              </span>
            </div>
          </section>
        </div>
      )}

      {streamUrl && <audio ref={audioRef} />}
    </div>
  )
}

export default RadioPlayer
