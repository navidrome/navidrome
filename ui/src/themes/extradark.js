import blue from '@material-ui/core/colors/blue'

export default {
  themeName: 'Extra Dark',
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
      welcome: {
        color: '#eee',
      },
    },
  },
  player: {
    theme: 'dark',
    stylesheet: require('./dark.css.js'),
  },
}
