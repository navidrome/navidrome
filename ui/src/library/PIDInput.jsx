import React, { useState } from 'react'
import PropTypes from 'prop-types'
import { TextInput, required, useTranslate } from 'react-admin'
import { useField } from 'react-final-form'
import { FormHelperText, Link, MenuItem, TextField } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import {
  PID_CUSTOM,
  PID_FOLDER,
  PID_GLOBAL,
  pidModeFromValue,
  pidValueForMode,
} from './pidPresets'
import config from '../config'
import { docsUrl } from '../utils'

const PID_DOCS_URL = docsUrl('/docs/usage/pids/')

const useStyles = makeStyles((theme) => ({
  help: { marginBottom: theme.spacing(1) },
}))

// PIDInput edits a library PID override: use the global setting, a preset, or a custom spec
export const PIDInput = ({ source, label, globalValue, allowFolder }) => {
  const translate = useTranslate()
  const classes = useStyles()
  const { input } = useField(source)
  // Local state, so choosing Custom shows the text box before anything is typed
  const [mode, setMode] = useState(() =>
    pidModeFromValue(input.value, allowFolder),
  )

  const choices = [
    {
      id: PID_GLOBAL,
      name: translate('resources.library.pid.global', { value: globalValue }),
    },
    ...(allowFolder
      ? [{ id: PID_FOLDER, name: translate('resources.library.pid.folder') }]
      : []),
    { id: PID_CUSTOM, name: translate('resources.library.pid.custom') },
  ]

  const handleModeChange = (event) => {
    const newMode = event.target.value
    setMode(newMode)
    input.onChange(pidValueForMode(newMode, globalValue))
  }

  return (
    <>
      <TextField
        id={`${source}-mode`}
        select
        fullWidth
        variant="outlined"
        margin="dense"
        label={label}
        value={mode}
        onChange={handleModeChange}
      >
        {choices.map((choice) => (
          <MenuItem key={choice.id} value={choice.id}>
            {choice.name}
          </MenuItem>
        ))}
      </TextField>
      {mode === PID_CUSTOM && (
        <>
          <TextInput
            source={source}
            label={translate('resources.library.pid.spec')}
            validate={[required()]}
            fullWidth
            variant="outlined"
            helperText={false}
          />
          <FormHelperText className={classes.help}>
            {translate('resources.library.pid.help')}{' '}
            <Link href={PID_DOCS_URL} target="_blank" rel="noopener noreferrer">
              {translate('resources.library.pid.docs')}
            </Link>
          </FormHelperText>
        </>
      )}
    </>
  )
}

PIDInput.propTypes = {
  source: PropTypes.string.isRequired,
  label: PropTypes.string.isRequired,
  globalValue: PropTypes.string,
  allowFolder: PropTypes.bool,
}

export const PIDInputs = () => {
  const translate = useTranslate()
  return (
    <>
      <PIDInput
        source="pidAlbum"
        label={translate('resources.library.fields.pidAlbum')}
        globalValue={config.pidAlbum}
        allowFolder
      />
      <PIDInput
        source="pidTrack"
        label={translate('resources.library.fields.pidTrack')}
        globalValue={config.pidTrack}
      />
    </>
  )
}
