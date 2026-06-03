import React from 'react'
import PropTypes from 'prop-types'
import { useDispatch } from 'react-redux'
import {
  Button,
  TopToolbar,
  useTranslate,
  useDataProvider,
  useNotify,
} from 'react-admin'
import { useMediaQuery, makeStyles } from '@material-ui/core'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import ShuffleIcon from '@material-ui/icons/Shuffle'
import CloudDownloadOutlinedIcon from '@material-ui/icons/CloudDownloadOutlined'
import { RiPlayListAddFill, RiPlayList2Fill } from 'react-icons/ri'
import PlaylistAddIcon from '@material-ui/icons/PlaylistAdd'
import ShareIcon from '@material-ui/icons/Share'
import LibraryAddIcon from '@material-ui/icons/LibraryAdd'
import {
  playNext,
  addTracks,
  playTracks,
  shuffleTracks,
  openAddToPlaylist,
  openDownloadMenu,
  DOWNLOAD_MENU_FOLDER,
  openShareMenu,
} from '../actions'
import { formatBytes } from '../utils'
import config from '../config'
import { ToggleFieldsMenu } from '../common'

const useStyles = makeStyles({
  toolbar: { display: 'flex', justifyContent: 'space-between', width: '100%' },
})

const FolderButton = ({ children, ...rest }) => {
  return (
    <Button {...rest}>
      {children}
    </Button>
  )
}

const FolderActions = ({
  className,
  record,
  ...rest
}) => {
  const dispatch = useDispatch()
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const notify = useNotify()
  const classes = useStyles()
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  const isNotSmall = useMediaQuery((theme) => theme.breakpoints.up('sm'))

  const getRecursiveTracks = React.useCallback(() => {
    return dataProvider.getList('song', {
      pagination: { page: 1, perPage: -1 },
      sort: { field: 'path', order: 'ASC' },
      filter: { folder_id_recursive: record.id, missing: false },
    }).then(({ data }) => {
      const ids = data.map((s) => s.id)
      const dataMap = data.reduce((acc, cur) => ({ ...acc, [cur.id]: cur }), {})
      return { data: dataMap, ids }
    }).catch((err) => {
      notify('ra.notification.http_error', 'warning')
      throw err
    })
  }, [dataProvider, record.id, notify])

  const handlePlay = React.useCallback(async () => {
    const { data, ids } = await getRecursiveTracks()
    dispatch(playTracks(data, ids))
  }, [getRecursiveTracks, dispatch])

  const handlePlayNext = React.useCallback(async () => {
    const { data, ids } = await getRecursiveTracks()
    dispatch(playNext(data, ids))
  }, [getRecursiveTracks, dispatch])

  const handlePlayLater = React.useCallback(async () => {
    const { data, ids } = await getRecursiveTracks()
    dispatch(addTracks(data, ids))
  }, [getRecursiveTracks, dispatch])

  const handleShuffle = React.useCallback(async () => {
    const { data, ids } = await getRecursiveTracks()
    dispatch(shuffleTracks(data, ids))
  }, [getRecursiveTracks, dispatch])

  const handleAddToPlaylist = React.useCallback(async () => {
    const { ids } = await getRecursiveTracks()
    dispatch(openAddToPlaylist({ selectedIds: ids }))
  }, [getRecursiveTracks, dispatch])

  const handleShare = React.useCallback(() => {
    dispatch(openShareMenu([record.id], 'folder', record.name))
  }, [dispatch, record])

  const handleDownload = React.useCallback(() => {
    dispatch(openDownloadMenu(record, DOWNLOAD_MENU_FOLDER))
  }, [dispatch, record])

  const handlePinAsPlaylist = React.useCallback(async () => {
    try {
      const { ids } = await getRecursiveTracks()
      await dataProvider.create('playlist', {
        data: {
          name: record.name,
          physicalFolderId: record.id,
          public: false,
          tracks: ids.map(id => ({ mediaFileId: id }))
        },
      })
      notify('resources.folder.notifications.pinnedAsPlaylist', 'info')
    } catch (e) {
      notify('ra.notification.http_error', 'warning')
    }
  }, [getRecursiveTracks, dataProvider, record, notify])

  if (!record) return null

  return (
    <TopToolbar className={className} {...rest}>
      <div className={classes.toolbar}>
        <div>
          <FolderButton
            onClick={handlePlay}
            label={translate('resources.album.actions.playAll')}
          >
            <PlayArrowIcon />
          </FolderButton>
          <FolderButton
            onClick={handleShuffle}
            label={translate('resources.album.actions.shuffle')}
          >
            <ShuffleIcon />
          </FolderButton>
          <FolderButton
            onClick={handlePlayNext}
            label={translate('resources.album.actions.playNext')}
          >
            <RiPlayList2Fill />
          </FolderButton>
          <FolderButton
            onClick={handlePlayLater}
            label={translate('resources.album.actions.addToQueue')}
          >
            <RiPlayListAddFill />
          </FolderButton>
          <FolderButton
            onClick={handleAddToPlaylist}
            label={translate('resources.album.actions.addToPlaylist')}
          >
            <PlaylistAddIcon />
          </FolderButton>
          <FolderButton
            onClick={handlePinAsPlaylist}
            label={translate('resources.folder.actions.pinAsPlaylist')}
          >
            <LibraryAddIcon />
          </FolderButton>
          {config.enableSharing && (
            <FolderButton
              onClick={handleShare}
              label={translate('ra.action.share')}
            >
              <ShareIcon />
            </FolderButton>
          )}
          {config.enableDownloads && (
            <FolderButton
              onClick={handleDownload}
              label={
                translate('ra.action.download') +
                (isDesktop && record.size ? ` (${formatBytes(record.size)})` : '')
              }
            >
              <CloudDownloadOutlinedIcon />
            </FolderButton>
          )}
        </div>
        <div>{isNotSmall && <ToggleFieldsMenu resource="folderSong" />}</div>
      </div>
    </TopToolbar>
  )
}

FolderActions.propTypes = {
  record: PropTypes.object,
}

export default FolderActions
