import stylesheet from './rosePineDawn.css.js'

export default {
  themeName: 'Rosé Pine Dawn',
  palette: {
    primary: {
      main: '#d7827e',
    },
    secondary: {
      main: '#fffaf3',
      contrastText: '#464261',
    },
    type: 'light',
    background: {
      default: '#faf4ed',
      paper: '#fffaf3',
    },
  },
  overrides: {
    MuiPaper: {
      root: {
        color: '#464261',
        backgroundColor: '#fffaf3',
      },
    },
    MuiButton: {
      textPrimary: {
        color: '#286983',
      },
      textSecondary: {
        color: '#464261',
      },
    },
    MuiIconButton: {
      colorSecondary: {
        color: '#9893a5',
      },
    },
    MuiChip: {
      clickable: {
        background: '#f2e9e1',
      },
    },
    MuiCheckbox: {
      colorSecondary: {
        color: '#9893a5',
        '&$checked': {
          color: '#d7827e',
        },
      },
    },
    MuiFormGroup: {
      root: {
        color: '#464261',
      },
    },
    MuiFormHelperText: {
      root: {
        '&$error': {
          color: '#b4637a',
        },
      },
    },
    MuiTableHead: {
      root: {
        color: '#464261',
        background: '#fffaf3',
      },
    },
    MuiTableCell: {
      root: {
        color: '#464261',
        background: '#fffaf3 !important',
      },
      head: {
        color: '#464261',
        background: '#fffaf3 !important',
      },
    },
    NDLogin: {
      systemNameLink: {
        color: '#d7827e',
      },
      icon: {},
      welcome: {
        color: '#464261',
      },
      card: {
        minWidth: 300,
        background: '#faf4ed',
      },
      avatar: {},
      button: {
        boxShadow: '3px 3px 5px rgba(87, 82, 121, 0.12)',
      },
    },
    NDMobileArtistDetails: {
      bgContainer: {
        background:
          'linear-gradient(to bottom, rgba(250, 244, 237, 0.72), rgb(250, 244, 237))!important',
      },
    },
  },
  player: {
    theme: 'light',
    stylesheet,
  },
}
