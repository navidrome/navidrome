import React, { useMemo } from 'react'
import PropTypes from 'prop-types'
import Chip from '@material-ui/core/Chip'
import config from '../config'
import { makeStyles } from '@material-ui/core'
import clsx from 'clsx'
import { calculateGain } from '../utils/calculateReplayGain'

const llFormats = new Set(config.losslessFormats.split(','))
const placeholder = 'N/A'

// Lossless streams have no meaningful bitrate to show, so their quality is
// described by the PCM characteristics instead: `FLAC 96/24`.
const formatSampleRate = (sampleRate) => {
  const khz = sampleRate / 1000
  return Number.isInteger(khz) ? String(khz) : khz.toFixed(1)
}

const losslessDetail = (sampleRate, bitDepth) => {
  // DSD carries a 1-bit stream at MHz rates, where `2822.4/1` would say
  // nothing useful. Those formats keep showing the container alone until
  // modulation gets its own representation.
  if (bitDepth === 1 || !(sampleRate > 0)) {
    return ''
  }
  const rate = formatSampleRate(sampleRate)
  return bitDepth > 0 ? `${rate}/${bitDepth}` : rate
}

const useStyle = makeStyles(
  (theme) => ({
    chip: {
      transform: 'scale(0.8)',
    },
  }),
  {
    name: 'NDQualityInfo',
  },
)

export const QualityInfo = ({
  record,
  size,
  gainMode,
  preAmp,
  className,
  transcodeStream,
  isDirectPlay,
}) => {
  const classes = useStyle()
  let {
    suffix,
    bitRate,
    sampleRate,
    bitDepth,
    rgAlbumGain,
    rgAlbumPeak,
    rgTrackGain,
    rgTrackPeak,
  } = record
  let info = placeholder

  if (suffix) {
    suffix = suffix.toUpperCase()
    info = suffix
    if (llFormats.has(suffix)) {
      const detail = losslessDetail(sampleRate, bitDepth)
      if (detail) {
        info += ' ' + detail
      }
    } else if (bitRate > 0) {
      info += ' ' + bitRate
    }
  }

  // Show transcode target when transcoding (not direct play)
  if (transcodeStream && !isDirectPlay) {
    const targetCodec = (transcodeStream.codec || '').toUpperCase()
    const targetBitrate = transcodeStream.audioBitrate
      ? Math.round(transcodeStream.audioBitrate / 1000)
      : 0
    let targetInfo = targetCodec
    if (targetBitrate > 0) {
      targetInfo += ' ' + targetBitrate
    }
    const sourceSuffix = suffix || placeholder
    info = `${sourceSuffix} → ${targetInfo}`
  }

  const extra = useMemo(() => {
    if (gainMode !== 'none') {
      const gainValue = calculateGain(
        { gainMode, preAmp },
        { rgAlbumGain, rgAlbumPeak, rgTrackGain, rgTrackPeak },
      )
      // convert normalized gain (after peak) back to dB
      const toDb = (Math.log10(gainValue) * 20).toFixed(2)
      return ` (${toDb} dB)`
    }

    return ''
  }, [gainMode, preAmp, rgAlbumGain, rgAlbumPeak, rgTrackGain, rgTrackPeak])

  return (
    <Chip
      className={clsx(classes.chip, className)}
      variant="outlined"
      size={size}
      label={`${info}${extra}`}
    />
  )
}

QualityInfo.propTypes = {
  record: PropTypes.object.isRequired,
  size: PropTypes.string,
  className: PropTypes.string,
  gainMode: PropTypes.string,
  transcodeStream: PropTypes.object,
  isDirectPlay: PropTypes.bool,
}

QualityInfo.defaultProps = {
  record: {},
  size: 'small',
  gainMode: 'none',
}
