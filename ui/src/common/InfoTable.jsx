import PropTypes from 'prop-types'
import TableCell from '@material-ui/core/TableCell'
import TableContainer from '@material-ui/core/TableContainer'
import { makeStyles } from '@material-ui/core/styles'

// max-width is ignored on table cells, so the label cap lives on an inner wrapper.
const useStyles = makeStyles((theme) => ({
  container: {
    overflowWrap: 'anywhere',
  },
  cell: {
    width: '1%',
  },
  label: {
    width: 'max-content',
    maxWidth: '16em',
    // Not the container's 'anywhere', which lets phone labels shrink to one letter.
    overflowWrap: 'break-word',
    [theme.breakpoints.down('xs')]: {
      width: 'auto',
    },
  },
}))

export const InfoTableContainer = ({ children }) => {
  const classes = useStyles()
  return (
    <TableContainer className={classes.container}>{children}</TableContainer>
  )
}

InfoTableContainer.propTypes = {
  children: PropTypes.node,
}

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
