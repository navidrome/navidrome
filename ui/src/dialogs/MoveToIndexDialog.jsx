import React, { useState } from 'react'
import PropTypes from 'prop-types'
import { useDispatch, useSelector } from 'react-redux'
import { useDataProvider, useNotify, useTranslate } from 'react-admin'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  List,
  ListItem,
  TextField,
} from '@material-ui/core'
import { closeMoveToIndexDialog } from '../actions'

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
      throw new Error("targetIndex must be within the range of listLength.");
  }

  // Minimum page size to include targetIndex and 2 items before and after
  const minStart = Math.max(1, targetIndex - 2);
  const maxEnd = Math.min(listLength - 1, targetIndex + 2);

  // Items per page should cover the range [minStart, maxEnd]
  const itemsPerPage = maxEnd - minStart + 1;

  // Calculate the page number (1-based) assuming the page starts at minStart
  // Assuming each page's items are sequential
  const pageNumber = Math.floor(minStart / itemsPerPage) + 1;

  return { itemsPerPage, pageNumber };
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
  const [to, setTo] = useState("1");
  /**
   * @type {ReturnType<typeof useState<string>>}
   */
  const [validationError, setValidationError] = useState();

  /**
   * @type {ReturnType<typeof useState<import('ra-core').Record[]>>}
   */
  const [targetArea, setTargetArea] = useState([]);
  const notify = useNotify()

  const dataProvider = useDataProvider();

  React.useEffect(() => {
    if (!to) {
      setValidationError(translate("ra.validation.required"));
      return;
    }

    const value = parseInt(to);
    if (Number.isNaN(value)) {
      setValidationError(translate("ra.validation.number"));
      return;
    }

    if (value < 1) {
      setValidationError(translate("ra.validation.minValue", { min: 0 }));
      return;
    }

    if (value > max) {
      setValidationError(translate("ra.validation.maxValue", { max: max}));
      return;
    }

    setValidationError(undefined);
  }, [to, max, translate]);

  React.useEffect(() => {
    if (validationError || !open)
      return;

    // FIXME: algorithm is not providing the correct amount of items above and below, fix it
    const { itemsPerPage, pageNumber } = CalculatePagination(max, parseInt(to))

    // TODO: stop interfering with the playlist page 
    dataProvider.getList('playlistTrack', {
      pagination: { page: pageNumber, perPage: itemsPerPage },
      sort: { field: 'id', order: 'ASC' },
      filter: { playlist_id: playlistId },
    }).then(e => {
      // TODO: Only show 3 above and 3 below
      setTargetArea(e.data);
    }).catch(() => {
      notify('ra.page.error', 'warning')
    })
  }, [validationError, to, dataProvider, playlistId, max, open, notify]);


  const handleClose = (e) => {
    dispatch(closeMoveToIndexDialog())
    e.stopPropagation()
  }

  const handleConfirm = (e) => {
    onSuccess(record.id, to)
    dispatch(closeMoveToIndexDialog())
    e.stopPropagation()
  }

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
        {(!validationError, targetArea) &&
          <List>
            {targetArea.map((x => {
              if (!to || (record.mediaFileId == x.mediaFileId && record.id != to))
                return null;

              const fromNum = parseInt(record.id);
              const toNum = parseInt(to);
              const current = parseInt(x.id);

              const movingUp = toNum > fromNum;
              const changeRange = [Math.min(fromNum, toNum), Math.max(fromNum, toNum)];

              const willMove = current >= changeRange[0] && current <= changeRange[1];
              const isTarget = current == toNum;

              let newIndex = current;
              if (willMove) {
                newIndex = movingUp ? current - 1 : current + 1;
              }

              const Target = () => {
                if (!isTarget)
                  return null;

                return (
                  <ListItem
                      disableGutters
                      selected
                  >
                    {toNum} - {record.title} - {record.album} - {record.artist}

                  </ListItem>
                )
              }

              return (
                <React.Fragment key={x.id}>
                  {newIndex == record.id && (
                    <Divider />
                  )}
                  {!movingUp && <Target />}
                  {willMove 
                  ?
                    <ListItem 
                      disableGutters
                    >
                      {newIndex}{movingUp ? "↑" : "↓"} - {x.title} - {x.album} - {x.artist}
                    </ListItem>
                  : 
                    <ListItem 
                      disableGutters
                    >
                      {x.id} - {x.title} - {x.album} - {x.artist}
                    </ListItem> 
                  }
                  {movingUp && <Target />}
                </React.Fragment>
              )
            }))}
          </List>
        }
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
  playlistId: PropTypes.string.isRequired
}

export default MoveToIndexDialog
