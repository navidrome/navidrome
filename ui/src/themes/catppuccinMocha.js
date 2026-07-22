import stylesheet from './catppuccinMocha.css.js'

export default {
  themeName: 'Catppuccin Mocha',
  palette: {
    primary: {
      main: '#cba6f7',
    },
    secondary: {
      main: '#181825',
      contrastText: '#cdd6f4',
    },
    type: 'dark',
    background: {
      default: '#1e1e2e',
    },
  },
  overrides: {
    MuiPaper: {
      root: {
        color: '#cdd6f4',
        backgroundColor: '#181825',
        MuiSnackbarContent: {
          root: {
            color: '#cdd6f4',
            backgroundColor: '#f38ba8',
          },
          message: {
            color: '#cdd6f4',
            backgroundColor: '#f38ba8',
          },
        },
      },
    },
    MuiButton: {
      textPrimary: {
        color: '#89b4fa',
      },
      textSecondary: {
        color: '#cdd6f4',
      },
    },
    MuiChip: {
      clickable: {
        background: '#181825',
      },
    },
    MuiFormGroup: {
      root: {
        color: '#cdd6f4',
      },
    },
    MuiFormHelperText: {
      root: {
        Mui: {
          error: {
            color: '#f38ba8',
          },
        },
      },
    },
    MuiTableHead: {
      root: {
        color: '#cdd6f4',
        background: '#181825',
      },
    },
    MuiTableCell: {
      root: {
        color: '#cdd6f4',
        background: '#181825 !important',
      },
      head: {
        color: '#cdd6f4',
        background: '#181825 !important',
      },
    },
    NDLogin: {
      systemNameLink: {
        color: '#cba6f7',
      },
      icon: {},
      welcome: {
        color: '#cdd6f4',
      },
      card: {
        minWidth: 300,
        background: '#1e1e2e',
      },
      avatar: {},
      button: {
        boxShadow: '3px 3px 5px #1e1e2e',
      },
    },
    NDMobileArtistDetails: {
      bgContainer: {
        background:
          'linear-gradient(to bottom, rgba(52 52 52 / 72%), rgb(48 48 48))!important',
      },
    },
  },
  player: {
    theme: 'dark',
    stylesheet,
  },
}
