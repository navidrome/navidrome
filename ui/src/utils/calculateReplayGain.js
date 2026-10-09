const calculateReplayGain = (preAmp, gain, peak) => {
  if (gain == null) {
    return 1
  }

  // https://wiki.hydrogenaud.io/index.php?title=ReplayGain_1.0_specification&section=19
  // Normalized to max gain
  const replayGain = 10 ** ((gain + preAmp) / 20)
  return peak == null ? replayGain : Math.min(replayGain, 1 / peak)
}

export const calculateGain = (gainInfo, song) => {
  switch (gainInfo.gainMode) {
    case 'album': {
      // Singles and untagged albums have no album gain, so use the track gain.
      if (song.rgAlbumGain == null) {
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
