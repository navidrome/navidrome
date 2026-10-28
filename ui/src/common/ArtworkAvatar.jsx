import { useRecordContext } from 'react-admin'
import { useTheme } from '@mui/material/styles'
import clsx from 'clsx'
import config from '../config'
import { Artwork } from './Artwork'

// TODO(v6): migrate makeStyles → sx prop / styled()
// const useStyles = makeStyles({
  avatar: {
    width: '55px',
    height: '55px',
  },
  square: {
    borderRadius: '4px',
  },
  circular: {
    borderRadius: '50%',
  },
})

export const ArtworkAvatar = ({ record: recordProp, variant = 'circular' }) => {
  const classes = useStyles()
  const recordContext = useRecordContext()
  const record = recordProp || recordContext
  if (!record) return null
  const square = variant !== 'circular'
  return (
    <Artwork
      record={record}
      size={config.uiCoverArtSize}
      square={square}
      className={clsx(
        classes.avatar,
        square ? classes.square : classes.circular,
      )}
      title={record.name}
    />
  )
}

ArtworkAvatar.defaultProps = { label: '', sortable: false }
