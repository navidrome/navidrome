import { useState } from 'react'
import PropTypes from 'prop-types'
import { useDataProvider, useNotify, useTranslate } from 'react-admin'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  TextField,
} from '@material-ui/core'

export const QuickConnectDialog = ({ open, onClose }) => {
  const translate = useTranslate()
  const notify = useNotify()
  const dataProvider = useDataProvider()
  const [code, setCode] = useState('')
  const [pending, setPending] = useState(null)
  const [loading, setLoading] = useState(false)

  const handleClose = () => {
    setCode('')
    setPending(null)
    onClose()
  }

  const handleError = (error) => {
    setPending(null)
    const invalid = error?.status === 404 || error?.status === 409
    notify(
      invalid ? 'message.quickConnectInvalidCode' : 'message.quickConnectError',
      'warning',
    )
  }

  const run = (request, onSuccess) => {
    setLoading(true)
    request
      .then(({ data }) => onSuccess(data))
      .catch(handleError)
      .finally(() => setLoading(false))
  }

  const lookup = (event) => {
    event.preventDefault()
    run(dataProvider.lookupQuickConnect(code), setPending)
  }

  const approve = () =>
    run(dataProvider.authorizeQuickConnect(code), (data) => {
      notify('message.quickConnectApproved', 'success', {
        app: data.appName,
        device: data.deviceName,
      })
      handleClose()
    })

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      aria-labelledby="quick-connect-dialog"
      fullWidth
      maxWidth="xs"
    >
      <DialogTitle id="quick-connect-dialog">
        {translate('menu.quickConnect.name')}
      </DialogTitle>
      {pending ? (
        <>
          <DialogContent>
            <DialogContentText>
              {translate('menu.quickConnect.confirm', {
                app: pending.appName,
                version: pending.appVersion,
                device: pending.deviceName,
              })}
            </DialogContentText>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setPending(null)} color="primary">
              {translate('ra.action.back')}
            </Button>
            <Button onClick={approve} color="primary" disabled={loading}>
              {translate('menu.quickConnect.approve')}
            </Button>
          </DialogActions>
        </>
      ) : (
        <form onSubmit={lookup}>
          <DialogContent>
            <DialogContentText>
              {translate('menu.quickConnect.help')}
            </DialogContentText>
            <TextField
              id="quickConnectCode"
              variant="outlined"
              fullWidth
              autoFocus
              label={translate('menu.quickConnect.code')}
              value={code}
              onChange={(event) => setCode(event.target.value)}
              inputProps={{ inputMode: 'numeric', autoComplete: 'off' }}
            />
          </DialogContent>
          <DialogActions>
            <Button onClick={handleClose} color="primary">
              {translate('ra.action.cancel')}
            </Button>
            <Button
              type="submit"
              color="primary"
              disabled={!code.trim() || loading}
            >
              {translate('menu.quickConnect.continue')}
            </Button>
          </DialogActions>
        </form>
      )}
    </Dialog>
  )
}

QuickConnectDialog.propTypes = {
  open: PropTypes.bool.isRequired,
  onClose: PropTypes.func.isRequired,
}
