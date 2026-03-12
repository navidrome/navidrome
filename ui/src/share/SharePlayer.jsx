import { useState } from 'react'
import { useTranslate } from 'react-admin'
import ReactJkMusicPlayer from 'navidrome-music-player'

import config, { shareInfo } from '../config'
import {
  shareCoverUrl,
  shareDownloadUrl,
  shareStreamUrl,
  toDownloadUrl,
} from '../utils'
import withTheme from '../utils/withTheme'
import { DialogTitle } from '../dialogs/DialogTitle'

import { makeStyles } from '@material-ui/core/styles'
import { Button, Dialog, DialogContent } from '@material-ui/core'
import {
  QueueMusic as QueueMusicIcon,
  MusicNote as MusicNoteIcon,
} from '@material-ui/icons'

const useStyle = makeStyles((theme) => ({
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
  columnLayout: {
    display: 'flex',
    flexDirection: 'column',
    gap: theme.spacing(2),
  },
}))

const downloadFile = (src, filename) => {
  const link = document.createElement('a')
  link.href = src
  link.download = filename ?? '' // when blank, will use the Content-Disposition header
  document.body.appendChild(link)
  link.click()
  link.remove()
}

const SharePlayer = () => {
  const translate = useTranslate()
  const classes = useStyle({ single: shareInfo?.tracks.length === 1 })
  const [downloadInfo, setDownloadInfo] = useState(null)

  const handleCustomDownload = (downloadInfo) => {
    if (shareInfo?.tracks) {
      if (shareInfo.tracks.length === 1) {
        downloadFile(toDownloadUrl(downloadInfo.src))
      } else {
        setDownloadInfo(downloadInfo)
      }
    }
  }

  const handleClose = () => setDownloadInfo(null)

  const list = shareInfo?.tracks.map((s) => {
    return {
      name: s.title,
      musicSrc: shareStreamUrl(s.id),
      cover: shareCoverUrl(s.id, true),
      singer: s.artist,
      duration: s.duration,
    }
  })

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
  }
  return (
    <>
      <ReactJkMusicPlayer
        {...options}
        className={classes.player}
        customDownloader={handleCustomDownload}
      />
      <Dialog
        id="share-download-menu"
        open={downloadInfo}
        onClose={handleClose}
        aria-labelledby="share-download-title"
      >
        <DialogTitle id="share-download-title" onClose={handleClose}>
          {translate('resources.share.actions.download.title')}
        </DialogTitle>
        <DialogContent className={classes.columnLayout} dividers>
          <Button
            variant="contained"
            startIcon={<MusicNoteIcon />}
            onClick={() => {
              downloadFile(toDownloadUrl(downloadInfo.src))
              setDownloadInfo(null)
            }}
          >
            {translate('resources.share.actions.download.currentTrack')}
          </Button>
          <Button
            variant="contained"
            startIcon={<QueueMusicIcon />}
            disabled={!shareInfo}
            onClick={() => {
              downloadFile(
                shareDownloadUrl(shareInfo?.id),
                shareInfo?.description + '.zip',
              )
              setDownloadInfo(null)
            }}
          >
            {translate('resources.share.actions.download.allTracks')}
          </Button>
        </DialogContent>
      </Dialog>
    </>
  )
}

const SharePlayerWithTheme = withTheme(SharePlayer)

export default SharePlayerWithTheme
