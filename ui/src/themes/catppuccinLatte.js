import stylesheet from './catppuccinLatte.css.js'

export default {
  themeName: 'Catppuccin Latte',
  palette: {
    primary: { main: '#8839ef' }, // mauve
    secondary: {
      main: '#ccd0da', // surface0
      contrastText: '#4c4f69', // text
    },
    type: 'light',
    background: {
      default: '#eff1f5', // base
    },
  },

  overrides: {
    MuiPaper: {
      root: {
        color: '#4c4f69', // text
        backgroundColor: '#e6e9ef', // mantle
      },
    },

    MuiButton: {
      textPrimary: {
        color: '#1e66f5', // blue
      },
      textSecondary: {
        color: '#4c4f69', // text
      },
    },

    MuiChip: {
      clickable: {
        background: '#ccd0da', // surface0
      },
    },

    MuiFormGroup: {
      root: {
        color: '#4c4f69',
      },
    },

    MuiFormHelperText: {
      root: {
        Mui: {
          error: {
            color: '#d20f39', // red
          },
        },
      },
    },

    MuiTableHead: {
      root: {
        color: '#4c4f69',
        background: '#e6e9ef',
      },
    },

    MuiTableCell: {
      root: {
        color: '#4c4f69',
        background: '#e6e9ef !important',
      },
      head: {
        color: '#4c4f69',
        background: '#e6e9ef !important',
      },
    },

    NDLogin: {
      systemNameLink: {
        color: '#8839ef', // mauve
      },
      icon: {},
      welcome: {
        color: '#4c4f69',
      },
      card: {
        minWidth: 300,
        background: '#eff1f5',
      },
      avatar: {},
      button: {
        boxShadow: '3px 3px 5px #ccd0da',
      },
    },

    NDMobileArtistDetails: {
      bgContainer: {
        background:
          'linear-gradient(to bottom, rgba(255 255 255 / 72%), rgb(239 241 245))!important',
      },
    },
  },

  player: {
    theme: 'light',
    stylesheet,
  },
}
