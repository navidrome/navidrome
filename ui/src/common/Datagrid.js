import React from 'react'
import { Datagrid as RADatagrid } from 'react-admin'
import { makeStyles } from '@material-ui/core/styles'

const useStyles = makeStyles({
  root: {
    '& .MuiTableCell-root': {
      overflow: 'hidden',
      whiteSpace: 'nowrap',
      textOverflow: 'ellipsis',
      maxWidth: '10em',
    },
  },
})

export const Datagrid = (props) => {
  const classes = useStyles()
  return <RADatagrid className={classes.root} {...props} />
}
