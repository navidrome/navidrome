import blue from '@material-ui/core/colors/blue'

export default {
  themeName: 'Dark',
  palette: {
    primary: {
      main: '#00fcf4',
    },
    secondary: blue,
    type: 'dark',
  },
  overrides: {
    MuiFormGroup: {
      root: {
        color: 'white',
      },
    },
  },
  player: {
    theme: 'dark',
  },
}
