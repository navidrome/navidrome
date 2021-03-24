import blue from '@material-ui/core/colors/blue'

export default {
  themeName: 'Dark',
  palette: {
    primary: {
      main: '#90caf9',
    },
    secondary: blue,
    type: 'dark',
    extraAttribute: {
      theme: 'dark',
    },
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
