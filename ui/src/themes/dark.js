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
        '@media screen and (max-width:600px)': {
          textDecoration: 'none',
          color: '#0085ff',
        },
      },
      icon: {
        backgroundColor: 'inherit',
        '@media screen and (max-width:600px)': {
          backgroundColor: 'transparent',
          width: '100px',
        },
      },
      welcome: {
        color: '#eee',
      },
      card: {
        minWidth: 300,
        marginTop: '6em',
        '@media screen and (max-width:600px)': {
          overflow: 'visible',
          backgroundColor: '#424242ed',
        },
      },
      avatar: {
        '@media screen and (max-width:600px)': {
          marginTop: '-50px',
        },
      },
      button: {
        '@media screen and (max-width:600px)': {
          borderRadius: '25px',
          backgroundColor: '#0085ff',
          boxShadow: '3px 3px 5px #000000a3',
        },
      },
    },
  },
  player: {
    theme: 'dark',
  },
}
