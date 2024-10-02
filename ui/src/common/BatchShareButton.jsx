import React from 'react'
import { Button, useTranslate, useUnselectAll } from 'react-admin'
import { useDispatch } from 'react-redux'
import { openShareMenu } from '../actions'
import ShareIcon from '@material-ui/icons/Share'

export const BatchShareButton = ({ resource, selectedIds, className }) => {
  const dispatch = useDispatch()
  const translate = useTranslate()
  const unselectAll = useUnselectAll()

  const share = () => {
    dispatch(
      openShareMenu(
        selectedIds,
        resource,
        translate('ra.action.bulk_actions', {
          _: 'ra.action.bulk_actions',
          smart_count: selectedIds.length,
        }),
        'message.shareBatchDialogTitle',
      ),
    )
    unselectAll(resource)
  }

  const caption = translate('ra.action.share')
  return (
    <Button
      aria-label={caption}
      onClick={share}
      label={caption}
      className={className}
    >
      <ShareIcon />
    </Button>
  )
}

BatchShareButton.propTypes = {}
