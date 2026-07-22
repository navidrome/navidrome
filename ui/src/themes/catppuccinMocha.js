import stylesheet from './catppuccinMocha.css.js'

export default {
  themeName: 'Catppuccin Mocha',
  palette: {
    primary: {
      main: '#cba6f7', // Mauve
    },
    secondary: {
      main: '#181825', // Mantle
      contrastText: '#cdd6f4', // Text
    },
    type: 'dark',
    background: {
      default: '#1e1e2e', // Base
    },
  },
  overrides: {
    MuiPaper: {
      root: {
        color: '#cdd6f4', // Text
        backgroundColor: '#181825', // Mantle
        MuiSnackbarContent: {
          root: {
            color: '#cdd6f4', // Text
            backgroundColor: '#f38ba8', // Red
          },
          message: {
            color: '#cdd6f4', // Text
            backgroundColor: '#f38ba8', // Red
          },
        },
      },
    },
    MuiButton: {
      textPrimary: {
        color: '#89b4fa', // Blue
      },
      textSecondary: {
        color: '#cdd6f4', // Text
      },
    },
    MuiChip: {
      clickable: {
        background: '#181825', // Mantle
      },
    },
    MuiFormGroup: {
      root: {
        color: '#cdd6f4', // Text
      },
    },
    MuiFormHelperText: {
      root: {
        Mui: {
          error: {
            color: '#f38ba8', // Red
          },
        },
      },
    },
    MuiTableHead: {
      root: {
        color: '#cdd6f4', // Text
        background: '#181825', // Mantle
      },
    },
    MuiTableCell: {
      root: {
        color: '#cdd6f4', // Text
        background: '#181825 !important', // Mantle
      },
      head: {
        color: '#cdd6f4', // Text
        background: '#181825 !important', // Mantle
      },
    },
    NDLogin: {
      systemNameLink: {
        color: '#cba6f7', // Mauve
      },
      icon: {},
      welcome: {
        color: '#cdd6f4', // Text
      },
      card: {
        minWidth: 300,
        background: '#1e1e2e', // Base
      },
      avatar: {},
      button: {
        boxShadow: '3px 3px 5px #1e1e2e', // Base
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
