import React, { createRef, useCallback, useState } from 'react'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  LinearProgress,
  Link,
  TextField,
} from '@material-ui/core'
import { useNotify, useTranslate } from 'react-admin'
import { useDispatch, useSelector } from 'react-redux'
import { closeListenBrainzTokenDialog } from '../actions'
import { httpClient } from '../dataProvider'

export const ListenBrainzTokenDialog = ({ setLinked }) => {
  const dispatch = useDispatch()
  const notify = useNotify()
  const translate = useTranslate()
  const { open } = useSelector((state) => state.listenBrainzTokenDialog)
  const [token, setToken] = useState('')
  const [checking, setChecking] = useState(false)
  const inputRef = createRef()

  const handleChange = (event) => {
    setToken(event.target.value)
  }

  const handleLinkClick = (event) => {
    inputRef.current.focus()
  }

  const handleSave = useCallback(
    (event) => {
      setChecking(true)
      httpClient('/api/listenbrainz/link', {
        method: 'PUT',
        body: JSON.stringify({ token: token }),
      })
        .then((response) => {
          notify('message.listenBrainzLinkSuccess', 'success', {
            user: response.json.user,
          })
          setLinked(true)
          setToken('')
        })
        .catch((error) => {
          notify('message.listenBrainzLinkFailure', 'warning', {
            error: error.body?.error || error.message,
          })
          setLinked(false)
        })
        .finally(() => {
          setChecking(false)
          dispatch(closeListenBrainzTokenDialog())
          event.stopPropagation()
        })
    },
    [dispatch, notify, setLinked, token],
  )

  const handleClickClose = (event) => {
    if (!checking) {
      dispatch(closeListenBrainzTokenDialog())
      event.stopPropagation()
    }
  }

  const handleKeyPress = useCallback(
    (event) => {
      if (event.key === 'Enter' && token !== '') {
        handleSave(event)
      }
    },
    [token, handleSave],
  )

  return (
    <>
      <Dialog
        open={open}
        onClose={handleClickClose}
        aria-labelledby="form-dialog-listenbrainz-token"
        fullWidth={true}
        maxWidth="md"
      >
        <DialogTitle id="form-dialog-listenbrainz-token">
          ListenBrainz
        </DialogTitle>
        <DialogContent>
          <DialogContentText>
            {translate('resources.user.message.listenBrainzToken')}{' '}
            <Link
              href="https://listenbrainz.org/profile/"
              onClick={handleLinkClick}
              target="_blank"
            >
              {translate('resources.user.message.clickHereForToken')}
            </Link>
          </DialogContentText>
          <TextField
            value={token}
            onKeyPress={handleKeyPress}
            onChange={handleChange}
            disabled={checking}
            required
            autoFocus
            fullWidth={true}
            variant={'outlined'}
            label={translate('resources.user.fields.token')}
            inputRef={inputRef}
          />
          {checking && <LinearProgress />}
        </DialogContent>
        <DialogActions>
          <Button
            onClick={handleClickClose}
            disabled={checking}
            color="primary"
          >
            {translate('ra.action.cancel')}
          </Button>
          <Button
            onClick={handleSave}
            disabled={checking || token === ''}
            color="primary"
            data-testid="listenbrainz-token-save"
          >
            {translate('ra.action.save')}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  )
}
