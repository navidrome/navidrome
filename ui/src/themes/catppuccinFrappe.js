import stylesheet from './catppuccinFrappe.css.js'

export default {
  themeName: 'Catppuccin Frappé',
  palette: {
    primary: {
      main: '#ca9ee6', // Mauve
    },
    secondary: {
      main: '#292c3c', //Mantle
      contrastText: '#c6d0f5', // Text
    },
    type: 'dark',
    background: {
      default: '#303446', // Base
    },
  },
  overrides: {
    MuiPaper: {
      root: {
        color: '#c6d0f5', // Text
        backgroundColor: '#292c3c', // Mantle
        MuiSnackbarContent: {
          root: {
            color: '#c6d0f5', // Text
            backgroundColor: '#e78284', // Red
          },
          message: {
            color: '#c6d0f5', // Text
            backgroundColor: '#e78284', // Red
          },
        },
      },
    },
    MuiButton: {
      textPrimary: {
        color: '#8caaee', // Blue
      },
      textSecondary: {
        color: '#c6d0f5', // Text
      },
    },
    MuiChip: {
      clickable: {
        background: '#292c3c', //Mantle
      },
    },
    MuiFormGroup: {
      root: {
        color: '#c6d0f5', // Text
      },
    },
    MuiFormHelperText: {
      root: {
        Mui: {
          error: {
            color: '#e78284', // Red
          },
        },
      },
    },
    MuiTableHead: {
      root: {
        color: '#c6d0f5', // Text
        background: '#292c3c', //Mantle
      },
    },
    MuiTableCell: {
      root: {
        color: '#c6d0f5', // Text
        background: '#292c3c !important', //Mantle
      },
      head: {
        color: '#c6d0f5', // Text
        background: '#292c3c !important', //Mantle
      },
    },
    NDLogin: {
      systemNameLink: {
        color: '#ca9ee6', // Mauve
      },
      icon: {},
      welcome: {
        color: '#c6d0f5', // Text
      },
      card: {
        minWidth: 300,
        background: '#303446', // Base
      },
      avatar: {},
      button: {
        boxShadow: '3px 3px 5px #303446', // Base
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
