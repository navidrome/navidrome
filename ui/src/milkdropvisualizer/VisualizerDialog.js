import React, {
  useState,
  useEffect,
  useLayoutEffect,
  useRef,
  createRef,
  memo,
} from 'react'
import {
  createMuiTheme,
  makeStyles,
  ThemeProvider,
} from '@material-ui/core/styles'
import Button from '@material-ui/core/Button'
import Dialog from '@material-ui/core/Dialog'
import ListItemText from '@material-ui/core/ListItemText'
import ListItem from '@material-ui/core/ListItem'
import List from '@material-ui/core/List'
import Divider from '@material-ui/core/Divider'
import AppBar from '@material-ui/core/AppBar'
import Toolbar from '@material-ui/core/Toolbar'
import IconButton from '@material-ui/core/IconButton'
import Typography from '@material-ui/core/Typography'
import CloseIcon from '@material-ui/icons/Close'
import Fade from '@material-ui/core/Fade'
import { useSelector, useDispatch } from 'react-redux'
import useMediaQuery from '@material-ui/core/useMediaQuery'
import { showMilkdropVisualizer } from '../actions'
import useCurrentTheme from '../themes/useCurrentTheme'
// import MilkDropVisualizer from './MilkDropVisualizer'

// import butterchurn from 'butterchurn'
// import butterchurnPresets from 'butterchurn-presets'

const useStyles = makeStyles((theme) => ({
  dialog: {
    zIndex: ({ isMobile }) => (!isMobile ? '5 !important' : '1300'),
    marginBottom: ({ isMobile }) => (!isMobile ? '80px' : '0px'),
  },

  appBar: {
    // backgroundColor: theme.palette.primary.main,
    position: 'relative',
  },
  title: {
    color: theme.palette.primary.light,
    marginLeft: theme.spacing(2),
    flex: 1,
  },
  milkdrop: {
    height: '100%',
    width: '100%',
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

export function Visualizer() {
  const isMobile = useMediaQuery('(max-width:768px)')
  // console.log(isMobile)
  const classes = useStyles(isMobile)
  // const [open, setOpen] = React.useState(true)
  const showVisualization = useSelector(
    (state) => state.visualizer.showVisualization
  )
  const theme = useCurrentTheme()

  const dispatch = useDispatch()
  // const [dimension, setDimension] = useState({ x: 0, y: 0 })
  const WrapperDiv = useRef(null)
  useEffect(() => {
    console.log(WrapperDiv)
    //   setDimension({
    //     x: WrapperDiv.current?.clientWidth,
    //     y: WrapperDiv.current?.clientHeight,
    //   })
  }, [])

  console.log('hey')
  const handleClickOpen = () => {
    dispatch(showMilkdropVisualizer(true))
  }

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
          <IconButton
            edge="start"
            // color="inherit"
            onClick={handleClose}
            aria-label="close"
          >
            <CloseIcon />
          </IconButton>
          <Typography
            variant="h6"
            className={classes.title}
            // color="text.primary"
          >
            Navidrome
          </Typography>
          <Button autoFocus color="inherit" onClick={handleClose}>
            save
          </Button>
        </Toolbar>
      </AppBar>
      <div className={classes.milkdrop} ref={WrapperDiv}>
        {/* <MilkDropVisualizer /> */}
        <List>
          <ListItem button>
            <ListItemText primary="Phone ringtone" secondary="Titania" />
          </ListItem>
          <Divider />
          <ListItem button>
            <ListItemText
              primary="Default notification ringtone"
              secondary="Tethys"
            />
          </ListItem>
        </List>
      </div>
    </Dialog>
  )
}

export const VisualizerWithTheme = () => {
  return <Visualizer />
}
