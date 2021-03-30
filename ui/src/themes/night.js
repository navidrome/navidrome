import blue from '@material-ui/core/colors/blue'

export default {
  themeName: 'night',
  palette: {
    background: {
      paper: '#000000',
      default: '#000000',
    },
    primary: {
      main: '#0f60b6',
      contrastText: '#909090',
    },
    secondary: blue,
    type: 'dark',
    extraAttribute: {
      theme: 'extradark',
      subtitle: '#555555',
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
