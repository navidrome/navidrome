import React, { useState } from 'react'
import { TextInput, required, useTranslate } from 'react-admin'
import { useField } from 'react-final-form'
import { MenuItem, TextField } from '@material-ui/core'
import {
  PID_CUSTOM,
  PID_DEFAULT,
  PID_FOLDER,
  pidAlbumMode,
  parsePidField,
} from './pidUtils'

const PIDAlbumInput = () => {
  const translate = useTranslate()
  // Keeps pidAlbum registered in every mode, otherwise final-form's pristine
  // flag never clears when Default/Folder is chosen (no field is registered).
  const { input } = useField('pidAlbum', { parse: parsePidField })
  const [mode, setMode] = useState(() => pidAlbumMode(input.value))

  const handleChange = (event) => {
    const next = event.target.value
    setMode(next)
    input.onChange(next === PID_FOLDER ? PID_FOLDER : '')
  }

  return (
    <>
      <TextField
        select
        fullWidth
        variant="outlined"
        margin="dense"
        value={mode}
        onChange={handleChange}
        label={translate('resources.library.fields.pidAlbum')}
        SelectProps={{
          SelectDisplayProps: { 'data-testid': 'pidAlbum-mode-select' },
        }}
      >
        <MenuItem value={PID_DEFAULT}>
          {translate('resources.library.pid.default')}
        </MenuItem>
        <MenuItem value={PID_FOLDER}>
          {translate('resources.library.pid.folder')}
        </MenuItem>
        <MenuItem value={PID_CUSTOM}>
          {translate('resources.library.pid.custom')}
        </MenuItem>
      </TextField>
      {mode === PID_CUSTOM && (
        <TextInput
          source="pidAlbum"
          label={translate('resources.library.fields.pidAlbum')}
          helperText={translate('resources.library.pid.customHelp')}
          validate={[
            required('resources.library.validation.pidAlbumCustomRequired'),
          ]}
          parse={parsePidField}
          data-testid="pidAlbum-custom-input"
          fullWidth
          variant="outlined"
        />
      )}
    </>
  )
}

export default PIDAlbumInput
