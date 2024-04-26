import { SimpleForm, useTranslate } from 'react-admin'
import { useDispatch, useSelector } from 'react-redux'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
} from '@material-ui/core'
import subsonic from '../subsonic'
import { closeDownloadMenu } from '../actions'
import { formatBytes } from '../utils'
import { useTranscodingOptions } from './useTranscodingOptions'

const DownloadMenuDialog = () => {
  const { open, record, recordType } = useSelector(
    (state) => state.downloadMenuDialog,
  )
  const dispatch = useDispatch()
  const translate = useTranslate()

  const { TranscodingOptionsInput, format, maxBitRate, originalFormat } =
    useTranscodingOptions()

  const handleClose = (e) => {
    dispatch(closeDownloadMenu())
    e.stopPropagation()
  }

  const handleDownload = (e) => {
    if (record) {
      const id = record.mediaFileId || record.id
      if (originalFormat) {
        subsonic.download(id, 'raw')
      } else {
        subsonic.download(id, format, maxBitRate?.toString())
      }
      dispatch(closeDownloadMenu())
    }
    e.stopPropagation()
  }

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      aria-labelledby="download-dialog"
      fullWidth={true}
      maxWidth={'sm'}
    >
      <DialogTitle id="download-dialog">
        {recordType &&
          translate('message.downloadDialogTitle', {
            resource: translate(`resources.${recordType}.name`, {
              smart_count: 1,
            }).toLocaleLowerCase(),
            name: record?.name || record?.title,
            size: formatBytes(record?.size),
          })}
      </DialogTitle>
      <DialogContent>
        <SimpleForm toolbar={null} variant={'outlined'}>
          <TranscodingOptionsInput
            fullWidth
            label={translate('message.downloadOriginalFormat')}
          />
        </SimpleForm>
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} color="secondary">
          {translate('ra.action.close')}
        </Button>
        <Button onClick={handleDownload} color="primary">
          {translate('ra.action.download')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export default DownloadMenuDialog
