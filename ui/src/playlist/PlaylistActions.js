import React from 'react'
import { useDispatch } from 'react-redux'
import {
  Button,
  sanitizeListRestProps,
  TopToolbar,
  useTranslate,
} from 'react-admin'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import ShuffleIcon from '@material-ui/icons/Shuffle'
import CloudDownloadOutlinedIcon from '@material-ui/icons/CloudDownloadOutlined'
import AddToQueueIcon from '@material-ui/icons/AddToQueue'
import QueueMusicIcon from '@material-ui/icons/QueueMusic'
import { httpClient } from '../dataProvider'
import { addTracks, playTracks, shuffleTracks } from '../audioplayer'
import { M3U_MIME_TYPE, REST_URL } from '../consts'
import subsonic from '../subsonic'

const PlaylistActions = ({
  className,
  ids,
  data,
  exporter,
  permanentFilter,
  record,
  ...rest
}) => {
  const dispatch = useDispatch()
  const translate = useTranslate()

  const handlePlay = React.useCallback(() => {
    dispatch(playTracks(data, ids))
  }, [dispatch, data, ids])

  const handlePlayLater = React.useCallback(() => {
    dispatch(addTracks(data, ids))
  }, [dispatch, data, ids])

  const handleShuffle = React.useCallback(() => {
    dispatch(shuffleTracks(data, ids))
  }, [dispatch, data, ids])

  const handleDownload = React.useCallback(() => {
    subsonic.download(record.id)
  }, [record])

  const handleExport = React.useCallback(
    () =>
      httpClient(`${REST_URL}/playlist/${record.id}/tracks`, {
        headers: new Headers({ Accept: M3U_MIME_TYPE }),
      }).then((res) => {
        console.log(res)
        const blob = new Blob([res.body], { type: M3U_MIME_TYPE })
        const url = window.URL.createObjectURL(blob)
        const link = document.createElement('a')
        link.href = url
        link.download = `${record.name}.m3u`
        document.body.appendChild(link)
        link.click()
        link.parentNode.removeChild(link)
      }),
    [record]
  )

  return (
    <TopToolbar className={className} {...sanitizeListRestProps(rest)}>
      <Button
        onClick={handlePlay}
        label={translate('resources.album.actions.playAll')}
      >
        <PlayArrowIcon />
      </Button>
      <Button
        onClick={handleShuffle}
        label={translate('resources.album.actions.shuffle')}
      >
        <ShuffleIcon />
      </Button>
      <Button
        onClick={handleDownload}
        label={translate('resources.album.actions.download')}
      >
        <CloudDownloadOutlinedIcon />
      </Button>
      <Button
        onClick={handlePlayLater}
        label={translate('resources.album.actions.addToQueue')}
      >
        <AddToQueueIcon />
      </Button>
      <Button
        onClick={handleExport}
        label={translate('resources.playlist.actions.export')}
      >
        <QueueMusicIcon />
      </Button>
    </TopToolbar>
  )
}

PlaylistActions.defaultProps = {
  selectedIds: [],
  onUnselectItems: () => null,
}

export default PlaylistActions
