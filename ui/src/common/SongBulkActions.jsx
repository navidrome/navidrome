import React, { Fragment, useEffect } from 'react'
import { useTranslate, useUnselectAll } from 'react-admin'
import { addTracks, playNext, playTracks } from '../actions'
import { RiPlayList2Fill, RiPlayListAddFill } from 'react-icons/ri'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import { BatchPlayButton } from './index'
import { AddToPlaylistButton } from './AddToPlaylistButton'
import { makeStyles } from '@material-ui/core/styles'
import { Tooltip } from '@material-ui/core'
import { BatchShareButton } from './BatchShareButton'
import { BatchRateButton } from './BatchRateButton'
import config from '../config'

const useStyles = makeStyles((theme) => ({
  button: {
    color: theme.palette.type === 'dark' ? 'white' : undefined,
    marginRight: 5,
    '&:hover': {
      backgroundColor: 'rgba(255, 255, 255, 0.15)',
    },
  },
}))

const TipButton = ({ labelKey, children }) => {
  const translate = useTranslate()
  return (
    <Tooltip title={translate(labelKey)}>
      <span>{children}</span>
    </Tooltip>
  )
}

export const SongBulkActions = (props) => {
  const classes = useStyles()
  const unselectAll = useUnselectAll()

  useEffect(() => {
    unselectAll(props.resource)
  }, [unselectAll, props.resource])
  return (
    <Fragment>
      <TipButton labelKey={'resources.song.actions.playNow'}>
        <BatchPlayButton
          {...props}
          action={playTracks}
          label={'resources.song.actions.playNowShort'}
          icon={<PlayArrowIcon />}
          className={classes.button}
        />
      </TipButton>
      <TipButton labelKey={'resources.song.actions.playNext'}>
        <BatchPlayButton
          {...props}
          action={playNext}
          label={'resources.song.actions.playNextShort'}
          icon={<RiPlayList2Fill />}
          className={classes.button}
        />
      </TipButton>
      <TipButton labelKey={'resources.song.actions.addToQueue'}>
        <BatchPlayButton
          {...props}
          action={addTracks}
          label={'resources.song.actions.addToQueueShort'}
          icon={<RiPlayListAddFill />}
          className={classes.button}
        />
      </TipButton>
      {config.enableSharing && (
        <TipButton labelKey={'resources.song.actions.share'}>
          <BatchShareButton {...props} className={classes.button} />
        </TipButton>
      )}
      <TipButton labelKey={'resources.song.actions.addToPlaylist'}>
        <AddToPlaylistButton
          {...props}
          className={classes.button}
          label={'resources.song.actions.addToPlaylistShort'}
        />
      </TipButton>
      {config.enableStarRating && (
        <TipButton labelKey={'resources.song.actions.batchRate'}>
          <BatchRateButton {...props} className={classes.button} />
        </TipButton>
      )}
    </Fragment>
  )
}
