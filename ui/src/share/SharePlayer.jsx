import ReactJkMusicPlayer from 'navidrome-music-player'
import { useCallback, useEffect, useRef, useState } from 'react'
import config, { shareInfo } from '../config'
import { shareCoverUrl, shareDownloadUrl, shareStreamUrl } from '../utils'
import VolumeControl from './VolumeControl'

import { makeStyles } from '@material-ui/core/styles'

// How long the download button stays inert after a click. The browser needs a
// moment to show its own download UI; until then the page looks unresponsive.
export const DOWNLOAD_FEEDBACK_MS = 5000

const useStyle = makeStyles({
  player: {
    '& .group .next-audio': {
      pointerEvents: (props) => props.single && 'none',
      opacity: (props) => props.single && 0.65,
    },
    '& .group.audio-download': {
      pointerEvents: (props) => props.downloading && 'none',
      opacity: (props) => props.downloading && 0.65,
    },
    // Free space goes evenly around the volume and icon rows, so the volume
    // row sits halfway between the play buttons and the icons
    '& .react-jinke-music-player-mobile-toggle': {
      flexGrow: 0,
      paddingBottom: 0,
    },
    '& .react-jinke-music-player-mobile-operation': {
      display: 'flex',
      flexDirection: 'column',
    },
    '& .react-jinke-music-player-mobile-operation .items': {
      flex: '1 1 auto',
      flexWrap: 'wrap',
      alignContent: 'space-evenly',
    },
    '@media (min-width: 768px)': {
      '& .react-jinke-music-player-mobile > div:not(.react-jinke-music-player-mobile-cover)':
        {
          width: 768,
          margin: 'auto',
        },
    },
  },
})

const SharePlayer = () => {
  const [downloading, setDownloading] = useState(false)
  const timer = useRef(null)
  const player = useRef(null)
  const classes = useStyle({
    single: shareInfo?.tracks.length === 1,
    downloading,
  })

  useEffect(() => () => clearTimeout(timer.current), [])

  const list = shareInfo?.tracks.map((s) => {
    return {
      name: s.title,
      musicSrc: shareStreamUrl(s.id),
      cover: shareCoverUrl(s.id, true),
      singer: s.artist,
      duration: s.duration,
    }
  })
  // An anchor, not a navigation: the service worker's NavigationRoute would
  // intercept the streamed archive and fail it.
  const customDownloader = useCallback(() => {
    const link = document.createElement('a')
    link.href = shareDownloadUrl(shareInfo?.id)
    link.download = ''
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)

    setDownloading(true)
    clearTimeout(timer.current)
    timer.current = setTimeout(
      () => setDownloading(false),
      DOWNLOAD_FEEDBACK_MS,
    )
  }, [])
  // Not audio.volume: the player would keep a stale volume and restore it after a pause
  const setVolume = useCallback(
    (volume) => player.current?.setAudioVolume(volume),
    [],
  )
  const setMuted = useCallback((muted) => {
    if (player.current) player.current.audio.muted = muted
  }, [])
  const options = {
    audioLists: list,
    mode: 'full',
    toggleMode: false,
    mobileMediaQuery: '',
    showDownload: shareInfo?.downloadable && config.enableDownloads,
    showReload: false,
    showMediaSession: true,
    theme: 'auto',
    showThemeSwitch: false,
    restartCurrentOnPrev: true,
    remove: false,
    spaceBar: true,
    volumeFade: { fadeIn: 200, fadeOut: 200 },
    sortableOptions: { delay: 200, delayOnTouchOnly: true },
    extendsContent: (
      <VolumeControl onVolumeChange={setVolume} onMutedChange={setMuted} />
    ),
  }
  return (
    <ReactJkMusicPlayer
      {...options}
      ref={player}
      className={classes.player}
      customDownloader={customDownloader}
    />
  )
}

export default SharePlayer
