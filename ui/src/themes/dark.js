import blue from '@material-ui/core/colors/blue'

export default {
  themeName: 'Dark (default)',
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
