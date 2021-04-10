import React, { useEffect, useState } from 'react'
import PropTypes from 'prop-types'
import { useDispatch, useSelector } from 'react-redux'
import {
  Button,
  sanitizeListRestProps,
  TopToolbar,
  useTranslate,
} from 'react-admin'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import PauseIcon from '@material-ui/icons/Pause'
import ShuffleIcon from '@material-ui/icons/Shuffle'
import CloudDownloadOutlinedIcon from '@material-ui/icons/CloudDownloadOutlined'
import { RiPlayListAddFill, RiPlayList2Fill } from 'react-icons/ri'
import {
  playNext,
  addTracks,
  playTracks,
  shuffleTracks,
  pauseTracks,
  pausePlayer,
} from '../actions'
import subsonic from '../subsonic'
import { formatBytes } from '../utils'
import { useMediaQuery } from '@material-ui/core'
import config from '../config'
import { get } from 'lodash'
import { playingInAlbumOrPlaylist } from '../common'

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
  const [playing, setPlaying] = useState(false)

  const currentTrack = useSelector((state) => get(state, 'queue.current', {}))
  const albumOrPlaylistId = useSelector((state) =>
    get(state, 'recentAlbumOrPlaylist.id', '')
  )
  const songAlbumOrPlaylistId = useSelector((state) =>
    get(state, 'queue.albumOrPlaylistId', '')
  )
  useEffect(() => {
    setPlaying(
      playingInAlbumOrPlaylist(
        currentTrack,
        albumOrPlaylistId,
        songAlbumOrPlaylistId
      )
    )
  }, [currentTrack, albumOrPlaylistId, songAlbumOrPlaylistId])

  const handlePlay = React.useCallback(() => {
    dispatch(playTracks(data, ids, undefined, record.id))
  }, [dispatch, data, ids])

  const handlePlayNext = React.useCallback(() => {
    dispatch(playNext(data, ids))
  }, [dispatch, data, ids])

  const handlePlayLater = React.useCallback(() => {
    dispatch(addTracks(data, ids))
  }, [dispatch, data, ids])

  const handleShuffle = React.useCallback(() => {
    dispatch(shuffleTracks(data, ids, record.id))
  }, [dispatch, data, ids])

  const handleDownload = React.useCallback(() => {
    subsonic.download(record.id)
  }, [record])

  return (
    <TopToolbar className={className} {...sanitizeListRestProps(rest)}>
      <Button
        onClick={playing ? () => dispatch(pausePlayer()) : handlePlay}
        label={
          playing
            ? translate('resources.album.actions.pause')
            : translate('resources.album.actions.playAll')
        }
      >
        {playing ? <PauseIcon /> : <PlayArrowIcon />}
      </Button>
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
