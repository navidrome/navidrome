import { useRecordContext } from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'
import clsx from 'clsx'
import config from '../config'
import { CoverImage } from './CoverImage'

const useStyles = makeStyles({
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

export const CoverArtAvatar = ({ record: recordProp, variant = 'circular' }) => {
  const classes = useStyles()
  const recordContext = useRecordContext()
  const record = recordProp || recordContext
  if (!record) return null
  const square = variant !== 'circular'
  return (
    <CoverImage
      record={record}
      size={config.uiCoverArtSize}
      square={square}
      className={clsx(classes.avatar, square ? classes.square : classes.circular)}
      title={record.name}
    />
  )
}

CoverArtAvatar.defaultProps = { label: '', sortable: false }
