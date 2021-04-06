import React from 'react'
import PropTypes from 'prop-types'
import ArrowDropUpIcon from '@material-ui/icons/ArrowDropUp'
import ArrowDropDownIcon from '@material-ui/icons/ArrowDropDown'
import IconButton from '@material-ui/core/IconButton'
import { makeStyles } from '@material-ui/core/styles'
import { useDispatch, useSelector } from 'react-redux'
import { showMilkdropVisualizer } from '../actions'

const useStyles = makeStyles({
  toggleButton: {
    height: '30px',
    width: '30px',
    color: (props) => props.color,
    visibility: (props) =>
      props.visible === false
        ? 'hidden'
        : props.checked
        ? 'visible'
        : 'inherit',
    '&. MuiSvgIcon': {
      fontSize: '2.1875rem !important',
    },
  },
})

export const ToggleButton = ({
  color,
  visible,
  size,
  component: Button,
  disabled,
  ...rest
}) => {
  const showVisualization = useSelector((state) => {
    // console.log(state)
    return state?.visualizer?.showVisualization
  })
  const dispatch = useDispatch()
  const classes = useStyles({ color, visible, checked: showVisualization })

  return (
    <Button
      onClick={() => {
        dispatch(showMilkdropVisualizer(!showVisualization))
      }}
      disabled={disabled}
      className={classes.toggleButton}
      {...rest}
    >
      {showVisualization ? (
        <ArrowDropDownIcon fontSize={size} />
      ) : (
        <ArrowDropUpIcon fontSize={size} />
      )}
    </Button>
  )
}

ToggleButton.propTypes = {
  visible: PropTypes.bool,
  color: PropTypes.string,
  size: PropTypes.string,
  component: PropTypes.object,
  disabled: PropTypes.bool,
}

ToggleButton.defaultProps = {
  visible: true,
  size: 'large',
  color: 'inherit',
  component: IconButton,
  disabled: false,
}
