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
        '@media screen and (max-width:600px)': {
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
          '& .MuiInput-underline:after': {
            borderBottom: '2px solid #0085ff',
          },
        },
      },
      card: {
        minWidth: 300,
        marginTop: '6em',
        '@media screen and (max-width:600px)': {
          overflow: 'visible',
          backgroundColor: '#ffffffe6',
        },
      },
      avatar: {
        '@media screen and (max-width:600px)': {
          marginTop: '-50px',
        },
      },
      icon: {
        '@media screen and (max-width:600px)': {
          backgroundColor: 'transparent',
          width: '100px',
        },
      },
      button: {
        '@media screen and (max-width:600px)': {
          borderRadius: '25px',
          backgroundColor: '#0085ff',
          boxShadow: '3px 3px 5px #000000a3',
        },
      },
      systemNameLink: {
        '@media screen and (max-width:600px)': {
          textDecoration: 'none',
          color: '#0085ff',
        },
      },
    },
  },
  player: {
    theme: 'light',
  },
}
