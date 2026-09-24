import React, { useState } from 'react'
import PropTypes from 'prop-types'
import {
  Button,
  Confirm,
  useDataProvider,
  useNotify,
  usePermissions,
  useRecordContext,
  useRefresh,
  useTranslate,
} from 'react-admin'
import {
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  TextField,
  Typography,
} from '@material-ui/core'
import { FaKey } from 'react-icons/fa'
import { MdContentCopy, MdDelete, MdRefresh } from 'react-icons/md'

const PlayerApiKey = (props) => {
  const record = useRecordContext(props)
  const translate = useTranslate()
  const notify = useNotify()
  const refresh = useRefresh()
  const dataProvider = useDataProvider()
  const { permissions } = usePermissions()
  const [confirm, setConfirm] = useState(null)
  const [newKey, setNewKey] = useState(null)
  const [loading, setLoading] = useState(false)

  if (!record?.id) {
    return null
  }
  const isOwner = record.userId === localStorage.getItem('userId')
  const canRevoke = isOwner || permissions === 'admin'

  const run = (call, onSuccess) => {
    setLoading(true)
    call(record.id)
      .then(onSuccess)
      .catch(() =>
        notify('resources.player.notifications.apiKeyError', 'warning'),
      )
      .finally(() => {
        setLoading(false)
        setConfirm(null)
      })
  }
  const generate = () =>
    run(dataProvider.generatePlayerApiKey, ({ data }) => setNewKey(data.apiKey))
  const revoke = () =>
    run(dataProvider.revokePlayerApiKey, () => {
      notify('resources.player.notifications.apiKeyRevoked')
      refresh()
    })
  const copy = () => {
    if (navigator.clipboard && window.isSecureContext) {
      navigator.clipboard
        .writeText(newKey)
        .then(() => notify('resources.player.notifications.apiKeyCopied'))
    } else {
      prompt(translate('message.shareCopyToClipboard'), newKey)
    }
  }
  const closeKeyDialog = () => {
    setNewKey(null)
    refresh()
  }

  return (
    <div>
      <Typography variant="subtitle2">
        {translate('resources.player.fields.apiKey')}
      </Typography>
      <Typography variant="body2" color="textSecondary">
        {translate(
          record.hasApiKey
            ? 'resources.player.message.apiKeyActive'
            : 'resources.player.message.apiKeyNone',
        )}
      </Typography>
      {isOwner && !record.hasApiKey && (
        <Button
          label="resources.player.actions.generateApiKey"
          onClick={generate}
          disabled={loading}
        >
          <FaKey />
        </Button>
      )}
      {isOwner && record.hasApiKey && (
        <Button
          label="resources.player.actions.regenerateApiKey"
          onClick={() => setConfirm('regenerate')}
          disabled={loading}
        >
          <MdRefresh />
        </Button>
      )}
      {canRevoke && record.hasApiKey && (
        <Button
          label="resources.player.actions.revokeApiKey"
          onClick={() => setConfirm('revoke')}
          disabled={loading}
        >
          <MdDelete />
        </Button>
      )}
      <Confirm
        isOpen={confirm === 'regenerate'}
        loading={loading}
        title="resources.player.message.regenerateApiKeyTitle"
        content="resources.player.message.regenerateApiKeyContent"
        onConfirm={generate}
        onClose={() => setConfirm(null)}
      />
      <Confirm
        isOpen={confirm === 'revoke'}
        loading={loading}
        title="resources.player.message.revokeApiKeyTitle"
        content="resources.player.message.revokeApiKeyContent"
        onConfirm={revoke}
        onClose={() => setConfirm(null)}
      />
      {/* The key is shown only once, so Escape or a stray click must not dismiss it */}
      <Dialog open={!!newKey} fullWidth>
        <DialogTitle>
          {translate('resources.player.message.apiKeyDialogTitle')}
        </DialogTitle>
        <DialogContent>
          <DialogContentText>
            {translate('resources.player.message.apiKeyDialogContent')}
          </DialogContentText>
          <TextField
            value={newKey || ''}
            fullWidth
            variant="outlined"
            InputProps={{ readOnly: true }}
          />
        </DialogContent>
        <DialogActions>
          <Button label="resources.player.actions.copyApiKey" onClick={copy}>
            <MdContentCopy />
          </Button>
          <Button label="ra.action.close" onClick={closeKeyDialog} />
        </DialogActions>
      </Dialog>
    </div>
  )
}

PlayerApiKey.propTypes = {
  record: PropTypes.object,
}

export default PlayerApiKey
