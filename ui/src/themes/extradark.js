import blue from '@material-ui/core/colors/blue'
import stylesheet from './dark.css.js'

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
    NDArtistPage: {
      bgContainer: {
        background:
          'linear-gradient(to bottom, rgba(52 52 52 / 72%), rgb(0 0 0))!important',
      },
    },
  },

  player: {
    theme: 'dark',
    stylesheet,
  },
}
