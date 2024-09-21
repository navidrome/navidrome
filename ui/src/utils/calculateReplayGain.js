const calculateReplayGain = (preAmp, gain, peak) => {
  if (gain === undefined || peak === undefined) {
    return 1
  }

  // https://wiki.hydrogenaud.io/index.php?title=ReplayGain_1.0_specification&section=19
  // Normalized to max gain
  return Math.min(10 ** ((gain + preAmp) / 20), 1 / peak)
}

export const calculateGain = (gainInfo, song) => {
  switch (gainInfo.gainMode) {
    case 'album': {
      return calculateReplayGain(
        gainInfo.preAmp,
        song.rgAlbumGain,
        song.rgAlbumPeak,
      )
    }
    case 'track': {
      return calculateReplayGain(
        gainInfo.preAmp,
        song.rgTrackGain,
        song.rgTrackPeak,
      )
    }
    default: {
      return 1
    }
  }
}
