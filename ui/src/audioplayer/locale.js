const locale = (translate) => ({
  playListsText: translate('player.playListsText'),
  openText: translate('player.openText'),
  closeText: translate('player.closeText'),
  notContentText: translate('player.notContentText'),
  clickToPlayText: translate('player.clickToPlayText'),
  clickToPauseText: translate('player.clickToPauseText'),
  nextTrackText: translate('player.nextTrackText'),
  previousTrackText: translate('player.previousTrackText'),
  reloadText: translate('player.reloadText'),
  volumeText: translate('player.volumeText'),
  toggleLyricText: translate('player.toggleLyricText'),
  toggleMiniModeText: translate('player.toggleMiniModeText'),
  destroyText: translate('player.destroyText'),
  downloadText: translate('player.downloadText'),
  removeAudioListsText: translate('player.removeAudioListsText'),
  clickToDeleteText: (name) => translate('player.clickToDeleteText', { name }),
  emptyLyricText: translate('player.emptyLyricText'),
  playModeText: {
    order: translate('player.playModeText.order'),
    orderLoop: translate('player.playModeText.orderLoop'),
    singleLoop: translate('player.playModeText.singleLoop'),
    shufflePlay: translate('player.playModeText.shufflePlay'),
  },
})

export default locale
