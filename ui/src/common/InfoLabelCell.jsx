import PropTypes from 'prop-types'
import TableCell from '@material-ui/core/TableCell'
import { makeStyles } from '@material-ui/core/styles'

// max-width is ignored on table cells, so the cap lives on an inner wrapper.
const useStyles = makeStyles((theme) => ({
  cell: {
    width: '1%',
  },
  label: {
    width: 'max-content',
    maxWidth: '16em',
    // Not the dialog's 'anywhere', which lets phone labels shrink to one letter.
    overflowWrap: 'break-word',
    [theme.breakpoints.down('xs')]: {
      width: 'auto',
    },
  },
}))

export const InfoLabelCell = ({ children, ...rest }) => {
  const classes = useStyles()
  return (
    <TableCell scope="row" className={classes.cell} {...rest}>
      <div className={classes.label}>{children}</div>
    </TableCell>
  )
}

InfoLabelCell.propTypes = {
  children: PropTypes.node,
}
