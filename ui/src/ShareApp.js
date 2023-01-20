import ReactJkMusicPlayer from 'navidrome-music-player'
import config, { shareInfo } from './config'
import { baseUrl } from './utils'

const ShareApp = (props) => {
  const list = shareInfo?.tracks.map((s) => {
    return {
      name: s.title,
      musicSrc: baseUrl(config.publicBaseUrl + '/s/' + s.id),
      cover: baseUrl(config.publicBaseUrl + '/img/' + s.id),
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
  }
  return <ReactJkMusicPlayer {...options} />
}

export default ShareApp
