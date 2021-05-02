import React, { useCallback } from 'react'
import PropTypes from 'prop-types'
import FavoriteIcon from '@material-ui/icons/Favorite'
import FavoriteBorderIcon from '@material-ui/icons/FavoriteBorder'
import IconButton from '@material-ui/core/IconButton'
import { makeStyles } from '@material-ui/core/styles'
import { useToggleLove } from './useToggleLove'
import { useRefresh } from 'react-admin'
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
  refreshPage,
  ...rest
}) => {
  const classes = useStyles({ color, visible, loved: record.starred })
  const [toggleLove, loading] = useToggleLove(resource, record)
  const refresh = useRefresh()
  const handleToggleLove = useCallback(
    (e) => {
      e.preventDefault()
      toggleLove()
      if (refreshPage) refresh()
      e.stopPropagation()
    },
    [toggleLove, refresh, refreshPage]
  )

  return (
    <Button
      onClick={handleToggleLove}
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
