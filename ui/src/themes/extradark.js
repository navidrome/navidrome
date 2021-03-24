import blue from '@material-ui/core/colors/blue'

export default {
  themeName: 'Extra Dark',
  palette: {
    background: {
      paper: '#000000',
      default: '#000000',
    },
    primary: {
      main: '#0f60b6',
      contrastText: '#909090',
    },
    secondary: blue,
    extra: {
      bordercolors: '#202020',
    },
    type: 'dark',
    extraAttribute: {
      theme: 'extradark',
      lines: '#353535',
      subtitle: '#555555',
    },
  },
  overrides: {
    MuiFormGroup: {
      root: {
        color: 'white',
      },
    },
  },
  player: {
    theme: 'extradark',
  },
}
