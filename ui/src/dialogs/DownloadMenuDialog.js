import React, { useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { ReferenceManyField, useTranslate } from 'react-admin'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  FormGroup,
  MenuItem,
  Switch,
  TextField,
} from '@material-ui/core'
import subsonic from '../subsonic'
import { closeDownloadMenu } from '../actions'
import { formatBytes } from '../utils'

const DownloadTranscodings = (props) => {
  const translate = useTranslate()

  return (
    <>
      <TextField
        fullWidth
        id="downloadFormat"
        select
        label={translate('resources.transcoding.fields.targetFormat')}
        onChange={(e) => props.onChange(e.target.value)}
        value={props.value}
      >
        {Object.values(props.data).map((transcoding) => (
          <MenuItem key={transcoding.id} value={transcoding.targetFormat}>
            {transcoding.name}
          </MenuItem>
        ))}
      </TextField>
    </>
  )
}

const DownloadMenuDialog = () => {
  const { open, record, recordType } = useSelector(
    (state) => state.downloadMenuDialog
  )
  const dispatch = useDispatch()
  const translate = useTranslate()

  const [originalFormat, setUseOriginalFormat] = useState(true)
  const [targetFormat, setTargetFormat] = useState('')
  const [targetRate, setTargetRate] = useState(0)

  const handleClose = (e) => {
    dispatch(closeDownloadMenu())
    e.stopPropagation()
  }

  const handleDownload = (e) => {
    if (record) {
      subsonic.download(
        record.id,
        originalFormat ? 'raw' : targetFormat,
        targetRate
      )
      dispatch(closeDownloadMenu())
    }
    e.stopPropagation()
  }

  const handleOriginal = (e) => {
    const original = e.target.checked

    setUseOriginalFormat(original)

    if (original) {
      setTargetFormat('')
      setTargetRate(0)
    }
  }

  const type = recordType
    ? translate(`resources.${recordType}.name`, {
        smart_count: 1,
      }).toLocaleLowerCase()
    : ''

  return (
    <>
      <Dialog
        open={open}
        onClose={handleClose}
        onBackdropClick={handleClose}
        aria-labelledby="download-dialog"
        fullWidth={true}
        maxWidth={'sm'}
      >
        <DialogTitle id="download-dialog">
          {record &&
            `${translate('resources.album.actions.download')} ${type} ${
              record.name || record.title
            } (${formatBytes(record.size)})`}
        </DialogTitle>
        <DialogContent>
          <Box
            component="form"
            sx={{
              '& .MuiTextField-root': { m: 1, width: '25ch' },
            }}
          >
            <div>
              <FormGroup>
                <FormControlLabel
                  control={<Switch checked={originalFormat} />}
                  label={translate('message.originalFormat')}
                  onChange={handleOriginal}
                />
              </FormGroup>
              {!originalFormat && (
                <>
                  <ReferenceManyField
                    fullWidth
                    source=""
                    target="name"
                    reference="transcoding"
                    sort={{ field: 'name', order: 'ASC' }}
                  >
                    <DownloadTranscodings
                      onChange={setTargetFormat}
                      value={targetFormat}
                    />
                  </ReferenceManyField>
                  <TextField
                    fullWidth
                    id="downloadRate"
                    select
                    label={translate('resources.player.fields.maxBitRate')}
                    value={targetRate}
                    onChange={(e) => setTargetRate(e.target.value)}
                  >
                    <MenuItem value={0}>-</MenuItem>
                    {[32, 48, 64, 80, 96, 112, 128, 160, 192, 256, 320].map(
                      (bits) => (
                        <MenuItem key={bits} value={bits}>
                          {bits}
                        </MenuItem>
                      )
                    )}
                  </TextField>
                </>
              )}
            </div>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button
            onClick={handleDownload}
            color="primary"
            disabled={!originalFormat && !targetFormat}
          >
            {translate('resources.album.actions.download')}
          </Button>
          <Button onClick={handleClose} color="secondary">
            {translate('ra.action.close')}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  )
}

export default DownloadMenuDialog
