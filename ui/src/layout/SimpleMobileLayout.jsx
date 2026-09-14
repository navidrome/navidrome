import React from 'react'
import { useSelector } from 'react-redux'
import { Button, Typography } from '@material-ui/core'
import { ThemeProvider, makeStyles } from '@material-ui/core/styles'
import { useTranslate } from 'react-admin'
import useCurrentTheme from '../themes/useCurrentTheme'
import { ShuffleAllButton } from '../common'
import Notification from './Notification'
import { setSimpleMobilePref } from './simpleMobile'

const useStyles = makeStyles((theme) => ({
  root: {
    minHeight: '100dvh',
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'stretch',
    justifyContent: 'center',
    backgroundColor: theme.palette.background.default,
    color: theme.palette.text.primary,
    padding: theme.spacing(3),
    paddingBottom: (props) =>
      props.addPadding
        ? `calc(${theme.spacing(12)}px + env(safe-area-inset-bottom))`
        : `calc(${theme.spacing(3)}px + env(safe-area-inset-bottom))`,
    boxSizing: 'border-box',
  },
  title: {
    textAlign: 'center',
    marginBottom: theme.spacing(4),
  },
  actions: {
    display: 'flex',
    flexDirection: 'column',
    gap: theme.spacing(2),
    width: '100%',
    maxWidth: 420,
    margin: '0 auto',
  },
  full: {
    minHeight: 56,
    fontSize: '1rem',
    textTransform: 'none',
  },
}))

const SimpleMobileLayout = () => {
  const theme = useCurrentTheme()
  const translate = useTranslate()
  const queue = useSelector((state) => state.player?.queue)
  const classes = useStyles({ addPadding: queue?.length > 0 })

  return (
    <ThemeProvider theme={theme}>
      <div className={classes.root} data-testid="simple-mobile-layout">
        <Typography variant="h5" className={classes.title}>
          Navidrome
        </Typography>
        <div className={classes.actions}>
          <ShuffleAllButton variant="hero" />
          <Button
            className={classes.full}
            onClick={() => setSimpleMobilePref(false)}
            data-testid="open-full-version"
          >
            {translate('menu.openFullVersion')}
          </Button>
        </div>
      </div>
      <Notification />
    </ThemeProvider>
  )
}

export default SimpleMobileLayout
