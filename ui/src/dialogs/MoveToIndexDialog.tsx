import React, { useCallback, useMemo, useState } from 'react'
import PropTypes from 'prop-types'
import { useDispatch, useSelector } from 'react-redux'
import { useDataProvider, useNotify, useTranslate } from 'react-admin'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  List,
  ListItem,
  ModalProps,
  TextField,
} from '@material-ui/core'
import { closeMoveToIndexDialog } from '../actions/dialogs'
import debounce from 'lodash.debounce'
import type { Identifier, Record } from 'ra-core'

interface PreviewItem {
  elem: Record
  newIndex: number
  willMove: boolean
}

interface PreviewListItemProps {
  loading?: boolean
  valid?: boolean
  moveDirection: 'no' | 'up' | 'down'
  item: PreviewItem | null
}

const PreviewListItem: React.FC<PreviewListItemProps> = ({
  loading,
  item,
  valid,
  moveDirection,
}) => {
  if (loading) {
    return <ListItem disableGutters>Loading...</ListItem>
  }

  if (!valid || !item) {
    return <ListItem disableGutters>---</ListItem>
  }

  return (
    <ListItem disableGutters>
      {item.newIndex}
      {item.willMove ? (moveDirection == 'up' ? '↑' : '↓') : null} -{' '}
      {item.elem.title} - {item.elem.album} - {item.elem.artist}
    </ListItem>
  )
}

/**
 * Calculate the optimal number of page items and the page number
    to include targetIndex and its surrounding items.
 */
function CalculatePagination(
  listLength: number,
  targetIndex: number,
): { itemsPerPage: number; pageNumber: number } {
  // Ensure valid inputs
  if (targetIndex < 0 || targetIndex >= listLength) {
    throw new Error('targetIndex must be within the range of listLength.')
  }

  if (targetIndex < 5) {
    return { itemsPerPage: 10, pageNumber: 1 }
  }

  let modulo = 0
  let divisor = 6
  // we need 3 items before and after in a page, we just go until we find one.
  // usually it takes 3-5 iterations
  while (divisor - modulo < 3 || modulo < 3) {
    divisor++
    modulo = targetIndex % divisor
  }

  const itemsPerPage = divisor
  const pageNumber = Math.ceil(targetIndex / itemsPerPage)

  return { itemsPerPage, pageNumber }
}

interface MoveToIndexDialogProps {
  title?: string
  onSuccess: (from: Identifier, to: string) => void
  max: number
  playlistId: string
}

const MoveToIndexDialog: React.FC<MoveToIndexDialogProps> = ({
  title,
  onSuccess,
  max,
  playlistId,
}) => {
  const { open, record } = useSelector((state) => state.moveToIndexDialog)
  const dispatch = useDispatch()
  const translate = useTranslate()
  const [to, setTo] = useState('1')
  const [validationError, setValidationError] = useState<string | undefined>()
  const [targetArea, setTargetArea] = useState<Record[]>([])
  const [loading, setLoading] = useState(false)
  const notify = useNotify()
  const dataProvider = useDataProvider()

  React.useEffect(() => {
    if (!to) {
      setValidationError(translate('ra.validation.required'))
      return
    }

    const value = parseInt(to)
    if (Number.isNaN(value)) {
      setValidationError(translate('ra.validation.number'))
      return
    }

    if (value < 1) {
      setValidationError(translate('ra.validation.minValue', { min: 0 }))
      return
    }

    if (value > max) {
      setValidationError(translate('ra.validation.maxValue', { max: max }))
      return
    }

    setValidationError(undefined)
  }, [to, max, translate])

  const getPreviewData = React.useRef(
    debounce(
      (to: string, max: number, playlistId: string) => {
        if (!to || parseInt(to) > max) {
          setLoading(false)
          return
        }

        const { itemsPerPage, pageNumber } = CalculatePagination(
          max,
          parseInt(to),
        )

        // TODO: stop interfering with the playlist page
        dataProvider
          .getList('playlistTrack', {
            pagination: { page: pageNumber, perPage: itemsPerPage },
            sort: { field: 'id', order: 'ASC' },
            filter: { playlist_id: playlistId },
          })
          .then((e) => {
            const target = e.data.findIndex((x) => x.id == to)
            // should not happen
            if (target == -1) {
              setLoading(false)
              return
            }

            const around = e.data.slice(
              Math.max(0, target - 3),
              Math.min(target + 4, e.data.length),
            )

            setTargetArea(around)
            setLoading(false)
          })
          .catch(() => {
            notify('ra.page.error', 'warning')
          })
      },
      500,
      { leading: false, trailing: true },
    ),
  )

  React.useEffect(() => {
    if (validationError || !open) {
      setLoading(false)
      return
    }

    setLoading(true)
    getPreviewData.current?.(to, max, playlistId)
  }, [validationError, to, dataProvider, playlistId, max, open, notify, record])

  const handleClose = (
    event: // Ne easier way to extract the correct type here
    | Parameters<Exclude<ModalProps['onClose'], undefined>>[0]
      | React.MouseEvent<HTMLButtonElement>,
  ) => {
    dispatch(closeMoveToIndexDialog())
    if ('stopPropagation' in event) event.stopPropagation()
  }

  const handleConfirm = (e: React.MouseEvent<HTMLButtonElement>) => {
    if (!record) return

    onSuccess(record.id, to)
    dispatch(closeMoveToIndexDialog())
    e.stopPropagation()
  }

  const fromNum =
    typeof record?.id === 'string' ? parseInt(record.id) : (record?.id ?? -1)
  const toNum = parseInt(to)
  const moveDirection =
    toNum == fromNum ? 'no' : toNum > fromNum ? 'up' : 'down'
  const moveRange = useMemo(
    () => [Math.min(fromNum, toNum), Math.max(fromNum, toNum)],
    [fromNum, toNum],
  )
  const targetIndex = targetArea.findIndex((x) => x.id == to)

  /**
   * @param {number} index relative to the target index
   */
  const getItem = useCallback(
    (index): PreviewItem | null => {
      if (!record) return null

      const goingBackwards = index < 0
      const initialIndexModifier = goingBackwards
        ? moveDirection == 'up'
          ? 1
          : 0
        : -(moveDirection == 'down' ? 1 : 0)

      let resultItem: Record | undefined
      for (
        let i = targetIndex + index + initialIndexModifier;
        goingBackwards ? i >= 0 : i < targetArea.length;
        goingBackwards ? i-- : i++
      ) {
        const elem = targetArea[i]
        if (!elem) return null

        const isTarget = elem.id == record.id
        if (isTarget) continue

        resultItem = elem
        break
      }

      if (!resultItem) return null

      const elemID =
        typeof resultItem.id === 'string'
          ? parseInt(resultItem.id)
          : resultItem.id
      const willMove = elemID >= moveRange[0] && elemID <= moveRange[1]
      let newIndex = elemID
      if (willMove) {
        newIndex = moveDirection == 'up' ? elemID - 1 : elemID + 1
      }

      return {
        elem: resultItem,
        newIndex,
        willMove,
      }
    },
    [record, moveRange, moveDirection, targetArea, targetIndex],
  )

  const itemsToDisplay = useMemo((): (PreviewItem | null)[] => {
    return [
      getItem(-2),
      getItem(-1),
      record ? { newIndex: parseInt(to), willMove: false, elem: record } : null,
      getItem(1),
      getItem(2),
    ]
  }, [getItem, record, to])

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      aria-labelledby="moveToIndex-dialog-song"
      fullWidth={true}
      maxWidth={'sm'}
    >
      <DialogTitle id="moveToIndex-dialog-song">
        {translate(title || 'resources.song.actions.moveToIndex')}
      </DialogTitle>
      <DialogContent>
        <TextField
          value={to}
          onChange={(e) => setTo(e.target.value)}
          helperText={validationError ?? `1 - ${max}`}
          error={!!validationError}
        />
        <List>
          {itemsToDisplay.map((item, i) => {
            return (
              <PreviewListItem
                key={item?.elem.id ?? `index_${i}`}
                loading={loading}
                item={item}
                moveDirection={moveDirection}
                valid={!validationError}
              />
            )
          })}
        </List>
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} color="primary">
          {translate('ra.action.close')}
        </Button>
        <Button onClick={handleConfirm} color="primary">
          {translate('ra.action.confirm')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export default MoveToIndexDialog
