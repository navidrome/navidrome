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
  },
}
