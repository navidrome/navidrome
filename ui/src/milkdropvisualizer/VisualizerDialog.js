import React, { lazy, Suspense } from 'react'
import {
  createMuiTheme,
  makeStyles,
  ThemeProvider,
} from '@material-ui/core/styles'
import Dialog from '@material-ui/core/Dialog'
import AppBar from '@material-ui/core/AppBar'
import Toolbar from '@material-ui/core/Toolbar'
import IconButton from '@material-ui/core/IconButton'
import Typography from '@material-ui/core/Typography'
import CloseIcon from '@material-ui/icons/Close'
import Fade from '@material-ui/core/Fade'
import { useSelector, useDispatch } from 'react-redux'
import { showMilkdropVisualizer } from '../actions'
import useCurrentTheme from '../themes/useCurrentTheme'
const MilkDropVisualizer = lazy(() => import('./MilkDropVisualizer'))

const useStyles = makeStyles((theme) => ({
  dialog: {
    zIndex: '5 !important',
    marginBottom: '80px',
  },
  '@media (max-width: 768px)': {
    dialog: {
      zIndex: 1300,
      marginBottom: '0px',
    },
  },

  appBar: {
    position: 'relative',
  },
  title: {
    color: theme.palette.primary.main,
    marginLeft: theme.spacing(2),
    flex: 1,
  },
  milkdrop: {
    display: 'flex',
    flexGrow: 1,
  },
}))
const Transition = React.forwardRef(function Transition(props, ref) {
  const duration = {
    enteringScreen: 200,
    leavingScreen: 0,
  }
  // return <Slide direction="left" ref={ref} {...props} />;
  return (
    <Fade
      ref={ref}
      {...props}
      timeout={{ enter: duration.enteringScreen, exit: duration.leavingScreen }}
    />
  )
})

export const VisualizerDialog = () => {
  const classes = useStyles()
  const showVisualization = useSelector(
    (state) => state.visualizer.showVisualization
  )

  const dispatch = useDispatch()

  const handleClose = () => {
    dispatch(showMilkdropVisualizer(false))
  }
  return (
    <Dialog
      fullScreen
      open={showVisualization}
      onClose={handleClose}
      TransitionComponent={Transition}
      className={classes.dialog}
    >
      <AppBar className={classes.appBar}>
        <Toolbar>
          <IconButton edge="start" onClick={handleClose} aria-label="close">
            <CloseIcon />
          </IconButton>
          <Typography variant="h6" className={classes.title}>
            Navidrome
          </Typography>
          {/*  add preset dialog here */}
        </Toolbar>
      </AppBar>
      <div className={classes.milkdrop}>
        <Suspense fallback={<div>Loading...</div>}>
          <MilkDropVisualizer />
        </Suspense>
      </div>
    </Dialog>
  )
}

export const Visualizer = () => {
  const theme = useCurrentTheme()

  const enableVisualization = useSelector(
    (state) => state.settings.visualization
  )

  return (
    <ThemeProvider theme={createMuiTheme(theme)}>
      {enableVisualization ? <VisualizerDialog /> : <></>}
    </ThemeProvider>
  )
}
