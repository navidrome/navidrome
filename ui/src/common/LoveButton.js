import React, { useCallback } from 'react'
import PropTypes from 'prop-types'
import FavoriteIcon from '@material-ui/icons/Favorite'
import FavoriteBorderIcon from '@material-ui/icons/FavoriteBorder'
import IconButton from '@material-ui/core/IconButton'
import { makeStyles } from '@material-ui/core/styles'
import { useToggleStar } from './useToggleStar'

const useStyles = makeStyles({
  love: {
    color: (props) => props.color,
    visibility: (props) =>
      props.visible === false ? 'hidden' : props.loved ? 'visible' : 'inherit',
  },
})

export const LoveButton = ({
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
  const classes = useStyles({ color, visible, loved: record.starred })
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
      className={classes.love}
      {...rest}
    >
      {record.starred ? (
        <FavoriteIcon fontSize={size} />
      ) : (
        <FavoriteBorderIcon fontSize={size} />
      )}
    </Button>
  )
}

LoveButton.propTypes = {
  resource: PropTypes.string.isRequired,
  record: PropTypes.object.isRequired,
  visible: PropTypes.bool,
  color: PropTypes.string,
  size: PropTypes.string,
  component: PropTypes.object,
  disabled: PropTypes.bool,
}

LoveButton.defaultProps = {
  addLabel: true,
  record: {},
  visible: true,
  size: 'small',
  color: 'inherit',
  component: IconButton,
  disabled: false,
}
