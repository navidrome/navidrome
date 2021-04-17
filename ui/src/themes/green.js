import green from '@material-ui/core/colors/green'

export default {
  themeName: 'Green',
  palette: {
    primary: {
      light: green['300'],
      main: green['500'],
    },
    secondary: {
      main: green['900'],
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
  },
  player: {
    theme: 'dark',
  },
}
