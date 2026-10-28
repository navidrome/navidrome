import React, { useCallback, useState } from 'react'
import ReactDOM from 'react-dom'
import { Chip, Dialog } from '@mui/material'
import { getApplicationKeyMap, GlobalHotKeys } from 'react-hotkeys'
import TableContainer from '@mui/material/TableContainer'
import Paper from '@mui/material/Paper'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableRow from '@mui/material/TableRow'
import TableCell from '@mui/material/TableCell'
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
