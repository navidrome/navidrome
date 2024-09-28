import blue from '@material-ui/core/colors/blue'
import stylesheet from './dark.css.js'

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
        color: '#0085ff',
      },
      icon: {},
      welcome: {
        color: '#eee',
      },
      card: {
        minWidth: 300,
        backgroundColor: '#424242ed',
      },
      avatar: {},
      button: {
        boxShadow: '3px 3px 5px #000000a3',
      },
    },
    NDMobileArtistDetails: {
      bgContainer: {
        background:
          'linear-gradient(to bottom, rgba(52 52 52 / 72%), rgb(48 48 48))!important',
      },
    },
  },
  player: {
    theme: 'dark',
    stylesheet,
  },
}
