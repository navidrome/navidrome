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
  TextField,
} from '@material-ui/core'
import { closeMoveToIndexDialog } from '../actions'
import debounce from 'lodash.debounce'

/**
 * Calculate the optimal number of page items and the page number
    to include targetIndex and its surrounding items.
 * @param {number} listLength - Total number of items in the list
 * @param {number} targetIndex - Index of the target item
 * @returns {{itemsPerPage: number, pageNumber: number}}
 */
function CalculatePagination(listLength, targetIndex) {
  /**
   * Calculate the optimal number of page items and the page number
   * to include targetIndex and its surrounding items.
   *
   * @param {number} listLength - Total number of items in the list
   * @param {number} targetIndex - Index of the target item (0-based)
   * @return {object} { itemsPerPage, pageNumber }
   */

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

/**
 * @component
 * @param {{
 *  loading?: boolean,
 *  valid?: boolean,
 *  moveDirection: "no" | "up" | "down"
 *  item: {
 *    elem: import('ra-core').Record,
 *    newIndex: number,
 *    willMove: boolean
 *  } | null
 * }}
 */
const PreviewItem = ({ loading, item, valid, moveDirection }) => {
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
 * @component
 * @param {{
 *  title?: string,
 *  onSuccess: (from: string, to: string) => void,
 *  max: number,
 *  playlistId: string
 * }}
 */
const MoveToIndexDialog = ({ title, onSuccess, max, playlistId }) => {
  /**
   * @type {{open: boolean, record: import('ra-core').Record}}
   */
  const { open, record } = useSelector((state) => state.moveToIndexDialog)
  const dispatch = useDispatch()
  const translate = useTranslate()
  /**
   * @type {ReturnType<typeof useState<string>>}
   */
  const [to, setTo] = useState('1')
  /**
   * @type {ReturnType<typeof useState<string>>}
   */
  const [validationError, setValidationError] = useState()

  /**
   * @type {ReturnType<typeof useState<import('ra-core').Record[]>>}
   */
  const [targetArea, setTargetArea] = useState([])
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

  const callback = React.useRef(
    debounce(
      (from, to, max, playlistId) => {
        if (!to || parseInt(to) > max || parseInt < 1) {
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
    callback.current?.(record.id, to, max, playlistId)
  }, [validationError, to, dataProvider, playlistId, max, open, notify, record])

  const handleClose = (e) => {
    dispatch(closeMoveToIndexDialog())
    e.stopPropagation()
  }

  const handleConfirm = (e) => {
    onSuccess(record.id, to)
    dispatch(closeMoveToIndexDialog())
    e.stopPropagation()
  }

  const fromNum = parseInt(record?.id ?? -1)
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
    (index) => {
      if (!record) return null

      const goingBackwards = index < 0
      const initialIndexModifier = goingBackwards
        ? moveDirection == 'up'
          ? 1
          : 0
        : -(moveDirection == 'down' ? 1 : 0)

      let resultItem
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

      const elemID = parseInt(resultItem.id)
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

  const itemsToDisplay = useMemo(() => {
    return [
      getItem(-2),
      getItem(-1),
      { newIndex: to, willMove: false, elem: record },
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
              <PreviewItem
                key={item?.id ?? `index_${i}`}
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

MoveToIndexDialog.propTypes = {
  title: PropTypes.string,
  onSuccess: PropTypes.func.isRequired,
  max: PropTypes.number.isRequired,
  playlistId: PropTypes.string.isRequired,
}

export default MoveToIndexDialog
