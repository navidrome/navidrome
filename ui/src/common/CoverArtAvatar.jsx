import { useRecordContext } from 'react-admin'
import { Avatar } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import clsx from 'clsx'
import subsonic from '../subsonic'

const useStyles = makeStyles({
  avatar: {
    width: '55px',
    height: '55px',
  },
  square: {
    borderRadius: '4px',
  },
})

export const CoverArtAvatar = ({
  record: recordProp,
  variant = 'circular',
}) => {
  const classes = useStyles()
  const recordContext = useRecordContext()
  const record = recordProp || recordContext
  if (!record) return null
  const square = variant !== 'circular'
  return (
    <Avatar
      src={subsonic.getCoverArtUrl(record, 80, square)}
      variant={variant}
      className={clsx(classes.avatar, square && classes.square)}
      alt={record.name}
    />
  )
}

CoverArtAvatar.defaultProps = { label: '', sortable: false }
