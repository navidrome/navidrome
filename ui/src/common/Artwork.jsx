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
  // className supplies the size and shape; overflow:hidden clips the fills to it.
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

// Renders a cover through the shared useImageUrl blob cache, so it survives remounts without
// re-fetching. The image mounts only once its blob is ready, so it never renders broken.
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

  // A blob already cached at mount paints on the first frame, so it skips the fade.
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

  // Timer, not transitionend: under prefers-reduced-motion there is no transition to end.
  useEffect(() => {
    if (!decoded || faded) return undefined
    const timer = setTimeout(() => setFaded(true), fadeMs)
    return () => clearTimeout(timer)
  }, [decoded, faded])

  if (!record) return null

  const instant = cachedOnMount.current
  // Kept mounted until the fade ends; swapping on blob arrival would flash an empty container.
  const showBlurHash = !!record.blurHash && !instant && !faded
  // A square request is padded, not cropped, so `contain` keeps placeholder and image aligned.
  const effectiveFit = square ? 'contain' : fit
  const ratio = record.imageWidth / record.imageHeight
  const handleClick = imgUrl && onClick ? onClick : undefined
  return (
    <div
      className={clsx(classes.root, className)}
      onClick={handleClick}
      style={{ cursor: handleClick ? 'pointer' : 'default' }}
    >
      {showBlurHash && (
        <BlurHashCanvas
          hash={record.blurHash}
          ratio={ratio}
          fit={effectiveFit}
          className={classes.fill}
        />
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
          style={{ objectFit: effectiveFit }}
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
