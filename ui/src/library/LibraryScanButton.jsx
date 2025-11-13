import React, { useState } from 'react'
import PropTypes from 'prop-types'
import {
  Button,
  useNotify,
  useRefresh,
  useTranslate,
  useUnselectAll,
} from 'react-admin'
import { useSelector } from 'react-redux'
import SyncIcon from '@material-ui/icons/Sync'
import CachedIcon from '@material-ui/icons/Cached'
import subsonic from '../subsonic'

const LibraryScanButton = ({ fullScan, selectedIds, className }) => {
  const [loading, setLoading] = useState(false)
  const notify = useNotify()
  const refresh = useRefresh()
  const translate = useTranslate()
  const unselectAll = useUnselectAll()
  const scanStatus = useSelector((state) => state.activity.scanStatus)

  const handleClick = async () => {
    setLoading(true)
    try {
      // Build scan options
      const options = { fullScan }

      // If specific libraries are selected, scan only those
      // Format: "libraryID:" to scan entire library (no folder path specified)
      if (selectedIds && selectedIds.length > 0) {
        options.target = selectedIds.map((id) => `${id}:`)
      }

      await subsonic.startScan(options)
      const notificationKey = fullScan
        ? 'resources.library.notifications.fullScanStarted'
        : 'resources.library.notifications.quickScanStarted'
      notify(notificationKey, 'info')
      refresh()

      // Unselect all items after successful scan
      unselectAll('library')
    } catch (error) {
      notify('resources.library.notifications.scanError', 'warning')
    } finally {
      setLoading(false)
    }
  }

  const isDisabled = loading || scanStatus.scanning

  const label = fullScan
    ? translate('resources.library.actions.fullScan')
    : translate('resources.library.actions.quickScan')

  const icon = fullScan ? <CachedIcon /> : <SyncIcon />

  return (
    <Button
      onClick={handleClick}
      disabled={isDisabled}
      label={label}
      className={className}
    >
      {icon}
    </Button>
  )
}

LibraryScanButton.propTypes = {
  fullScan: PropTypes.bool.isRequired,
  selectedIds: PropTypes.array,
  className: PropTypes.string,
}

export default LibraryScanButton
