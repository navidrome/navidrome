export default {
  themeName: 'Light',
  palette: {
    secondary: {
      light: '#5f5fc4',
      dark: '#001064',
      main: '#3f51b5',
      contrastText: '#fff',
    },
    type: 'light',
    extraAttribute: {
      theme: 'light',
    },
  },
  overrides: {
    MuiFilledInput: {
      root: {
        backgroundColor: 'rgba(0, 0, 0, 0.04)',
        '&$disabled': {
          backgroundColor: 'rgba(0, 0, 0, 0.04)',
        },
      },
    },
  },
  player: {
    theme: 'light',
  },
}
