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
        textDecoration: 'none',
        color: '#0085ff',
      },
      icon: {
        backgroundColor: 'transparent',
        width: '100px',
      },
      welcome: {
        color: '#eee',
      },
      card: {
        minWidth: 300,
        marginTop: '6em',
        overflow: 'visible',
        backgroundColor: '#424242ed',
      },
      avatar: {
        marginTop: '-50px',
      },
      button: {
        boxShadow: '3px 3px 5px #000000a3',
      },
    },
  },
  player: {
    theme: 'dark',
  },
}
