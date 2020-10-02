import React from 'react'
import PropTypes from 'prop-types'
import { withStyles } from '@material-ui/core/styles'
import Link from '@material-ui/core/Link'
import Dialog from '@material-ui/core/Dialog'
import MuiDialogTitle from '@material-ui/core/DialogTitle'
import MuiDialogContent from '@material-ui/core/DialogContent'
import IconButton from '@material-ui/core/IconButton'
import CloseIcon from '@material-ui/icons/Close'
import Typography from '@material-ui/core/Typography'
import TableContainer from '@material-ui/core/TableContainer'
import Table from '@material-ui/core/Table'
import TableBody from '@material-ui/core/TableBody'
import TableRow from '@material-ui/core/TableRow'
import TableCell from '@material-ui/core/TableCell'
import Paper from '@material-ui/core/Paper'
import FavoriteBorderIcon from '@material-ui/icons/FavoriteBorder'
import inflection from 'inflection'
import { useTranslate } from 'react-admin'
import config from '../config'

const styles = (theme) => ({
  root: {
    margin: 0,
    padding: theme.spacing(2),
  },
  closeButton: {
    position: 'absolute',
    right: theme.spacing(1),
    top: theme.spacing(1),
    color: theme.palette.grey[500],
  },
})

const links = {
  homepage: 'navidrome.org',
  reddit: 'reddit.com/r/Navidrome',
  twitter: 'twitter.com/navidrome',
  discord: 'discord.gg/xh7j7yF',
  source: 'github.com/deluan/navidrome',
  featureRequests: 'github.com/deluan/navidrome/issues',
}

const DialogTitle = withStyles(styles)((props) => {
  const { children, classes, onClose, ...other } = props
  return (
    <MuiDialogTitle disableTypography className={classes.root} {...other}>
      <Typography variant="h5">{children}</Typography>
      {onClose ? (
        <IconButton
          aria-label="close"
          className={classes.closeButton}
          onClick={onClose}
        >
          <CloseIcon />
        </IconButton>
      ) : null}
    </MuiDialogTitle>
  )
})

const DialogContent = withStyles((theme) => ({
  root: {
    padding: theme.spacing(2),
  },
}))(MuiDialogContent)

const AboutDialog = ({ open, onClose }) => {
  const translate = useTranslate()
  return (
    <Dialog
      onClose={onClose}
      onBackdropClick={onClose}
      aria-labelledby="about-dialog-title"
      open={open}
    >
      <DialogTitle id="about-dialog-title" onClose={onClose}>
        Navidrome Music Server
      </DialogTitle>
      <DialogContent dividers>
        <TableContainer component={Paper}>
          <Table aria-label="song details" size="small">
            <TableBody>
              <TableRow>
                <TableCell align="right" component="th" scope="row">
                  {translate('menu.version')}:
                </TableCell>
                <TableCell align="left">{config.version}</TableCell>
              </TableRow>
              {Object.keys(links).map((key) => {
                return (
                  <TableRow key={key}>
                    <TableCell align="right" component="th" scope="row">
                      {translate(`about.links.${key}`, {
                        _: inflection.humanize(inflection.underscore(key)),
                      })}
                      :
                    </TableCell>
                    <TableCell align="left">
                      <Link
                        href={`https://${links[key]}`}
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        {links[key]}
                      </Link>
                    </TableCell>
                  </TableRow>
                )
              })}
              <TableRow>
                <TableCell align="right" component="th" scope="row">
                  <Link
                    href={'https://github.com/sponsors/deluan'}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    <IconButton size={'small'}>
                      <FavoriteBorderIcon fontSize={'small'} />
                    </IconButton>
                  </Link>
                </TableCell>
                <TableCell align="left">
                  <Link
                    href={'https://ko-fi.com/deluan'}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    ko-fi.com/deluan
                  </Link>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </TableContainer>
      </DialogContent>
    </Dialog>
  )
}

AboutDialog.propTypes = {
  open: PropTypes.bool.isRequired,
  onClose: PropTypes.func.isRequired,
}

export default AboutDialog
