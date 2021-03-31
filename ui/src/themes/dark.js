import blue from '@material-ui/core/colors/blue'

export default {
  themeName: 'Dark',
  palette: {
    primary: {
      main: '#90caf9',
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
    NDLogin: {
      systemNameLink: {
        color: '#fff',
      },
      icon: {
        backgroundColor: 'inherit',
      },
      welcome: {
        color: '#eee',
      },
    },
  },
  player: {
    theme: 'dark',
  },
}
