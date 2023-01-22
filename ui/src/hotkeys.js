import config from './config'
const keyMap = {
  SHOW_HELP: { name: 'show_help', sequence: 'shift+?', group: 'Global' },
  TOGGLE_MENU: { name: 'toggle_menu', sequence: 'm', group: 'Global' },
  TOGGLE_PLAY: { name: 'toggle_play', sequence: 'space', group: 'Player' },
  PREV_SONG: { name: 'prev_song', sequence: 'left', group: 'Player' },
  NEXT_SONG: { name: 'next_song', sequence: 'right', group: 'Player' },
  CURRENT_SONG: { name: 'current_song', sequence: 'shift+c', group: 'Player' },
  VOL_UP: { name: 'vol_up', sequence: '=', group: 'Player' },
  VOL_DOWN: { name: 'vol_down', sequence: '-', group: 'Player' },
  ...(config.enableFavourites && {
    TOGGLE_LOVE: { name: 'toggle_love', sequence: 'l', group: 'Player' },
  }),
}

export { keyMap }
