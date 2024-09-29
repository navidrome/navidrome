import stylesheet from './light.css.js'

export default {
  themeName: 'Light',
  palette: {
    secondary: {
      light: '#5f5fc4',
      dark: '#001064',
      main: '#3f51b5',
      contrastText: '#fff',
    },
  },
  overrides: {
    MuiFilledInput: {
      root: {
        backgroundColor: 'rgba(0, 0, 0, 0.04)',
        '&$disabled': {
          backgroundColor: 'rgba(0, 0, 0, 0.04)',
        },
      },
    },
    NDLogin: {
      main: {
        '& .MuiFormLabel-root': {
          color: '#000000',
        },
        '& .MuiFormLabel-root.Mui-focused': {
          color: '#0085ff',
        },
        '& .MuiFormLabel-root.Mui-error': {
          color: '#f44336',
        },
        '& .MuiInput-underline:after': {
          borderBottom: '2px solid #0085ff',
        },
      },
      card: {
        minWidth: 300,
        marginTop: '6em',
        backgroundColor: '#ffffffe6',
      },
      avatar: {},
      icon: {},
      button: {
        boxShadow: '3px 3px 5px #000000a3',
      },
      systemNameLink: {
        color: '#0085ff',
      },
    },
    NDMobileArtistDetails: {
      bgContainer: {
        background:
          'linear-gradient(to bottom, rgb(255 255 255 / 51%), rgb(250 250 250))!important',
      },
    },
  },
  player: {
    theme: 'light',
    stylesheet,
  },
}
