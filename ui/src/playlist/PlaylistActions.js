import { useMediaQuery } from '@material-ui/core'
import CloudDownloadOutlinedIcon from '@material-ui/icons/CloudDownloadOutlined'
import QueueMusicIcon from '@material-ui/icons/QueueMusic'
import ShuffleIcon from '@material-ui/icons/Shuffle'
import PropTypes from 'prop-types'
import React from 'react'
import {
  Button,
  sanitizeListRestProps,
  TopToolbar,
  useDataProvider,
  useNotify,
  useTranslate,
} from 'react-admin'
import { RiPlayList2Fill, RiPlayListAddFill } from 'react-icons/ri'
import { useDispatch } from 'react-redux'
import { addTracks, playNext, playTracks, shuffleTracks } from '../actions'
import { PlayButton } from '../common'
import config from '../config'
import { M3U_MIME_TYPE, REST_URL } from '../consts'
import { httpClient } from '../dataProvider'
import subsonic from '../subsonic'
import { formatBytes } from '../utils'

const PlaylistActions = ({ className, ids, data, record, ...rest }) => {
  const dispatch = useDispatch()
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const notify = useNotify()
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))

  const getAllSongsAndDispatch = React.useCallback(
    (action) => {
      if (ids.length === record.songCount) {
        return dispatch(action(data, ids, undefined, record.id))
      }

      dataProvider
        .getList('playlistTrack', {
          pagination: { page: 1, perPage: 0 },
          sort: { field: 'id', order: 'ASC' },
          filter: { playlist_id: record.id },
        })
        .then((res) => {
          const data = res.data.reduce(
            (acc, curr) => ({ ...acc, [curr.id]: curr }),
            {}
          )
          dispatch(action(data, undefined, undefined, record.id))
        })
        .catch(() => {
          notify('ra.page.error', 'warning')
        })
    },
    [dataProvider, dispatch, record, data, ids, notify]
  )

  const handlePlay = React.useCallback(() => {
    getAllSongsAndDispatch(playTracks)
  }, [getAllSongsAndDispatch])

  const handlePlayNext = React.useCallback(() => {
    getAllSongsAndDispatch(playNext)
  }, [getAllSongsAndDispatch])

  const handlePlayLater = React.useCallback(() => {
    getAllSongsAndDispatch(addTracks)
  }, [getAllSongsAndDispatch])

  const handleShuffle = React.useCallback(() => {
    getAllSongsAndDispatch(shuffleTracks)
  }, [getAllSongsAndDispatch])

  const handleDownload = React.useCallback(() => {
    subsonic.download(record.id)
  }, [record])

  const handleExport = React.useCallback(
    () =>
      httpClient(`${REST_URL}/playlist/${record.id}/tracks`, {
        headers: new Headers({ Accept: M3U_MIME_TYPE }),
      }).then((res) => {
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
      <PlayButton record={record} buttonType="button" handlePlay={handlePlay} />
      <Button
        onClick={handleShuffle}
        label={translate('resources.album.actions.shuffle')}
      >
        <ShuffleIcon />
      </Button>
      <Button
        onClick={handlePlayNext}
        label={translate('resources.album.actions.playNext')}
      >
        <RiPlayList2Fill />
      </Button>
      <Button
        onClick={handlePlayLater}
        label={translate('resources.album.actions.addToQueue')}
      >
        <RiPlayListAddFill />
      </Button>
      {config.enableDownloads && (
        <Button
          onClick={handleDownload}
          label={
            translate('resources.album.actions.download') +
            (isDesktop ? ` (${formatBytes(record.size)})` : '')
          }
        >
          <CloudDownloadOutlinedIcon />
        </Button>
      )}
      <Button
        onClick={handleExport}
        label={translate('resources.playlist.actions.export')}
      >
        <QueueMusicIcon />
      </Button>
    </TopToolbar>
  )
}

PlaylistActions.propTypes = {
  record: PropTypes.object.isRequired,
  selectedIds: PropTypes.arrayOf(PropTypes.number),
}

PlaylistActions.defaultProps = {
  record: {},
  selectedIds: [],
  onUnselectItems: () => null,
}

export default PlaylistActions
