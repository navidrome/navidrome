import React, { Fragment, useEffect } from 'react'
import { useUnselectAll } from 'react-admin'
import { addTracks, playNext, playTracks } from '../actions'
import { RiPlayList2Fill, RiPlayListAddFill } from 'react-icons/ri'
import PlayArrowIcon from '@mui/icons-material/PlayArrow'
import { BatchPlayButton } from './index'
import { AddToPlaylistButton } from './AddToPlaylistButton'
import { useTheme } from '@mui/material/styles'
import { BatchShareButton } from './BatchShareButton'
import config from '../config'

// TODO(v6): migrate makeStyles → sx prop / styled()
// const useStyles = makeStyles((theme) => ({
  button: {
    color: theme.palette.type === 'dark' ? 'white' : undefined,
  },
}))

export const SongBulkActions = (props) => {
  const classes = useStyles()
  const unselectAll = useUnselectAll()
  useEffect(() => {
    unselectAll(props.resource)
  }, [unselectAll, props.resource])
  return (
    <Fragment>
      <BatchPlayButton
        {...props}
        action={playTracks}
        label={'resources.song.actions.playNow'}
        icon={<PlayArrowIcon />}
        className={classes.button}
      />
      <BatchPlayButton
        {...props}
        action={playNext}
        label={'resources.song.actions.playNext'}
        icon={<RiPlayList2Fill />}
        className={classes.button}
      />
      <BatchPlayButton
        {...props}
        action={addTracks}
        label={'resources.song.actions.addToQueue'}
        icon={<RiPlayListAddFill />}
        className={classes.button}
      />
      {config.enableSharing && (
        <BatchShareButton {...props} className={classes.button} />
      )}
      <AddToPlaylistButton {...props} className={classes.button} />
    </Fragment>
  )
}
