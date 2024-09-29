import ReactJkMusicPlayer from 'navidrome-music-player'
import config, { shareInfo } from '../config'
import { shareCoverUrl, shareDownloadUrl, shareStreamUrl } from '../utils'

import { makeStyles } from '@material-ui/core/styles'

const useStyle = makeStyles({
  player: {
    '& .group .next-audio': {
      pointerEvents: (props) => props.single && 'none',
      opacity: (props) => props.single && 0.65,
    },
    '@media (min-width: 768px)': {
      '& .react-jinke-music-player-mobile > div': {
        width: 768,
        margin: 'auto',
      },
      '& .react-jinke-music-player-mobile-cover': {
        width: 'auto !important',
      },
    },
  },
})

const SharePlayer = () => {
  const classes = useStyle({ single: shareInfo?.tracks.length === 1 })

  const list = shareInfo?.tracks.map((s) => {
    return {
      name: s.title,
      musicSrc: shareStreamUrl(s.id),
      cover: shareCoverUrl(s.id),
      singer: s.artist,
      duration: s.duration,
    }
  })
  const onBeforeAudioDownload = () => {
    return Promise.resolve({
      src: shareDownloadUrl(shareInfo?.id),
    })
  }
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
  }
  return (
    <ReactJkMusicPlayer
      {...options}
      className={classes.player}
      onBeforeAudioDownload={onBeforeAudioDownload}
    />
  )
}

export default SharePlayer
