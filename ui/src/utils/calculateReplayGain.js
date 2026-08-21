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
      // Fall back to track gain when the album gain is missing (singles, or
      // tracks without album ReplayGain tags), matching common ReplayGain
      // players instead of applying no adjustment at all.
      if (song.rgAlbumGain === undefined || song.rgAlbumPeak === undefined) {
        return calculateReplayGain(
          gainInfo.preAmp,
          song.rgTrackGain,
          song.rgTrackPeak,
        )
      }
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
