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
    NDArtistPage: {
      bgContainer: {
        background:
          'linear-gradient(to bottom, rgba(52 52 52 / 72%), rgb(48 48 48))!important',
      },
      more: {
        boxShadow: '-10px 0px 18px 5px #303030!important',
      },
      less: {
        boxShadow: '-10px 0px 18px 5px #303030!important',
      },
    },
  },
  player: {
    theme: 'dark',
  },
}
