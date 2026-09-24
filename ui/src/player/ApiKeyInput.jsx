import React from 'react'
import PropTypes from 'prop-types'
import { useInput, useNotify, usePermissions, useTranslate } from 'react-admin'
import {
  Button,
  IconButton,
  InputAdornment,
  TextField,
  Tooltip,
} from '@material-ui/core'
import { FaKey } from 'react-icons/fa'
import { MdContentCopy, MdDelete, MdRefresh } from 'react-icons/md'
import { generateApiKey } from './apiKey'

const identity = (v) => v
const MASK = '•'.repeat(26)

const ApiKeyInput = ({ record, isCreate, ...props }) => {
  const translate = useTranslate()
  const notify = useNotify()
  const { permissions } = usePermissions()
  // Identity format/parse keep "" (revoke) distinct from undefined (untouched)
  const {
    input: { value, onChange },
    meta: { error, touched },
  } = useInput({ ...props, format: identity, parse: identity })

  const isOwner = isCreate || record?.userId === localStorage.getItem('userId')
  const canRevoke = !isCreate && (isOwner || permissions === 'admin')
  const pending = typeof value === 'string' && value !== ''
  const revoking = value === '' && !!record?.hasApiKey
  const saved = !pending && !revoking && !!record?.hasApiKey

  const copy = () => {
    const fallback = () =>
      prompt(translate('message.shareCopyToClipboard'), value)
    if (navigator.clipboard && window.isSecureContext) {
      navigator.clipboard
        .writeText(value)
        .then(
          () => notify('resources.player.notifications.apiKeyCopied'),
          fallback,
        )
    } else {
      fallback()
    }
  }

  let helperText = isOwner
    ? 'resources.player.message.apiKeyNone'
    : 'resources.player.message.apiKeyNoneOther'
  if (pending) helperText = 'resources.player.message.apiKeyPending'
  else if (revoking) helperText = 'resources.player.message.apiKeyRevokePending'
  else if (saved) helperText = 'resources.player.message.apiKeyActive'

  return (
    <div>
      <TextField
        label={translate('resources.player.fields.hasApiKey')}
        value={pending ? value : saved ? MASK : ''}
        variant="outlined"
        margin="dense"
        fullWidth
        InputProps={{
          readOnly: true,
          endAdornment: pending && (
            <InputAdornment position="end">
              <Tooltip title={translate('resources.player.actions.copyApiKey')}>
                <IconButton
                  aria-label={translate('resources.player.actions.copyApiKey')}
                  onClick={copy}
                  edge="end"
                >
                  <MdContentCopy />
                </IconButton>
              </Tooltip>
            </InputAdornment>
          ),
        }}
        error={!!(touched && error)}
        helperText={translate(touched && error ? error : helperText)}
      />
      {isOwner && (pending || saved) && (
        <Button
          startIcon={<MdRefresh />}
          onClick={() => onChange(generateApiKey())}
        >
          {translate('resources.player.actions.regenerateApiKey')}
        </Button>
      )}
      {isOwner && !pending && !saved && (
        <Button
          startIcon={<FaKey />}
          onClick={() => onChange(generateApiKey())}
        >
          {translate('resources.player.actions.generateApiKey')}
        </Button>
      )}
      {canRevoke && saved && (
        <Button startIcon={<MdDelete />} onClick={() => onChange('')}>
          {translate('resources.player.actions.revokeApiKey')}
        </Button>
      )}
    </div>
  )
}

ApiKeyInput.propTypes = {
  source: PropTypes.string.isRequired,
  record: PropTypes.object,
  isCreate: PropTypes.bool,
  validate: PropTypes.oneOfType([PropTypes.func, PropTypes.array]),
}

export default ApiKeyInput
