import React, { useCallback } from 'react'
import PropTypes from 'prop-types'
import StarIcon from '@material-ui/icons/Star'
import StarBorderIcon from '@material-ui/icons/StarBorder'
import IconButton from '@material-ui/core/IconButton'
import { makeStyles } from '@material-ui/core/styles'
import { useToggleStar } from './useToggleStar'

const useStyles = makeStyles({
  star: {
    color: (props) => props.color,
    visibility: (props) =>
      props.visible === false
        ? 'hidden'
        : props.starred
        ? 'visible'
        : 'inherit',
  },
})

export const StarButton = ({
  resource,
  record,
  color,
  visible,
  size,
  component: Button,
  addLabel,
  disabled,
  ...rest
}) => {
  const classes = useStyles({ color, visible, starred: record.starred })
  const [toggleStar, loading] = useToggleStar(resource, record)

  const handleToggleStar = useCallback(
    (e) => {
      e.preventDefault()
      toggleStar()
      e.stopPropagation()
    },
    [toggleStar]
  )

  return (
    <Button
      onClick={handleToggleStar}
      size={'small'}
      disabled={disabled || loading}
      className={classes.star}
      {...rest}
    >
      {record.starred ? (
        <StarIcon fontSize={size} />
      ) : (
        <StarBorderIcon fontSize={size} />
      )}
    </Button>
  )
}

StarButton.propTypes = {
  resource: PropTypes.string.isRequired,
  record: PropTypes.object.isRequired,
  visible: PropTypes.bool,
  color: PropTypes.string,
  size: PropTypes.string,
  component: PropTypes.object,
  disabled: PropTypes.bool,
}

StarButton.defaultProps = {
  addLabel: true,
  record: {},
  visible: true,
  size: 'small',
  color: 'inherit',
  component: IconButton,
  disabled: false,
}
