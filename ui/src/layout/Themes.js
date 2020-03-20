import blue from '@material-ui/core/colors/blue'

export const DarkTheme = {
  palette: {
    primary: {
      main: '#90caf9'
    },
    secondary: blue,
    type: 'dark'
  },
  overrides: {
    MuiFormGroup: {
      root: {
        color: 'white'
      }
    }
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
