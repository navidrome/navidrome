import React, { useCallback } from 'react'
import PropTypes from 'prop-types'
import Tooltip from '@material-ui/core/Tooltip'
import { makeStyles } from '@material-ui/core/styles'
import { Button, usePermissions, useTranslate } from 'react-admin'
import { MdRefresh } from 'react-icons/md'
import { useRefreshMetadata } from './useRefreshMetadata'

const useStyles = makeStyles({
  // Tooltip needs a ref-holding child, and react-admin's Button does not forward one.
  wrapper: { display: 'inline-flex', verticalAlign: 'middle' },
  button: { minWidth: 'auto' },
})

// react-admin's Button, not an IconButton: the toolbars use it, so colour and the icon-only swap
// at xs match without restating either rule.
export const RefreshMetadataButton = ({
  resource,
  record,
  className,
  size,
}) => {
  const translate = useTranslate()
  const { permissions } = usePermissions()
  const refreshMetadata = useRefreshMetadata()
  const classes = useStyles()

  const handleClick = useCallback(
    () => refreshMetadata(resource, record?.id),
    [refreshMetadata, resource, record],
  )

  if (permissions !== 'admin' || !record?.id) return null

  const label = translate('resources.album.actions.refresh')
  return (
    <Tooltip title={label}>
      <span className={classes.wrapper}>
        <Button
          aria-label={label}
          className={className}
          classes={{ button: classes.button }}
          size={size}
          onClick={handleClick}
        >
          <MdRefresh />
        </Button>
      </span>
    </Tooltip>
  )
}

RefreshMetadataButton.propTypes = {
  resource: PropTypes.oneOf(['album', 'artist']).isRequired,
  record: PropTypes.object,
  className: PropTypes.string,
  size: PropTypes.oneOf(['small', 'medium']),
}

RefreshMetadataButton.defaultProps = {
  size: 'small',
}
