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
  playButton: {
    opacity: 0,
    transition: 'all 150ms ease-out',
  },
  albumCover: {
    display: 'inline-flex',
    justifyContent: 'center',
    alignItems: 'center',
    cursor: 'pointer',

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
    '&:hover $playButton': {
      opacity: 1,
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
      width: '43em',
    },
  },
  albumTitle: {
    whiteSpace: 'nowrap',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
  },
}))
