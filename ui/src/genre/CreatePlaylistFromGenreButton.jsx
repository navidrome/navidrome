import { useState } from 'react'
import PropTypes from 'prop-types'
import { useDispatch } from 'react-redux'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  TextField,
  FormControlLabel,
  Checkbox,
} from '@material-ui/core'
import PlaylistAddIcon from '@material-ui/icons/PlaylistAdd'
import { Button as RaButton, useNotify, useTranslate } from 'react-admin'
import { httpClient } from '../dataProvider'
import { REST_URL } from '../consts'
import { openAddToPlaylist } from '../actions'
import { DialogTitle } from '../dialogs/DialogTitle'

const DEFAULT_TRACK_COUNT = 50

export const CreatePlaylistFromGenreButton = ({ record }) => {
  const translate = useTranslate()
  const notify = useNotify()
  const dispatch = useDispatch()
  const [open, setOpen] = useState(false)
  const [trackCount, setTrackCount] = useState(DEFAULT_TRACK_COUNT)
  const [excludeSkipped, setExcludeSkipped] = useState(true)
  const [loading, setLoading] = useState(false)

  if (!record) return null

  const handleClose = () => setOpen(false)

  const handleSubmit = () => {
    setLoading(true)
    const params = new URLSearchParams({
      count: trackCount || DEFAULT_TRACK_COUNT,
      excludeSkipped: excludeSkipped ? 'true' : 'false',
    })
    httpClient(`${REST_URL}/genre/${record.id}/randomSongs?${params}`)
      .then((res) => {
        const ids = res.json || []
        if (ids.length === 0) {
          notify('resources.genre.createPlaylist.empty', { type: 'warning' })
          return
        }
        dispatch(openAddToPlaylist({ selectedIds: ids }))
      })
      .catch(() => notify('ra.page.error', { type: 'warning' }))
      .finally(() => {
        setLoading(false)
        setOpen(false)
      })
  }

  return (
    <>
      <RaButton
        onClick={() => setOpen(true)}
        label={translate('resources.genre.createPlaylist.action')}
      >
        <PlaylistAddIcon />
      </RaButton>
      <Dialog open={open} onClose={handleClose}>
        <DialogTitle onClose={handleClose}>
          {translate('resources.genre.createPlaylist.title', {
            genre: record.name,
          })}
        </DialogTitle>
        <DialogContent>
          <TextField
            type="number"
            fullWidth
            margin="normal"
            label={translate('resources.genre.createPlaylist.trackCount')}
            value={trackCount}
            onChange={(e) => setTrackCount(Number(e.target.value))}
            inputProps={{ min: 1, max: 500 }}
          />
          <FormControlLabel
            control={
              <Checkbox
                checked={excludeSkipped}
                onChange={(e) => setExcludeSkipped(e.target.checked)}
              />
            }
            label={translate('resources.genre.createPlaylist.excludeSkipped')}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClose}>{translate('ra.action.cancel')}</Button>
          <Button
            onClick={handleSubmit}
            color="primary"
            disabled={loading}
            variant="contained"
          >
            {translate('resources.genre.createPlaylist.submit')}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  )
}

CreatePlaylistFromGenreButton.propTypes = {
  record: PropTypes.object,
}
