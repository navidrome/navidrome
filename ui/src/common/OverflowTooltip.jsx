import React from 'react'
import PropTypes from 'prop-types'
import { Tooltip } from '@material-ui/core'
import useMenuTooltipStyles from './useMenuTooltipStyles'

export const OverflowTooltip = ({
  children,
  title,
  placement = 'bottom-start',
}) => {
  const classes = useMenuTooltipStyles()
  const textRef = React.useRef(null)
  const [isOverflowing, setIsOverflowing] = React.useState(false)

  React.useLayoutEffect(() => {
    const el = textRef.current
    if (!el) return

    const checkOverflow = () =>
      setIsOverflowing(el.scrollWidth > el.clientWidth)

    checkOverflow()
    window.addEventListener('resize', checkOverflow)

    return () => window.removeEventListener('resize', checkOverflow)
  }, [title, children])

  return (
    <Tooltip
      title={title}
      disableHoverListener={!isOverflowing}
      disableTouchListener
      placement={placement}
      TransitionProps={{ timeout: 0 }}
      classes={{ tooltip: classes.tooltip }}
    >
      {React.cloneElement(children, { ref: textRef })}
    </Tooltip>
  )
}

OverflowTooltip.propTypes = {
  children: PropTypes.element.isRequired,
  title: PropTypes.string.isRequired,
  placement: PropTypes.string,
}
