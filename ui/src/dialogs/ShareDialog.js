import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
} from '@material-ui/core'
import {
  SimpleForm,
  TextInput,
  useCreate,
  useNotify,
  useTranslate,
} from 'react-admin'
import { useState } from 'react'
import { shareUrl } from '../utils'
import { useTranscodingOptions } from './useTranscodingOptions'

export const ShareDialog = ({ open, onClose, ids, resource, name }) => {
  const notify = useNotify()
  const translate = useTranslate()
  const [description, setDescription] = useState('')
  const { TranscodingOptionsInput, format, maxBitRate, originalFormat } =
    useTranscodingOptions()
  const [createShare] = useCreate(
    'share',
    {
      resourceType: resource,
      resourceIds: ids?.join(','),
      description,
      ...(!originalFormat && { format }),
      ...(!originalFormat && { maxBitRate }),
    },
    {
      onSuccess: (res) => {
        const url = shareUrl(res?.data?.id)
        onClose()
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
      <DialogTitle id="share-dialog">
        {translate('message.shareDialogTitle', {
          resource: translate(`resources.${resource}.name`, {
            smart_count: ids?.length,
          }).toLocaleLowerCase(),
          name,
        })}
      </DialogTitle>
      <DialogContent>
        <SimpleForm toolbar={null} variant={'outlined'}>
          <TextInput
            source="description"
            fullWidth
            onChange={(event) => {
              setDescription(event.target.value)
            }}
          />
          <TranscodingOptionsInput
            fullWidth
            label={translate('message.shareOriginalFormat')}
          />
        </SimpleForm>
      </DialogContent>
      <DialogActions>
        <Button onClick={createShare} color="primary">
          {translate('ra.action.share')}
        </Button>
        <Button onClick={onClose} color="primary">
          {translate('ra.action.close')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
