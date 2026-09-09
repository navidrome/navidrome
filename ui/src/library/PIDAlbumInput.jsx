import React, { useState } from 'react'
import { TextInput, required, useTranslate } from 'react-admin'
import { useForm, useFormState } from 'react-final-form'
import { MenuItem, TextField } from '@material-ui/core'
import { PID_CUSTOM, PID_DEFAULT, PID_FOLDER, pidAlbumMode } from './pidUtils'

const PIDAlbumInput = () => {
  const translate = useTranslate()
  const form = useForm()
  const { values } = useFormState({ subscription: { values: true } })
  const [mode, setMode] = useState(() => pidAlbumMode(values.pidAlbum))

  const handleChange = (event) => {
    const next = event.target.value
    setMode(next)
    if (next === PID_FOLDER) {
      form.change('pidAlbum', PID_FOLDER)
    } else {
      form.change('pidAlbum', '')
    }
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
          validate={[required('resources.library.validation.pidAlbumCustomRequired')]}
          data-testid="pidAlbum-custom-input"
          fullWidth
          variant="outlined"
        />
      )}
    </>
  )
}

export default PIDAlbumInput
