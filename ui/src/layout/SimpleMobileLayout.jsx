import React from 'react'
import { useSelector } from 'react-redux'
import { Button, Typography } from '@material-ui/core'
import { ThemeProvider, makeStyles } from '@material-ui/core/styles'
import { useTranslate } from 'react-admin'
import useCurrentTheme from '../themes/useCurrentTheme'
import { ShuffleAllButton } from '../common/ShuffleAllButton'
import Notification from './Notification'
import { setSimpleMobilePref } from './simpleMobile'

const useStyles = makeStyles((theme) => {
  const space = (...args) => {
    const v = theme.spacing(...args)
    return typeof v === 'number' ? `${v}px` : String(v)
  }
  return {
    root: {
      position: 'relative',
      zIndex: 2,
      minHeight: '100vh',
      '@supports (min-height: 100dvh)': { minHeight: '100dvh' },
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'stretch',
      justifyContent: 'center',
      backgroundColor: theme.palette.background.default,
      color: theme.palette.text.primary,
      padding: space(3),
      paddingBottom: (props) =>
        props.addPadding
          ? `calc(${space(12)} + env(safe-area-inset-bottom, 0px))`
          : `calc(${space(3)} + env(safe-area-inset-bottom, 0px))`,
      boxSizing: 'border-box',
    },
    title: {
      textAlign: 'center',
      marginBottom: space(4),
      color: theme.palette.text.primary,
    },
    actions: {
      display: 'flex',
      flexDirection: 'column',
      gap: space(2),
      width: '100%',
      maxWidth: 420,
      margin: '0 auto',
    },
    full: {
      minHeight: 56,
      fontSize: '1rem',
      textTransform: 'none',
      color: theme.palette.text.primary,
      border: `1px solid ${theme.palette.divider}`,
    },
  }
})

const SimpleMobileLayout = () => {
  const theme = useCurrentTheme()
  const translate = useTranslate()
  const queue = useSelector((state) => state.player?.queue) || []
  const classes = useStyles({ addPadding: queue.length > 0 })

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
            variant="outlined"
            color="inherit"
            onClick={() => setSimpleMobilePref(false)}
            data-testid="open-full-version"
          >
            {translate('menu.openFullVersion', { _: 'Open full version' })}
          </Button>
        </div>
      </div>
      <Notification />
    </ThemeProvider>
  )
}

export default SimpleMobileLayout
