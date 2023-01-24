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
import { useEffect, useState } from 'react'
import { shareUrl } from '../utils'
import { useTranscodingOptions } from './useTranscodingOptions'
import { useDispatch, useSelector } from 'react-redux'
import { closeShareMenu } from '../actions'

export const ShareDialog = () => {
  const {
    open,
    ids,
    resource,
    name,
    label = 'message.shareDialogTitle',
  } = useSelector((state) => state.shareDialog)
  const dispatch = useDispatch()
  const notify = useNotify()
  const translate = useTranslate()
  const [description, setDescription] = useState('')
  useEffect(() => {
    setDescription('')
  }, [ids])
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
        navigator.clipboard
          .writeText(url)
          .then(() => {
            notify('message.shareSuccess', 'info', { url }, false, 0)
          })
          .catch((err) => {
            notify(
              translate('message.shareFailure', { url }) + ': ' + err.message,
              {
                type: 'warning',
                multiLine: true,
                duration: 0,
              }
            )
          })
      },
      onFailure: (error) =>
        notify(translate('ra.page.error') + ': ' + error.message, {
          type: 'warning',
        }),
    }
  )

  const handleShare = (e) => {
    createShare()
    dispatch(closeShareMenu())
    e.stopPropagation()
  }

  const handleClose = (e) => {
    dispatch(closeShareMenu())
    e.stopPropagation()
  }

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      onBackdropClick={handleClose}
      aria-labelledby="share-dialog"
      fullWidth={true}
      maxWidth={'sm'}
    >
      <DialogTitle id="share-dialog">
        {resource &&
          translate(label, {
            resource: translate(`resources.${resource}.name`, {
              smart_count: ids?.length,
            }).toLocaleLowerCase(),
            name,
            smart_count: ids?.length,
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
        <Button onClick={handleClose} color="primary">
          {translate('ra.action.close')}
        </Button>
        <Button onClick={handleShare} color="primary">
          {translate('ra.action.share')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
