import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  TextField,
} from '@material-ui/core'
import { createRef, useCallback, useState } from 'react'
import { useNotify, useTranslate } from 'react-admin'
import { useDispatch, useSelector } from 'react-redux'
import { closeWebhookDialog } from '../actions'
import { httpClient } from '../dataProvider'

export const WebhookDialog = ({ setLinked }) => {
  const dispatch = useDispatch()
  const notify = useNotify()
  const translate = useTranslate()

  const { open, name, url } = useSelector((state) => state.webhookTokenDialog)

  const [checking, setChecking] = useState(false)
  const [token, setToken] = useState('')
  const inputRef = createRef()

  const handleChange = (event) => {
    setToken(event.target.value)
  }

  const handleSave = useCallback(
    (event) => {
      setChecking(true)
      httpClient(`/api/webhook/${name}/link`, {
        method: 'PUT',
        body: JSON.stringify({ token: token }),
      })
        .then((response) => {
          notify('message.webhookLinkSuccess', 'success', {
            name,
            user: response.json.user,
          })
          setLinked(name, true)
          setToken('')
        })
        .catch((error) => {
          notify('message.webhookLinkFailure', 'warning', {
            name,
            error: error.body?.error || error.message,
          })
          setLinked(name, false)
        })
        .finally(() => {
          setChecking(false)
          dispatch(closeWebhookDialog())
          event.stopPropagation()
        })
    },
    [dispatch, name, notify, setLinked, token]
  )

  const handleClose = (e) => {
    dispatch(closeWebhookDialog())
    e.stopPropagation()
  }

  const handleClickClose = (event) => {
    if (!checking) {
      dispatch(closeWebhookDialog())
      event.stopPropagation()
    }
  }

  const handleKeyPress = useCallback(
    (event) => {
      if (event.key === 'Enter' && token !== '') {
        handleSave(event)
      }
    },
    [token, handleSave]
  )

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      onBackdropClick={handleClose}
      aria-labelledby="webhook-dialog"
      fullWidth={true}
      maxWidth={'sm'}
    >
      <DialogTitle id="webhook-dialog">
        {translate('message.webhookTitle', { name })}
      </DialogTitle>
      <DialogContent>
        <DialogContentText>
          {translate('resources.user.message.webhookToken', { name, url })}
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
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClickClose} disabled={checking} color="primary">
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
  )
}
