import ReactJkMusicPlayer from 'navidrome-music-player'
import { shareInfo } from './config'
import { shareCoverUrl, shareStreamUrl } from './utils'

const SharePlayer = () => {
  const list = shareInfo?.tracks.map((s) => {
    return {
      name: s.title,
      musicSrc: shareStreamUrl(s.id),
      cover: shareCoverUrl(s.id),
      singer: s.artist,
      duration: s.duration,
    }
  })
  const options = {
    audioLists: list,
    mode: 'full',
    toggleMode: false,
    mobileMediaQuery: '',
    showDownload: false,
    showReload: false,
    showMediaSession: true,
    theme: 'auto',
    showThemeSwitch: false,
    remove: false,
    spaceBar: true,
    volumeFade: { fadeIn: 200, fadeOut: 200 },
  }
  return <ReactJkMusicPlayer {...options} />
}

export default SharePlayer
