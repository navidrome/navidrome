import PropTypes from 'prop-types'
import clsx from 'clsx'
import { makeStyles } from '@material-ui/core/styles'
import config from '../config'
import subsonic from '../subsonic'
import { useImageUrl } from './useImageUrl'
import { BlurHashCanvas } from './BlurHashCanvas'

const useStyles = makeStyles({
  root: { position: 'relative', display: 'inline-flex' },
  blur: { position: 'absolute', top: 0, left: 0, zIndex: 0 },
  img: { position: 'relative', zIndex: 1 },
})

// CoverImage renders an entity's cover through the shared useImageUrl blob cache, so it survives
// React remounts without re-fetching, with the blurhash as the loading placeholder. `className`
// supplies the size (and any transition); the fade opacity is applied here.
export const CoverImage = ({
  record,
  size = config.uiCoverArtSize,
  square = false,
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
    <div className={classes.root}>
      {showBlurHash && (
        <BlurHashCanvas
          hash={record.blurHash}
          className={clsx(className, classes.blur)}
        />
      )}
      <img
        src={imgUrl || undefined}
        alt={title}
        title={title}
        onClick={handleClick}
        className={clsx(className, showBlurHash && classes.img)}
        style={{
          objectFit: 'cover',
          opacity: loading ? 0.5 : 1,
          cursor: handleClick ? 'pointer' : 'default',
        }}
      />
    </div>
  )
}

CoverImage.propTypes = {
  record: PropTypes.object,
  size: PropTypes.number,
  square: PropTypes.bool,
  className: PropTypes.string,
  title: PropTypes.string,
  onClick: PropTypes.func,
}
