import stylesheet from './rosePine.css.js'

export default {
  themeName: 'Rosé Pine',
  palette: {
    primary: {
      main: '#ebbcba',
    },
    secondary: {
      main: '#1f1d2e',
      contrastText: '#e0def4',
    },
    type: 'dark',
    background: {
      default: '#191724',
      paper: '#1f1d2e',
    },
  },
  overrides: {
    MuiPaper: {
      root: {
        color: '#e0def4',
        backgroundColor: '#1f1d2e',
      },
    },
    MuiButton: {
      textPrimary: {
        color: '#31748f',
      },
      textSecondary: {
        color: '#e0def4',
      },
    },
    MuiIconButton: {
      colorSecondary: {
        color: '#6e6a86',
      },
    },
    MuiChip: {
      clickable: {
        background: '#26233a',
      },
    },
    MuiCheckbox: {
      colorSecondary: {
        color: '#6e6a86',
        '&$checked': {
          color: '#ebbcba',
        },
      },
    },
    MuiFormGroup: {
      root: {
        color: '#e0def4',
      },
    },
    MuiFormHelperText: {
      root: {
        '&$error': {
          color: '#eb6f92',
        },
      },
    },
    MuiTableHead: {
      root: {
        color: '#e0def4',
        background: '#1f1d2e',
      },
    },
    MuiTableCell: {
      root: {
        color: '#e0def4',
        background: '#1f1d2e !important',
      },
      head: {
        color: '#e0def4',
        background: '#1f1d2e !important',
      },
    },
    NDLogin: {
      systemNameLink: {
        color: '#ebbcba',
      },
      icon: {},
      welcome: {
        color: '#e0def4',
      },
      card: {
        minWidth: 300,
        background: '#191724',
      },
      avatar: {},
      button: {
        boxShadow: '3px 3px 5px rgba(25, 23, 36, 0.35)',
      },
    },
    NDMobileArtistDetails: {
      bgContainer: {
        background:
          'linear-gradient(to bottom, rgba(25, 23, 36, 0.72), rgb(25, 23, 36))!important',
      },
    },
  },
  player: {
    theme: 'dark',
    stylesheet,
  },
}
