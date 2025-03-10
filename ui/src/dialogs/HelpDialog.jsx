import React, { useCallback, useState } from 'react'
import ReactDOM from 'react-dom'
import { Chip, Dialog } from '@material-ui/core'
import { getApplicationKeyMap, GlobalHotKeys } from 'react-hotkeys'
import TableContainer from '@material-ui/core/TableContainer'
import Paper from '@material-ui/core/Paper'
import Table from '@material-ui/core/Table'
import TableBody from '@material-ui/core/TableBody'
import TableRow from '@material-ui/core/TableRow'
import TableCell from '@material-ui/core/TableCell'
import { useTranslate } from 'react-admin'
import { humanize } from 'inflection'
import { keyMap } from '../hotkeys'
import { DialogTitle } from './DialogTitle'
import { DialogContent } from './DialogContent'

const HelpTable = (props) => {
  const keyMap = getApplicationKeyMap()
  const translate = useTranslate()
  return ReactDOM.createPortal(
    <Dialog {...props}>
      <DialogTitle onClose={props.onClose}>
        {translate('help.title')}
      </DialogTitle>
      <DialogContent dividers>
        <TableContainer component={Paper}>
          <Table size="small">
            <TableBody>
              {Object.keys(keyMap).map((key) => {
                const { sequences, name } = keyMap[key]
                const description = translate(`help.hotkeys.${name}`, {
                  _: humanize(name),
                })
                return (
                  <TableRow key={key}>
                    <TableCell align="right" component="th" scope="row">
                      {description}
                    </TableCell>
                    <TableCell align="left">
                      {sequences.map(({ sequence }) => (
                        <Chip
                          label={<kbd>{sequence}</kbd>}
                          size="small"
                          variant={'outlined'}
                          key={sequence}
                        />
                      ))}
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </TableContainer>
      </DialogContent>
    </Dialog>,
    document.body,
  )
}

export const HelpDialog = (props) => {
  const [open, setOpen] = useState(false)

  const handleClickClose = (e) => {
    setOpen(false)
    e.stopPropagation()
  }

  const handlers = {
    SHOW_HELP: useCallback(() => setOpen(true), [setOpen]),
  }

  return (
    <>
      <GlobalHotKeys keyMap={keyMap} handlers={handlers} allowChanges />
      <HelpTable open={open} onClose={handleClickClose} />
    </>
  )
}
