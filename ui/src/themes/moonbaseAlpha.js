import stylesheet from './moonbaseAlpha.css.js'

export default {
  themeName: 'Moonbase - Alpha',
  palette: {
    primary: {
      main: '#9a7420',
    },
    secondary: {
      main: '#ede8df',
      contrastText: '#1a1917',
    },
    type: 'light',
    background: {
      default: '#f5f0e8',
    },
  },
  overrides: {
    MuiPaper: {
      root: {
        color: '#1a1917',
        backgroundColor: '#faf8f4',
      },
    },
    MuiButton: {
      textPrimary: {
        color: '#9a7420',
      },
      textSecondary: {
        color: '#1a1917',
      },
    },
    MuiChip: {
      clickable: {
        background: '#ede8df',
      },
    },
    MuiFormGroup: {
      root: {
        color: '#1a1917',
      },
    },
    MuiFormHelperText: {
      error: {
        color: '#b04a2e',
      },
    },
    MuiTableHead: {
      root: {
        color: '#6b635a',
        background: '#f5f0e8 !important',
      },
    },
    MuiTableCell: {
      root: {
        color: '#1a1917',
        background: '#faf8f4 !important',
      },
      head: {
        color: '#6b635a',
        background: '#f5f0e8 !important',
      },
    },
    NDLogin: {
      systemNameLink: {
        color: '#9a7420',
      },
      welcome: {
        color: '#1a1917',
      },
      card: {
        minWidth: 300,
        background: '#faf8f4',
      },
      button: {
        boxShadow: '3px 3px 5px rgba(0, 0, 0, 0.12)',
      },
    },
    NDMobileArtistDetails: {
      bgContainer: {
        background:
          'linear-gradient(to bottom, rgba(245, 240, 232, 0.72), #faf8f4)!important',
      },
    },
  },
  player: {
    theme: 'light',
    stylesheet,
  },
}
