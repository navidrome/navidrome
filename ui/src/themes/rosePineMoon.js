import stylesheet from './rosePineMoon.css.js'

export default {
  themeName: 'Rosé Pine Moon',
  palette: {
    primary: {
      main: '#ea9a97',
    },
    secondary: {
      main: '#2a273f',
      contrastText: '#e0def4',
    },
    type: 'dark',
    background: {
      default: '#232136',
      paper: '#2a273f',
    },
  },
  overrides: {
    MuiPaper: {
      root: {
        color: '#e0def4',
        backgroundColor: '#2a273f',
      },
    },
    MuiButton: {
      textPrimary: {
        color: '#3e8fb0',
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
        background: '#393552',
      },
    },
    MuiCheckbox: {
      colorSecondary: {
        color: '#6e6a86',
        '&$checked': {
          color: '#ea9a97',
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
        background: '#2a273f',
      },
    },
    MuiTableCell: {
      root: {
        color: '#e0def4',
        background: '#2a273f !important',
      },
      head: {
        color: '#e0def4',
        background: '#2a273f !important',
      },
    },
    NDLogin: {
      systemNameLink: {
        color: '#ea9a97',
      },
      icon: {},
      welcome: {
        color: '#e0def4',
      },
      card: {
        minWidth: 300,
        background: '#232136',
      },
      avatar: {},
      button: {
        boxShadow: '3px 3px 5px rgba(35, 33, 54, 0.35)',
      },
    },
    NDMobileArtistDetails: {
      bgContainer: {
        background:
          'linear-gradient(to bottom, rgba(35, 33, 54, 0.72), rgb(35, 33, 54))!important',
      },
    },
  },
  player: {
    theme: 'dark',
    stylesheet,
  },
}
