export default {
  themeName: 'Catppuccin Macchiato',
  palette: {
    primary: {
      main: '#c6a0f6',
    },
    secondary: {
      main: '#1e2030',
      contrastText: '#cad3f5',
    },
    type: 'dark',
    background: {
      default: '#24273a',
    },
  },
  overrides: {
    MuiPaper: {
      root: {
        color: '#cad3f5',
        backgroundColor: '#1e2030',
        MuiSnackbarContent: {
          root: {
            color: '#cad3f5',
            backgroundColor: '#ed8796',
          },
          message: {
            color: '#cad3f5',
            backgroundColor: '#ed8796',
          },
        },
      },
    },
    MuiButton: {
      textPrimary: {
        color: '#8aadf4',
      },
      textSecondary: {
        color: '#cad3f5',
      },
    },
    MuiChip: {
      clickable: {
        background: '#1e2030',
      },
    },
    MuiFormGroup: {
      root: {
        color: '#cad3f5',
      },
    },
    MuiFormHelperText: {
      root: {
        Mui: {
          error: {
            color: '#ed8796',
          },
        },
      },
    },
    MuiTableHead: {
      root: {
        color: '#cad3f5',
        background: '#1e2030',
      },
    },
    MuiTableCell: {
      root: {
        color: '#cad3f5',
        background: '#1e2030 !important',
      },
      head: {
        color: '#cad3f5',
        background: '#1e2030 !important',
      },
    },
    NDLogin: {
      systemNameLink: {
        color: '#c6a0f6',
      },
      icon: {},
      welcome: {
        color: '#cad3f5',
      },
      card: {
        minWidth: 300,
        background: '#24273a',
      },
      avatar: {},
      button: {
        boxShadow: '3px 3px 5px #24273a',
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
    stylesheet: require('./catppuccinMacchiato.css.js'),
  },
}
