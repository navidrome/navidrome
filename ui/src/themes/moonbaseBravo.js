import stylesheet from './moonbaseBravo.css.js'

export default {
  themeName: 'Moonbase - Bravo',
  palette: {
    primary: {
      main: '#d4a039',
    },
    secondary: {
      main: '#1e1e1c',
      contrastText: '#e5ddd3',
    },
    type: 'dark',
    background: {
      default: '#0a0a09',
    },
  },
  overrides: {
    MuiPaper: {
      root: {
        color: '#e5ddd3',
        backgroundColor: '#141413',
      },
    },
    MuiButton: {
      textPrimary: {
        color: '#d4a039',
      },
      textSecondary: {
        color: '#e5ddd3',
      },
    },
    MuiChip: {
      clickable: {
        background: '#1e1e1c',
      },
    },
    MuiFormGroup: {
      root: {
        color: '#e5ddd3',
      },
    },
    MuiFormHelperText: {
      error: {
        color: '#c45c3c',
      },
    },
    MuiTableHead: {
      root: {
        color: '#8a8278',
        background: '#0a0a09 !important',
      },
    },
    MuiTableCell: {
      root: {
        color: '#e5ddd3',
        background: '#141413 !important',
      },
      head: {
        color: '#8a8278',
        background: '#0a0a09 !important',
      },
    },
    NDLogin: {
      systemNameLink: {
        color: '#d4a039',
      },
      welcome: {
        color: '#e5ddd3',
      },
      card: {
        minWidth: 300,
        background: '#1e1e1c',
      },
      button: {
        boxShadow: '3px 3px 5px #0a0a09',
      },
    },
    NDMobileArtistDetails: {
      bgContainer: {
        background:
          'linear-gradient(to bottom, rgba(10, 10, 9, 0.72), #141413)!important',
      },
    },
  },
  player: {
    theme: 'dark',
    stylesheet,
  },
}
