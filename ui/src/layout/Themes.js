import blue from '@material-ui/core/colors/blue'

export const DarkTheme = {
  palette: {
    secondary: blue,
    type: 'dark'
  }
}

export const LightTheme = {
  palette: {
    secondary: {
      light: '#5f5fc4',
      main: '#283593',
      dark: '#001064',
      contrastText: '#fff'
    }
  },
  overrides: {
    MuiFilledInput: {
      root: {
        backgroundColor: 'rgba(0, 0, 0, 0.04)',
        '&$disabled': {
          backgroundColor: 'rgba(0, 0, 0, 0.04)'
        }
      }
    }
  }
}
