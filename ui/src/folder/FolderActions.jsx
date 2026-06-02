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
import { RiPlayListAddFill, RiPlayList2Fill } from 'react-icons/ri'
import PlaylistAddIcon from '@material-ui/icons/PlaylistAdd'
import {
  playNext,
  addTracks,
  playTracks,
  shuffleTracks,
  openAddToPlaylist,
} from '../actions'
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
  }, [dispatch, getRecursiveTracks])

  const handlePlayNext = React.useCallback(async () => {
    const { data, ids } = await getRecursiveTracks()
    dispatch(playNext(data, ids))
  }, [dispatch, getRecursiveTracks])

  const handlePlayLater = React.useCallback(async () => {
    const { data, ids } = await getRecursiveTracks()
    dispatch(addTracks(data, ids))
  }, [dispatch, getRecursiveTracks])

  const handleShuffle = React.useCallback(async () => {
    const { data, ids } = await getRecursiveTracks()
    dispatch(shuffleTracks(data, ids))
  }, [dispatch, getRecursiveTracks])

  const handleAddToPlaylist = React.useCallback(async () => {
    const { ids } = await getRecursiveTracks()
    dispatch(openAddToPlaylist({ selectedIds: ids }))
  }, [dispatch, getRecursiveTracks])

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
