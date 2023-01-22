import ReactJkMusicPlayer from 'navidrome-music-player'
import config, { shareInfo } from './config'
import { baseUrl, shareCoverUrl, shareStreamUrl } from './utils'

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
    mobileMediaQuery: '',
    showDownload: false,
    showReload: false,
    showMediaSession: true,
    theme: 'auto',
    showThemeSwitch: false,
  }
  return <ReactJkMusicPlayer {...options} />
}

export default SharePlayer
