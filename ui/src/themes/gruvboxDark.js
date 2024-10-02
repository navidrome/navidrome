import stylesheet from './gruvboxDark.css.js'

export default {
  themeName: 'Gruvbox Dark',
  palette: {
    primary: {
      main: '#8ec07c',
    },
    secondary: {
      main: '#3c3836',
      contrastText: '#ebdbb2',
    },
    type: 'dark',
    background: {
      default: '#282828',
    },
  },
  overrides: {
    MuiPaper: {
      root: {
        color: '#ebdbb2',
        backgroundColor: '#3c3836',
        MuiSnackbarContent: {
          root: {
            color: '#ebdbb2',
            backgroundColor: '#cc241d',
          },
          message: {
            color: '#ebdbb2',
            backgroundColor: '#cc241d',
          },
        },
      },
    },
    MuiButton: {
      textPrimary: {
        color: '#458588',
      },
      textSecondary: {
        color: '#ebdbb2',
      },
    },
    MuiChip: {
      clickable: {
        background: '#49483e',
      },
    },
    MuiFormGroup: {
      root: {
        color: '#ebdbb2',
      },
    },
    MuiFormHelperText: {
      root: {
        Mui: {
          error: {
            color: '#cc241d',
          },
        },
      },
    },
    MuiTableHead: {
      root: {
        color: '#ebdbb2',
        background: '#3c3836 !important',
      },
    },
    MuiTableCell: {
      root: {
        color: '#ebdbb2',
        background: '#3c3836 !important',
      },
      head: {
        color: '#ebdbb2',
        background: '#3c3836 !important',
      },
    },
    NDLogin: {
      systemNameLink: {
        color: '#8ec07c',
      },
      icon: {},
      welcome: {
        color: '#ebdbb2',
      },
      card: {
        minWidth: 300,
        background: '#3c3836',
      },
      avatar: {},
      button: {
        boxShadow: '3px 3px 5px #3c3836',
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
