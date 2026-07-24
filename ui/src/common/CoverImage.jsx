import PropTypes from 'prop-types'
import clsx from 'clsx'
import { makeStyles } from '@material-ui/core/styles'
import config from '../config'
import subsonic from '../subsonic'
import { useImageUrl } from './useImageUrl'
import { BlurHashCanvas } from './BlurHashCanvas'

const useStyles = makeStyles({
  // className supplies the size and shape; overflow:hidden clips the fills to a rounded shape.
  root: {
    position: 'relative',
    display: 'inline-flex',
    overflow: 'hidden',
  },
  fill: {
    position: 'absolute',
    top: 0,
    left: 0,
    width: '100%',
    height: '100%',
  },
  '@keyframes fadeIn': { from: { opacity: 0 }, to: { opacity: 1 } },
  img: { animation: '$fadeIn 0.3s ease-in-out' },
})

// CoverImage renders an entity's cover through the shared useImageUrl blob cache, so it survives
// React remounts without re-fetching. The blurhash is the loading placeholder; the image is only
// mounted once its blob is ready, so an unresolved cover never renders as a broken <img>.
export const CoverImage = ({
  record,
  size = config.uiCoverArtSize,
  square = false,
  fit = 'cover',
  className,
  title,
  onClick,
}) => {
  const classes = useStyles()
  const url = record ? subsonic.getCoverArtUrl(record, size, square) : ''
  const { imgUrl, loading } = useImageUrl(url)
  if (!record) return null

  const showBlurHash = loading && record.blurHash
  const handleClick = imgUrl && onClick ? onClick : undefined
  return (
    <div
      className={clsx(classes.root, className)}
      onClick={handleClick}
      style={{ cursor: handleClick ? 'pointer' : 'default' }}
    >
      {showBlurHash && (
        <BlurHashCanvas hash={record.blurHash} className={classes.fill} />
      )}
      {imgUrl && (
        <img
          src={imgUrl}
          alt={title}
          title={title}
          className={clsx(classes.fill, classes.img)}
          style={{ objectFit: fit }}
        />
      )}
    </div>
  )
}

CoverImage.propTypes = {
  record: PropTypes.object,
  size: PropTypes.number,
  square: PropTypes.bool,
  fit: PropTypes.oneOf(['cover', 'contain']),
  className: PropTypes.string,
  title: PropTypes.string,
  onClick: PropTypes.func,
}
