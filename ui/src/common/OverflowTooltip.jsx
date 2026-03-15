import React from 'react'
import PropTypes from 'prop-types'
import { Tooltip } from '@material-ui/core'
import { makeStyles, alpha } from '@material-ui/core/styles'
import grey from '@material-ui/core/colors/grey'

const useStyles = makeStyles(
  (theme) => ({
    tooltip: {
      backgroundColor:
        theme.palette.type === 'dark'
          ? alpha(grey[700], 0.92)
          : alpha(grey[300], 0.92),
      color:
        theme.palette.type === 'dark'
          ? theme.palette.common.white
          : theme.palette.common.black,
      borderRadius: theme.shape.borderRadius,
      ...theme.typography.body2,
      padding: theme.spacing(0.5, 1),
      maxWidth: 300,
    },
  }),
  { name: 'NDOverflowTooltip' },
)

const transitionProps = { timeout: 0 }

export const OverflowTooltip = ({
  children,
  title,
  placement = 'bottom-start',
}) => {
  const classes = useStyles()
  const textRef = React.useRef(null)
  const [isOverflowing, setIsOverflowing] = React.useState(false)
  const tooltipClasses = React.useMemo(
    () => ({ tooltip: classes.tooltip }),
    [classes.tooltip],
  )

  React.useLayoutEffect(() => {
    const el = textRef.current
    if (!el) return

    const checkOverflow = () => {
      setIsOverflowing(el.scrollWidth > el.clientWidth)
    }

    const resizeObserver = new ResizeObserver(checkOverflow)
    resizeObserver.observe(el)

    checkOverflow()

    return () => resizeObserver.disconnect()
  }, [])

  const mergedRef = React.useCallback(
    (el) => {
      textRef.current = el

      const { ref } = children
      if (typeof ref === 'function') {
        ref(el)
      } else if (ref && typeof ref === 'object') {
        ref.current = el
      }
    },
    [children],
  )

  return (
    <Tooltip
      title={title}
      disableHoverListener={!isOverflowing}
      disableTouchListener
      placement={placement}
      TransitionProps={transitionProps}
      classes={tooltipClasses}
    >
      {React.cloneElement(children, { ref: mergedRef })}
    </Tooltip>
  )
}

OverflowTooltip.propTypes = {
  children: PropTypes.element.isRequired,
  title: PropTypes.string.isRequired,
  placement: PropTypes.string,
}
