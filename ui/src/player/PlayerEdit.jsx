import React, { useState, useCallback } from 'react'
import {
  TextInput,
  BooleanInput,
  TextField,
  Edit,
  required,
  SimpleForm,
  SelectInput,
  ReferenceInput,
  useTranslate,
  useNotify,
  Button,
  useRecordContext,
} from 'react-admin'
import FileCopyIcon from '@material-ui/icons/FileCopy'
import VpnKeyIcon from '@material-ui/icons/VpnKey'
import VisibilityIcon from '@material-ui/icons/Visibility'
import VisibilityOffIcon from '@material-ui/icons/VisibilityOff'
import { Title } from '../common'
import { BITRATE_CHOICES, REST_URL } from '../consts'
import { makeStyles } from '@material-ui/core/styles'
import {
  IconButton,
  InputAdornment,
  Tooltip,
  TextField as MuiTextField,
} from '@material-ui/core'
import httpClient from '../dataProvider/httpClient.js'
import config from '../config.js'

const PlayerTitle = ({ record }) => {
  const translate = useTranslate()
  const resourceName = translate('resources.player.name', { smart_count: 1 })
  return <Title subTitle={`${resourceName} ${record ? record.name : ''}`} />
}

const useStyles = makeStyles({
  apiKeyField: {
    marginTop: '16px',
    marginBottom: '8px',
  },
  generateButton: {
    marginTop: '16px',
    marginBottom: '8px',
  },
  copyButton: {
    padding: 4,
  },
})

const ApiKeySection = () => {
  const record = useRecordContext()
  const recordId = record ? record.id : null
  const initialApiKey = record ? record.apiKey : null

  const classes = useStyles()
  const notify = useNotify()
  const [showApiKey, setShowApiKey] = useState(false)
  const [loading, setLoading] = useState(false)
  const [apiKey, setApiKey] = useState(initialApiKey)

  const generateApiKey = useCallback(async () => {
    if (!recordId) {
      notify('Player ID not available', 'error')
      return
    }

    try {
      setLoading(true)
      const { json } = await httpClient(
        `${REST_URL}/player/${recordId}/apiKey`,
        {
          method: 'POST',
        },
      )
      setApiKey(json.apiKey)
      setShowApiKey(true)
      notify('message.apiKeyGenerated', 'info')
    } catch (error) {
      notify(error.message || 'Error generating API key', 'error')
    } finally {
      setLoading(false)
    }
  }, [recordId, notify])

  const copyToClipboard = () => {
    if (apiKey) {
      navigator.clipboard.writeText(apiKey)
      notify('API key copied to clipboard', 'info')
    }
  }

  if (!recordId) {
    return <></>
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', marginTop: 16 }}>
      {apiKey ? (
        <MuiTextField
          label="API Key"
          value={showApiKey ? apiKey : '*'.repeat(apiKey.length)}
          InputProps={{
            readOnly: true,
            disableUnderline: true,
            endAdornment: (
              <InputAdornment position="end">
                <Tooltip title={showApiKey ? 'Hide' : 'Show'}>
                  <IconButton
                    aria-label="toggle api key visibility"
                    onClick={() => setShowApiKey(!showApiKey)}
                    size="small"
                  >
                    {showApiKey ? <VisibilityOffIcon /> : <VisibilityIcon />}
                  </IconButton>
                </Tooltip>
                <Tooltip title="Copy to clipboard">
                  <IconButton
                    aria-label="copy api key"
                    onClick={copyToClipboard}
                    className={classes.copyButton}
                    size="small"
                  >
                    <FileCopyIcon />
                  </IconButton>
                </Tooltip>
              </InputAdornment>
            ),
          }}
          style={{ width: 320, color: 'black' }}
        />
      ) : (
        !loading && (
          <MuiTextField
            label="API Key"
            value="No API Key Found"
            disabled
            style={{ width: 320 }}
          />
        )
      )}

      <Button
        style={{ marginTop: 12, alignSelf: 'flex-start' }}
        onClick={generateApiKey}
        label="Generate New API Key"
        startIcon={<VpnKeyIcon />}
        variant="outlined"
        disabled={loading}
      />
    </div>
  )
}

const PlayerEdit = (props) => {
  return (
    <Edit title={<PlayerTitle />} {...props}>
      <SimpleForm variant={'outlined'}>
        <TextInput source="name" validate={[required()]} />
        <ReferenceInput
          source="transcodingId"
          reference="transcoding"
          sort={{ field: 'name', order: 'ASC' }}
        >
          <SelectInput source="name" resettable />
        </ReferenceInput>
        <SelectInput source="maxBitRate" resettable choices={BITRATE_CHOICES} />
        <BooleanInput source="reportRealPath" fullWidth />
        {(config.lastFMEnabled || config.listenBrainzEnabled) && (
          <BooleanInput source="scrobbleEnabled" fullWidth />
        )}
        <TextField source="client" />
        <TextField source="userName" />

        <ApiKeySection />
      </SimpleForm>
    </Edit>
  )
}

export default PlayerEdit
