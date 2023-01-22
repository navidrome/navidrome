import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  Switch,
} from '@material-ui/core'
import {
  SelectInput,
  SimpleForm,
  useCreate,
  useGetList,
  useNotify,
} from 'react-admin'
import { useMemo, useState } from 'react'
import { shareUrl } from '../utils'
import config from '../config'
import { DEFAULT_SHARE_BITRATE } from '../consts'

export const ShareDialog = ({ open, close, onClose, ids, resource, title }) => {
  const notify = useNotify()
  const [format, setFormat] = useState(config.defaultDownsamplingFormat)
  const [maxBitRate, setMaxBitRate] = useState(DEFAULT_SHARE_BITRATE)
  const [originalFormat, setUseOriginalFormat] = useState(true)
  const { data: formats, loading: loadingFormats } = useGetList(
    'transcoding',
    {
      page: 1,
      perPage: 1000,
    },
    { field: 'name', order: 'ASC' }
  )

  const formatOptions = useMemo(
    () =>
      loadingFormats
        ? []
        : Object.values(formats).map((f) => {
            return { id: f.targetFormat, name: f.targetFormat }
          }),
    [formats, loadingFormats]
  )

  const handleOriginal = (e) => {
    const original = e.target.checked

    setUseOriginalFormat(original)

    if (original) {
      setFormat('')
      setMaxBitRate(0)
    }
  }

  const [createShare] = useCreate(
    'share',
    {
      resourceType: resource,
      resourceIds: ids?.join(','),
      ...(!originalFormat && { format }),
      ...(!originalFormat && { maxBitRate }),
    },
    {
      onSuccess: (res) => {
        const url = shareUrl(res?.data?.id)
        close()
        navigator.clipboard
          .writeText(url)
          .then(() => {
            notify(`URL copied to clipboard: ${url}`, {
              type: 'info',
              multiLine: true,
              duration: 0,
            })
          })
          .catch((err) => {
            notify(`Error copying URL ${url} to clipboard: ${err.message}`, {
              type: 'warning',
              multiLine: true,
              duration: 0,
            })
          })
      },
      onFailure: (error) =>
        notify(`Error sharing media: ${error.message}`, { type: 'warning' }),
    }
  )

  return (
    <Dialog
      open={open}
      onClose={onClose}
      onBackdropClick={onClose}
      aria-labelledby="share-dialog"
      fullWidth={true}
      maxWidth={'sm'}
    >
      <DialogTitle id="share-dialog">{title}</DialogTitle>
      <DialogContent>
        <SimpleForm toolbar={null} variant={'outlined'}>
          <FormControlLabel
            control={<Switch checked={originalFormat} />}
            label={'Share in original format'}
            onChange={handleOriginal}
          />
          {!originalFormat && (
            <SelectInput
              source="format"
              defaultValue={format}
              choices={formatOptions}
              onChange={(event) => {
                setFormat(event.target.value)
              }}
            />
          )}
          {!originalFormat && (
            <SelectInput
              source="maxBitRate"
              defaultValue={maxBitRate}
              choices={[
                { id: 32, name: '32' },
                { id: 48, name: '48' },
                { id: 64, name: '64' },
                { id: 80, name: '80' },
                { id: 96, name: '96' },
                { id: 112, name: '112' },
                { id: 128, name: '128' },
                { id: 160, name: '160' },
                { id: 192, name: '192' },
                { id: 256, name: '256' },
                { id: 320, name: '320' },
              ]}
              onChange={(event) => {
                setMaxBitRate(event.target.value)
              }}
            />
          )}
        </SimpleForm>
      </DialogContent>
      <DialogActions>
        <Button onClick={createShare} color="primary">
          Share
        </Button>
        <Button onClick={onClose} color="primary">
          Cancel
        </Button>
      </DialogActions>
    </Dialog>
  )
}
