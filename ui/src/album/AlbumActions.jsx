import React from 'react'
import PropTypes from 'prop-types'
import { useDispatch } from 'react-redux'
import {
  Button,
  sanitizeListRestProps,
  TopToolbar,
  useDataProvider,
  useNotify,
  useRecordContext,
  useTranslate,
} from 'react-admin'
import { useMediaQuery, makeStyles } from '@material-ui/core'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import ShuffleIcon from '@material-ui/icons/Shuffle'
import CloudDownloadOutlinedIcon from '@material-ui/icons/CloudDownloadOutlined'
import { RiPlayListAddFill, RiPlayList2Fill } from 'react-icons/ri'
import PlaylistAddIcon from '@material-ui/icons/PlaylistAdd'
import ShareIcon from '@material-ui/icons/Share'
import {
  playNext,
  addTracks,
  playTracks,
  openAddToPlaylist,
  openDownloadMenu,
  DOWNLOAD_MENU_ALBUM,
  openShareMenu,
} from '../actions'
import { formatBytes } from '../utils'
import config from '../config'
import { ToggleFieldsMenu } from '../common'

const useStyles = makeStyles({
  toolbar: { display: 'flex', justifyContent: 'space-between', width: '100%' },
})

const AlbumButton = ({ children, ...rest }) => {
  const record = useRecordContext(rest) || {}
  return (
    <Button {...rest} disabled={record.missing}>
      {children}
    </Button>
  )
}

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
  const dataProvider = useDataProvider()
  const notify = useNotify()
  const classes = useStyles()
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  const isNotSmall = useMediaQuery((theme) => theme.breakpoints.up('sm'))

  const getAllSongsAndDispatch = React.useCallback(
    (action) => {
      if (ids && ids.length === record.songCount) {
        return dispatch(action(data, ids))
      }
      dataProvider
        .getList('song', {
          pagination: { page: 1, perPage: 0 },
          sort: { field: 'album', order: 'ASC' },
          filter: { album_id: record.id },
        })
        .then((res) => {
          const allData = res.data.reduce(
            (acc, curr) => ({ ...acc, [curr.id]: curr }),
            {},
          )
          const allIds = res.data.map((s) => s.id)
          dispatch(action(allData, allIds))
        })
        .catch(() => {
          notify('ra.page.error', 'warning')
        })
    },
    [dataProvider, dispatch, record, data, ids, notify],
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
    const filter = { album_id: record.id }
    if (config.skipLowRatingInShuffle) {
      filter.not_disliked = true
    }
    dataProvider
      .getList('song', {
        pagination: { page: 1, perPage: 500 },
        sort: { field: 'random', order: 'ASC' },
        filter,
      })
      .then((res) => {
        const allData = res.data.reduce(
          (acc, curr) => ({ ...acc, [curr.id]: curr }),
          {},
        )
        dispatch(playTracks(allData))
      })
      .catch(() => {
        notify('ra.page.error', 'warning')
      })
  }, [dataProvider, dispatch, record, notify])

  const handleAddToPlaylist = React.useCallback(() => {
    dispatch(openAddToPlaylist({ albumIds: [record.id] }))
  }, [dataProvider, dispatch, record, data, ids, notify])

  const handleShare = React.useCallback(() => {
    dispatch(openShareMenu([record.id], 'album', record.name))
  }, [dispatch, record])

  const handleDownload = React.useCallback(() => {
    dispatch(openDownloadMenu(record, DOWNLOAD_MENU_ALBUM))
  }, [dispatch, record])

  return (
    <TopToolbar className={className} {...sanitizeListRestProps(rest)}>
      <div className={classes.toolbar}>
        <div>
          <AlbumButton
            onClick={handlePlay}
            label={translate('resources.album.actions.playAll')}
          >
            <PlayArrowIcon />
          </AlbumButton>
          <AlbumButton
            onClick={handleShuffle}
            label={translate('resources.album.actions.shuffle')}
          >
            <ShuffleIcon />
          </AlbumButton>
          <AlbumButton
            onClick={handlePlayNext}
            label={translate('resources.album.actions.playNext')}
          >
            <RiPlayList2Fill />
          </AlbumButton>
          <AlbumButton
            onClick={handlePlayLater}
            label={translate('resources.album.actions.addToQueue')}
          >
            <RiPlayListAddFill />
          </AlbumButton>
          <AlbumButton
            onClick={handleAddToPlaylist}
            label={translate('resources.album.actions.addToPlaylist')}
          >
            <PlaylistAddIcon />
          </AlbumButton>
          {config.enableSharing && (
            <AlbumButton
              onClick={handleShare}
              label={translate('ra.action.share')}
            >
              <ShareIcon />
            </AlbumButton>
          )}
          {config.enableDownloads && (
            <AlbumButton
              onClick={handleDownload}
              label={
                translate('ra.action.download') +
                (isDesktop ? ` (${formatBytes(record.size)})` : '')
              }
            >
              <CloudDownloadOutlinedIcon />
            </AlbumButton>
          )}
        </div>
        <div>{isNotSmall && <ToggleFieldsMenu resource="albumSong" />}</div>
      </div>
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
}

export default AlbumActions
