import React from 'react'
import PropTypes from 'prop-types'
import { useNotify, useRefresh, useUpdate } from 'react-admin'
import StarIcon from '@material-ui/icons/Star'
import StarBorderIcon from '@material-ui/icons/StarBorder'
import IconButton from '@material-ui/core/IconButton'
import { makeStyles } from '@material-ui/core/styles'

const useStyles = makeStyles({
  star: {
    color: (props) => props.color,
    visibility: (props) =>
      props.visible || props.starred ? 'visible' : 'hidden',
  },
})

const StarButton = ({ resource, record, color, visible, size }) => {
  const classes = useStyles({ color, visible, starred: record.starred })
  const notify = useNotify()
  const refresh = useRefresh()

  const [toggleStarred, { loading }] = useUpdate(
    resource,
    record.id,
    {
      ...record,
      starred: !record.starred,
    },
    {
      undoable: false,
      onFailure: (error) => {
        console.log(error)
        notify('ra.page.error', 'warning')
        refresh()
      },
    }
  )

  const handleToggleStar = (e) => {
    e.preventDefault()
    toggleStarred()
    e.stopPropagation()
  }

  return (
    <IconButton
      onClick={handleToggleStar}
      size={'small'}
      disabled={loading}
      className={classes.star}
    >
      {record.starred ? (
        <StarIcon fontSize={size} />
      ) : (
        <StarBorderIcon fontSize={size} />
      )}
    </IconButton>
  )
}

StarButton.propTypes = {
  resource: PropTypes.string.isRequired,
  record: PropTypes.object.isRequired,
  visible: PropTypes.bool,
  color: PropTypes.string,
  size: PropTypes.string,
}

StarButton.defaultProps = {
  addLabel: true,
  record: {},
  visible: true,
  size: 'small',
  color: 'inherit',
}

export default StarButton
