export default {
  themeName: 'Spotify',
  palette: {
    primary: {
      main: '#ffffff',
    },
    secondary: {
      light: '#4ac776',
      dark: '#14813a',
      main: '#1DB954',
    },
    type: 'dark',
  },
  overrides: {
    MuiFormGroup: {
      root: {
        color: 'white',
      },
    },
    player: {
      theme: 'dark',
    },
  },
}
