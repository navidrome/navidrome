import { useEffect, useRef, useState } from 'react'
import PropTypes from 'prop-types'
import clsx from 'clsx'
import { makeStyles } from '@material-ui/core/styles'
import config from '../config'
import subsonic from '../subsonic'
import { useImageUrl } from './useImageUrl'
import { BlurHashCanvas } from './BlurHashCanvas'

// Drives both the CSS transition and the timer that retires the blurhash, so they cannot drift.
const fadeMs = 500

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
  img: {
    opacity: 0,
    transition: `opacity ${fadeMs}ms ease-out`,
    '@media (prefers-reduced-motion: reduce)': { transition: 'none' },
  },
  imgVisible: { opacity: 1 },
  // Already-decoded blobs appear at once: fading them in would re-animate on every remount.
  imgInstant: { opacity: 1, transition: 'none' },
})

// Artwork renders an entity's cover through the shared useImageUrl blob cache, so it survives
// React remounts without re-fetching. The blurhash is the loading placeholder; the image is only
// mounted once its blob is ready, so an unresolved cover never renders as a broken <img>.
export const Artwork = ({
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
  const { imgUrl } = useImageUrl(url)

  // A blob already cached when this instance mounted paints on the first frame, so it skips the
  // fade; anything fetched later cross-fades over the blurhash.
  const cachedOnMount = useRef(null)
  if (cachedOnMount.current === null) {
    cachedOnMount.current = !!imgUrl
  }
  const [decoded, setDecoded] = useState(false)
  const [faded, setFaded] = useState(false)
  useEffect(() => {
    setDecoded(false)
    setFaded(false)
  }, [url])

  // Retire the blurhash on a timer rather than transitionend: under prefers-reduced-motion the
  // transition is none, so the event never fires and the placeholder would stay up forever.
  useEffect(() => {
    if (!decoded || faded) return undefined
    const timer = setTimeout(() => setFaded(true), fadeMs)
    return () => clearTimeout(timer)
  }, [decoded, faded])

  if (!record) return null

  const instant = cachedOnMount.current
  // The blurhash stays mounted under the image until the fade ends. Swapping them the moment the
  // blob arrives would expose the empty container for the length of the fade.
  const showBlurHash = !!record.blurHash && !instant && !faded
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
          className={clsx(
            classes.fill,
            classes.img,
            instant && classes.imgInstant,
            decoded && classes.imgVisible,
          )}
          style={{ objectFit: fit }}
          // Fading on decode, not on mount, keeps the image from ramping up before it can paint.
          onLoad={() => setDecoded(true)}
        />
      )}
    </div>
  )
}

Artwork.propTypes = {
  record: PropTypes.object,
  size: PropTypes.number,
  square: PropTypes.bool,
  fit: PropTypes.oneOf(['cover', 'contain']),
  className: PropTypes.string,
  title: PropTypes.string,
  onClick: PropTypes.func,
}
