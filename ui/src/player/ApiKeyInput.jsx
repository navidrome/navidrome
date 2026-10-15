import React from 'react'
import PropTypes from 'prop-types'
import { useInput, useNotify, useTranslate } from 'react-admin'
import { Button, TextField } from '@material-ui/core'
import { FaKey } from 'react-icons/fa'
import { MdContentCopy, MdDelete, MdRefresh } from 'react-icons/md'
import { isWritable } from '../common/playlistUtils'
import { generateApiKey } from './apiKey'

const identity = (v) => v
const MASK = '•'.repeat(26)

const ApiKeyInput = ({ record, isCreate, fullWidth, className, ...props }) => {
  const translate = useTranslate()
  const notify = useNotify()
  // Identity format/parse keep "" (revoke) distinct from undefined (untouched)
  const {
    input: { value, onChange },
    meta: { error, touched },
  } = useInput({ ...props, format: identity, parse: identity })

  const isOwner = isCreate || record?.userId === localStorage.getItem('userId')
  const pending = !!value
  const revoking = value === '' && !!record?.hasApiKey
  const saved = value == null && !!record?.hasApiKey
  const hasKey = pending || saved

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

  const helperText = pending
    ? 'resources.player.message.apiKeyPending'
    : revoking
      ? 'resources.player.message.apiKeyRevokePending'
      : saved
        ? 'resources.player.message.apiKeyActive'
        : isOwner
          ? 'resources.player.message.apiKeyNone'
          : 'resources.player.message.apiKeyNoneOther'

  return (
    <div>
      <TextField
        label={translate('resources.player.fields.hasApiKey')}
        value={pending ? value : saved ? MASK : ''}
        variant="outlined"
        margin="dense"
        fullWidth={fullWidth}
        className={className}
        // Monospace keeps the whole key visible in a standard-width input
        inputProps={{
          style: { fontFamily: 'monospace', fontSize: '0.875rem' },
        }}
        InputProps={{ readOnly: true }}
        error={!!(touched && error)}
        helperText={translate(touched && error ? error : helperText)}
      />
      <div>
        {pending && (
          <Button startIcon={<MdContentCopy />} onClick={copy}>
            {translate('resources.player.actions.copyApiKey')}
          </Button>
        )}
        {isOwner && (
          <Button
            startIcon={hasKey ? <MdRefresh /> : <FaKey />}
            onClick={() => onChange(generateApiKey())}
          >
            {translate(
              hasKey
                ? 'resources.player.actions.regenerateApiKey'
                : 'resources.player.actions.generateApiKey',
            )}
          </Button>
        )}
        {saved && isWritable(record?.userId) && (
          <Button startIcon={<MdDelete />} onClick={() => onChange('')}>
            {translate('resources.player.actions.revokeApiKey')}
          </Button>
        )}
      </div>
    </div>
  )
}

ApiKeyInput.propTypes = {
  source: PropTypes.string.isRequired,
  record: PropTypes.object,
  isCreate: PropTypes.bool,
  fullWidth: PropTypes.bool,
  className: PropTypes.string,
  validate: PropTypes.oneOfType([PropTypes.func, PropTypes.array]),
}

export default ApiKeyInput
