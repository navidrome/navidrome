import stylesheet from './electricPurple.css.js'

export default {
  themeName: 'Electric Purple',
  palette: {
    primary: {
      light: '#f757ff',
      dark: '#8800cb',
      main: '#bf00ff',
      contrastText: '#fff',
    },
    secondary: {
      light: '#bd4aff',
      dark: '#530099',
      main: '#8800cb',
      contrastText: '#fff',
    },
    warn: {
      light: '#ffff82',
      dark: '#c9bf07',
      main: '#fff14e',
      contrastText: '#000',
    },
    error: {
      light: '#ff763a',
      dark: '#c30000',
      main: '#ff3f00',
      contrastText: '#000',
    },
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
