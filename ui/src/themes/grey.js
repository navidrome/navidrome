import blueGrey from '@material-ui/core/colors/blueGrey'

export default {
  themeName: 'Grey',
  palette: {
    secondary: blueGrey,
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
  },
  player: {
    theme: 'light',
  },
}
