import { makeStyles } from '@material-ui/core/styles'

export const useStyles = makeStyles((theme) => ({
  container: {
    [theme.breakpoints.down('xs')]: {
      padding: '0.7em',
      minWidth: '24em',
    },
    [theme.breakpoints.up('sm')]: {
      padding: '1em',
      minWidth: '32em',
    },
  },
  albumCover: {
    display: 'inline-block',
    [theme.breakpoints.down('xs')]: {
      height: '8em',
      width: '8em',
    },
    [theme.breakpoints.up('sm')]: {
      height: '10em',
      width: '10em',
    },
    [theme.breakpoints.up('lg')]: {
      height: '15em',
      width: '15em',
    },
  },
  albumDetails: {
    display: 'inline-block',
    verticalAlign: 'top',
    [theme.breakpoints.down('xs')]: {
      width: '14em',
    },
    [theme.breakpoints.up('sm')]: {
      width: '26em',
    },
    [theme.breakpoints.up('lg')]: {
      width: '38em',
    },
  },
  albumTitle: {
    whiteSpace: 'nowrap',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
  },
}))
