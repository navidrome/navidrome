import React, { useState } from 'react'
import {
  Button,
  useDataProvider,
  useTranslate,
  useUnselectAll,
  useNotify,
  useRefresh,
} from 'react-admin'
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button as MuiButton,
  IconButton,
  Tooltip,
} from '@material-ui/core'
import Rating from '@material-ui/lab/Rating'
import StarBorderIcon from '@material-ui/icons/StarBorder'
import FavoriteIcon from '@material-ui/icons/Favorite'
import FavoriteBorderIcon from '@material-ui/icons/FavoriteBorder'
import ClearIcon from '@material-ui/icons/Clear'
import { makeStyles } from '@material-ui/core/styles'
import subsonic from '../subsonic'
import config from '../config'

const useStyles = makeStyles({
  comboIcon: {
    position: 'relative',
    display: 'inline-flex',
    width: 24,
    height: 24,
  },
  starPart: {
    position: 'absolute',
    top: -1,
    left: 0,
    fontSize: 20,
    opacity: 0.9,
  },
  heartPart: {
    position: 'absolute',
    bottom: -1,
    right: -2,
    fontSize: 14,
    opacity: 0.9,
  },
  ratingRow: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    marginBottom: 12,
  },
  loveRow: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
  },
})

const ComboIcon = () => {
  const classes = useStyles()
  return (
    <span className={classes.comboIcon}>
      <StarBorderIcon className={classes.starPart} />
      <FavoriteBorderIcon className={classes.heartPart} />
    </span>
  )
}

export const BatchRateButton = ({
  resource,
  selectedIds,
  className,
  label: labelOverride,
}) => {
  const [open, setOpen] = useState(false)
  const [rating, setRating] = useState(0)
  const [starred, setStarred] = useState(null) // null = don't change, true/false = set
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const unselectAll = useUnselectAll()
  const notify = useNotify()
  const refresh = useRefresh()
  const classes = useStyles()

  const handleOpen = () => {
    setRating(0)
    setStarred(null)
    setOpen(true)
  }

  const handleApply = async () => {
    setOpen(false)
    try {
      for (const id of selectedIds) {
        if (rating > 0) {
          await subsonic.setRating(id, rating)
        }
        if (starred === true) {
          await subsonic.star(id)
        } else if (starred === false) {
          await subsonic.unstar(id)
        }
      }
      // Clear rating if "delete" was chosen (rating === -1)
      if (rating === -1) {
        for (const id of selectedIds) {
          await subsonic.setRating(id, 0)
        }
      }
      // Force React-Admin to re-fetch the records, then refresh the view
      await dataProvider.getMany(resource, { ids: selectedIds })
      notify('message.batchRateSuccess', { type: 'info' })
      refresh()
    } catch (e) {
      notify('ra.page.error', { type: 'warning' })
    }
    unselectAll(resource)
  }

  const caption = labelOverride || translate('resources.song.actions.batchRate')

  return (
    <>
      <Button
        aria-label={caption}
        onClick={handleOpen}
        label={caption}
        className={className}
      >
        <ComboIcon />
      </Button>
      <Dialog open={open} onClose={() => setOpen(false)}>
        <DialogTitle style={{ textAlign: 'center' }}>{translate('resources.song.actions.batchRateTitle', { smart_count: selectedIds?.length || 0 })}</DialogTitle>
        <DialogContent>
          <div className={classes.ratingRow}>
            <Tooltip title={translate('resources.song.actions.clearRating')}>
              <IconButton
                size="small"
                onClick={() => setRating(rating === -1 ? 0 : -1)}
                color={rating === -1 ? 'secondary' : 'default'}
              >
                <ClearIcon />
              </IconButton>
            </Tooltip>
            <Rating
              value={rating > 0 ? rating : 0}
              onChange={(_, val) => setRating(val || 0)}
              emptyIcon={<StarBorderIcon fontSize="inherit" />}
            />
          </div>
          {config.enableFavourites && (
            <div className={classes.loveRow}>
              <Tooltip title={translate('resources.song.actions.unlike')}>
                <IconButton
                  size="small"
                  onClick={() => setStarred(starred === false ? null : false)}
                  color={starred === false ? 'secondary' : 'default'}
                >
                  <FavoriteBorderIcon />
                </IconButton>
              </Tooltip>
              <Tooltip title={translate('resources.song.actions.like')}>
                <IconButton
                  size="small"
                  onClick={() => setStarred(starred === true ? null : true)}
                  color={starred === true ? 'secondary' : 'default'}
                >
                  <FavoriteIcon />
                </IconButton>
              </Tooltip>
            </div>
          )}
        </DialogContent>
        <DialogActions>
          <MuiButton onClick={() => setOpen(false)}>
            {translate('ra.action.cancel')}
          </MuiButton>
          <MuiButton
            onClick={handleApply}
            color="primary"
            disabled={rating === 0 && starred === null}
          >
            {translate('ra.action.confirm')}
          </MuiButton>
        </DialogActions>
      </Dialog>
    </>
  )
}
