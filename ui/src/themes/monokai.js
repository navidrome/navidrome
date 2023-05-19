export default {
  themeName: 'Monokai',
  palette: {
    primary: {
      main: '#66d9ef',
    },
    secondary: {
      main: '#49483e',
      contrastText: '#f8f8f2',
    },
    type: 'dark',
    background: {
      default: '#272822',
    },
  },
  overrides: {
    MuiPaper: {
      root: {
        color: '#f8f8f2',
        backgroundColor: '#3b3a32',
        MuiSnackbarContent: {
          root: {
            color: '#f8f8f2',
            backgroundColor: '#f92672',
          },
          message: {
            color: '#f8f8f2',
            backgroundColor: '#f92672',
          },
        },
      },
    },
    MuiButton: {
      textPrimary: {
        color: '#66d9ef',
      },
      textSecondary: {
        color: '#f8f8f2',
      },
    },
    MuiChip: {
      clickable: {
        background: '#49483e',
      },
    },
    MuiFormGroup: {
      root: {
        color: '#f8f8f2',
      },
    },
    MuiFormHelperText: {
      root: {
        Mui: {
          error: {
            color: '#f92672',
          },
        },
      },
    },
    MuiTableHead: {
      root: {
        color: '#f8f8f2',
        background: '#3b3a32 !important',
      },
    },
    MuiTableCell: {
      root: {
        color: '#f8f8f2',
        background: '#3b3a32 !important',
      },
      head: {
        color: '#f8f8f2',
        background: '#3b3a32 !important',
      },
    },
    NDLogin: {
      systemNameLink: {
        color: '#66d9ef',
      },
      icon: {},
      welcome: {
        color: '#f8f8f2',
      },
      card: {
        minWidth: 300,
        background: '#3b3a32',
      },
      avatar: {},
      button: {
        boxShadow: '3px 3px 5px #272822',
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
    stylesheet: require('./monokai.css.js'),
  },
}
