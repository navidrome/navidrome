import { useRecordContext } from 'react-admin'
import { Avatar } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import clsx from 'clsx'
import config from '../config'
import subsonic from '../subsonic'
import { useImageUrl } from './useImageUrl'
import { BlurHashCanvas } from './BlurHashCanvas'

const useStyles = makeStyles({
  root: {
    position: 'relative',
    display: 'inline-flex',
    width: '55px',
    height: '55px',
  },
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
  circular: {
    borderRadius: '50%',
  },
  blur: {
    position: 'absolute',
    top: 0,
    left: 0,
    width: '100%',
    height: '100%',
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
    ? subsonic.getCoverArtUrl(record, config.uiCoverArtSize, square)
    : null
  const { imgUrl, loading } = useImageUrl(url)
  if (!record) return null

  const avatar = (
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

  // Show the blurhash behind the transparent avatar until the real image loads.
  if (!(loading && record.blurHash)) return avatar
  return (
    <div className={classes.root}>
      <BlurHashCanvas
        hash={record.blurHash}
        className={clsx(classes.blur, square ? classes.square : classes.circular)}
      />
      {avatar}
    </div>
  )
}

CoverArtAvatar.defaultProps = { label: '', sortable: false }
