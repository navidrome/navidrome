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

// light, darg, ligera
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
  const [lastUpdate, setLastUpdate] = useState(null)
  const [loading, setLoading] = useState(false)
  const [metadata, setMetadata] = useState({})
  const [playing, setPlaying] = useState(false)
  const [savedVolume, setSavedVolume] = useState(0)
  const [timeoutId, setTimeoutId] = useState(null)
  const [volume, setVolume] = useState(1)

  // Yes. We have a lock. This is here to prevent attempts to create multiple
  // players. Necessary because they are created in an async function
  const [lock, setLock] = useState(false)

  const isMobile = useMediaQuery(
    '(max-width: 768px) and (orientation : portrait)'
  )

  const Spin = () => <span className="loading group">{icon.loading}</span>

  const iconMap = { ...DEFAULT_ICON, ...icon, loading: <Spin /> }

  const mapListenToBar = (vol) => Math.sqrt(vol)
  const mapBarToListen = (vol) => vol ** 2

  const metadataUpdate = useCallback(
    (newMetadata) => {
      if (timeoutId !== null) {
        clearTimeout(timeoutId)
      }

      const [artist, title] = parseMetadata(newMetadata)

      if ('mediaSession' in navigator) {
        navigator.mediaSession.metadata = new MediaMetadata({
          album: name,
          artist,
          title,
        })
      }

      subsonic.scrobbleRadio(artist, title, false)

      const updateTime = new Date()

      if (lastUpdate !== null && metadata.StreamTitle) {
        const diffMillis = updateTime - lastUpdate

        if (diffMillis > MIN_TIME_BETWEEN_SCROBBLE_MS) {
          const [priorArtist, priorTitle] = parseMetadata(metadata)

          subsonic.scrobbleRadio(priorArtist, priorTitle, true)
        }
      }

      const newTimer = setTimeout(() => {
        subsonic.scrobbleRadio(artist, title, true)
      }, SCROBBLE_DELAY_MS)

      setLastUpdate(updateTime)
      setMetadata(newMetadata)
      setTimeoutId(newTimer)
    },
    [lastUpdate, metadata, name, timeoutId]
  )

  useEffect(() => {
    async function handleChange() {
      if (
        lock ||
        (cast && currentStream === streamUrl && cast.state !== 'stopped')
      ) {
        return
      }

      if (cast) {
        setLock(true)

        if (timeoutId !== null) {
          clearTimeout(timeoutId)
          setTimeoutId(null)
        }

        await cast.stop()
        await cast.detachAudioElement()
      }

      if (streamUrl) {
        const player = new IcecastMetadataPlayer(streamUrl, {
          onMetadata: metadataUpdate,
          onPlay: () => {
            setLoading(false)
            setPlaying(true)
          },
          onStop: () => {
            setPlaying(false)
          },
          onLoad: () => {
            console.log('loading')
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
        setMetadata({})
      } else {
        setCast(null)
      }

      setCurrentStream(streamUrl)
      setLock(false)
    }

    handleChange()
  }, [
    audioRef,
    cast,
    currentStream,
    lock,
    metadataUpdate,
    streamUrl,
    timeoutId,
  ])

  useEffect(() => {
    const audio = audioRef.current

    if (audio) {
      audio.onvolumechange = () => {
        const { volume } = audio
        setVolume(mapListenToBar(volume))
      }
    }
  }, [audioRef])

  const setAudioVolume = useCallback(
    (volumeBarVal) => {
      if (cast) {
        cast.audioElement.volume = mapBarToListen(volumeBarVal)

        setSavedVolume(volumeBarVal)
        setVolume(volumeBarVal)
      }
    },
    [cast]
  )

  const mute = useCallback(() => {
    if (cast) {
      setVolume(0)
      setSavedVolume(cast.audioElement.volume)

      cast.audioElement.volume = 0
    }
  }, [cast])

  const resetVolume = useCallback(() => {
    setAudioVolume(mapListenToBar(savedVolume || 0.1))
  }, [savedVolume, setAudioVolume])

  const togglePlay = useCallback(() => {
    if (cast) {
      const elem = cast.audioElement

      if (elem.paused) {
        elem.play()
        setPlaying(true)
      } else {
        elem.pause()
        setPlaying(false)
      }
    }
  }, [cast])

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
