import { useMediaQuery } from '@material-ui/core'
import CloudDownloadOutlinedIcon from '@material-ui/icons/CloudDownloadOutlined'
import ShuffleIcon from '@material-ui/icons/Shuffle'
import PropTypes from 'prop-types'
import React from 'react'
import {
  Button,
  sanitizeListRestProps,
  TopToolbar,
  useTranslate,
} from 'react-admin'
import { RiPlayList2Fill, RiPlayListAddFill } from 'react-icons/ri'
import { useDispatch } from 'react-redux'
import { addTracks, playNext, playTracks, shuffleTracks } from '../actions'
import { PlayButton } from '../common'
import config from '../config'
import subsonic from '../subsonic'
import { formatBytes } from '../utils'

const AlbumActions = ({
  className,
  ids,
  data,
  record,
  permanentFilter,
  ...rest
}) => {
  const dispatch = useDispatch()
  const translate = useTranslate()
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))

  const handlePlay = React.useCallback(() => {
    dispatch(playTracks(data, ids, undefined, record.id))
  }, [dispatch, data, ids, record.id])

  const handlePlayNext = React.useCallback(() => {
    dispatch(playNext(data, ids))
  }, [dispatch, data, ids])

  const handlePlayLater = React.useCallback(() => {
    dispatch(addTracks(data, ids))
  }, [dispatch, data, ids])

  const handleShuffle = React.useCallback(() => {
    dispatch(shuffleTracks(data, ids, record.id))
  }, [dispatch, data, ids, record.id])

  const handleDownload = React.useCallback(() => {
    subsonic.download(record.id)
  }, [record])

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
    </TopToolbar>
  )
}

AlbumActions.propTypes = {
  record: PropTypes.object.isRequired,
  selectedIds: PropTypes.arrayOf(PropTypes.number),
}

AlbumActions.defaultProps = {
  record: {},
  selectedIds: [],
  onUnselectItems: () => null,
}

export default AlbumActions
