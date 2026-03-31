import { useRecordContext } from 'react-admin'
import { Avatar } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import clsx from 'clsx'
import { COVER_ART_SIZE } from '../consts'
import subsonic from '../subsonic'
import { useImageUrl } from './useImageUrl'

const useStyles = makeStyles({
  avatar: {
    width: '55px',
    height: '55px',
  },
  avatarEmpty: {
    backgroundColor: 'transparent',
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
  const square = variant !== 'circular'
  const url = record
    ? subsonic.getCoverArtUrl(record, COVER_ART_SIZE, square)
    : null
  const { imgUrl } = useImageUrl(url)
  if (!record) return null
  return (
    <Avatar
      src={imgUrl || undefined}
      variant={variant}
      className={clsx(
        classes.avatar,
        square && classes.square,
        !imgUrl && classes.avatarEmpty,
      )}
      alt={record.name}
    >
      {/* Empty child prevents default person icon while loading */}
      {!imgUrl && <span />}
    </Avatar>
  )
}

CoverArtAvatar.defaultProps = { label: '', sortable: false }
