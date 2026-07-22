import stylesheet from './catppuccinFrappe.css.js'

export default {
  themeName: 'Catppuccin Frappé',
  palette: {
    primary: {
      main: '#ca9ee6',
    },
    secondary: {
      main: '#292c3c',
      contrastText: '#c6d0f5',
    },
    type: 'dark',
    background: {
      default: '#303446',
    },
  },
  overrides: {
    MuiPaper: {
      root: {
        color: '#c6d0f5',
        backgroundColor: '#292c3c',
        MuiSnackbarContent: {
          root: {
            color: '#c6d0f5',
            backgroundColor: '#e78284',
          },
          message: {
            color: '#c6d0f5',
            backgroundColor: '#e78284',
          },
        },
      },
    },
    MuiButton: {
      textPrimary: {
        color: '#8caaee',
      },
      textSecondary: {
        color: '#c6d0f5',
      },
    },
    MuiChip: {
      clickable: {
        background: '#292c3c',
      },
    },
    MuiFormGroup: {
      root: {
        color: '#c6d0f5',
      },
    },
    MuiFormHelperText: {
      root: {
        Mui: {
          error: {
            color: '#e78284',
          },
        },
      },
    },
    MuiTableHead: {
      root: {
        color: '#c6d0f5',
        background: '#292c3c',
      },
    },
    MuiTableCell: {
      root: {
        color: '#c6d0f5',
        background: '#292c3c !important',
      },
      head: {
        color: '#c6d0f5',
        background: '#292c3c !important',
      },
    },
    NDLogin: {
      systemNameLink: {
        color: '#ca9ee6',
      },
      icon: {},
      welcome: {
        color: '#c6d0f5',
      },
      card: {
        minWidth: 300,
        background: '#303446',
      },
      avatar: {},
      button: {
        boxShadow: '3px 3px 5px #303446',
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
