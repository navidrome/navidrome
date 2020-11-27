import React, { useCallback, useEffect, useRef, useState } from 'react'
import PropTypes from 'prop-types'
import { useNotify, useDataProvider } from 'react-admin'
import StarIcon from '@material-ui/icons/Star'
import StarBorderIcon from '@material-ui/icons/StarBorder'
import IconButton from '@material-ui/core/IconButton'
import { makeStyles } from '@material-ui/core/styles'
import subsonic from '../subsonic'

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
  ...rest
}) => {
  const [loading, setLoading] = useState(false)
  const classes = useStyles({ color, visible, starred: record.starred })
  const notify = useNotify()

  const mountedRef = useRef(false)
  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
    }
  }, [])

  const dataProvider = useDataProvider()

  const refreshRecord = useCallback(() => {
    dataProvider.getOne(resource, { id: record.id }).then(() => {
      if (mountedRef.current) {
        setLoading(false)
      }
    })
  }, [dataProvider, record.id, resource])

  const handleToggleStar = (e) => {
    e.preventDefault()
    const toggleStar = record.starred ? subsonic.unstar : subsonic.star

    setLoading(true)
    toggleStar(record.id)
      .then(refreshRecord)
      .catch((e) => {
        console.log('Error toggling star: ', e)
        notify('ra.page.error', 'warning')
        if (mountedRef.current) {
          setLoading(false)
        }
      })
    e.stopPropagation()
  }

  return (
    <Button
      onClick={handleToggleStar}
      size={'small'}
      disabled={loading}
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
}

StarButton.defaultProps = {
  addLabel: true,
  record: {},
  visible: true,
  size: 'small',
  color: 'inherit',
  component: IconButton,
}
