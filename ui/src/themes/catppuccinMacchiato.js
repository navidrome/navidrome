import stylesheet from './catppuccinMacchiato.css.js'

export default {
  themeName: 'Catppuccin Macchiato',
  palette: {
    primary: {
      main: '#c6a0f6', // Mauve
    },
    secondary: {
      main: '#1e2030', // Mantle
      contrastText: '#cad3f5', // Text
    },
    type: 'dark',
    background: {
      default: '#24273a', // Base
    },
  },
  overrides: {
    MuiPaper: {
      root: {
        color: '#cad3f5', // Text
        backgroundColor: '#1e2030', // Mantle
        MuiSnackbarContent: {
          root: {
            color: '#cad3f5', // Text
            backgroundColor: '#ed8796', // Red
          },
          message: {
            color: '#cad3f5', // Text
            backgroundColor: '#ed8796', // Red
          },
        },
      },
    },
    MuiButton: {
      textPrimary: {
        color: '#8aadf4', // Blue
      },
      textSecondary: {
        color: '#cad3f5', // Text
      },
    },
    MuiChip: {
      clickable: {
        background: '#1e2030', // Mantle
      },
    },
    MuiFormGroup: {
      root: {
        color: '#cad3f5', // Text
      },
    },
    MuiFormHelperText: {
      root: {
        Mui: {
          error: {
            color: '#ed8796', // Red
          },
        },
      },
    },
    MuiTableHead: {
      root: {
        color: '#cad3f5', // Text
        background: '#1e2030', // Mantle
      },
    },
    MuiTableCell: {
      root: {
        color: '#cad3f5', // Text
        background: '#1e2030 !important', // Mantle
      },
      head: {
        color: '#cad3f5', // Text
        background: '#1e2030 !important', // Mantle
      },
    },
    NDLogin: {
      systemNameLink: {
        color: '#c6a0f6', // Mauve
      },
      icon: {},
      welcome: {
        color: '#cad3f5', // Text
      },
      card: {
        minWidth: 300,
        background: '#24273a', // Base
      },
      avatar: {},
      button: {
        boxShadow: '3px 3px 5px #24273a', // Base
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
