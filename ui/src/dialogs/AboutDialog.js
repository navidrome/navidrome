import React from 'react'
import PropTypes from 'prop-types'
import Link from '@material-ui/core/Link'
import Dialog from '@material-ui/core/Dialog'
import IconButton from '@material-ui/core/IconButton'
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
import { DialogTitle } from './DialogTitle'
import { DialogContent } from './DialogContent'

const links = {
  homepage: 'navidrome.org',
  reddit: 'reddit.com/r/Navidrome',
  twitter: 'twitter.com/navidrome',
  discord: 'discord.gg/xh7j7yF',
  source: 'github.com/navidrome/navidrome',
  bugReports: 'github.com/navidrome/navidrome/issues/new/choose',
  featureRequests: 'github.com/navidrome/navidrome/discussions/new',
}

const LinkToVersion = ({ version }) => {
  if (version === 'dev') {
    return <TableCell align="left">{version}</TableCell>
  }

  const parts = version.split(' ')
  const commitID = parts[1].replace(/[()]/g, '')
  const isSnapshot = version.includes('SNAPSHOT')
  const url = isSnapshot
    ? `https://github.com/navidrome/navidrome/compare/v${
        parts[0].split('-')[0]
      }...${commitID}`
    : `https://github.com/navidrome/navidrome/releases/tag/v${parts[0]}`
  return (
    <TableCell align="left">
      <Link href={url} target="_blank" rel="noopener noreferrer">
        {parts[0]}
      </Link>
      {' (' + commitID + ')'}
    </TableCell>
  )
}

const AboutDialog = ({ open, onClose }) => {
  const translate = useTranslate()
  return (
    <Dialog onClose={onClose} aria-labelledby="about-dialog-title" open={open}>
      <DialogTitle id="about-dialog-title" onClose={onClose}>
        Navidrome Music Server
      </DialogTitle>
      <DialogContent dividers>
        <TableContainer component={Paper}>
          <Table aria-label={translate('menu.about')} size="small">
            <TableBody>
              <TableRow>
                <TableCell align="right" component="th" scope="row">
                  {translate('menu.version')}:
                </TableCell>
                <LinkToVersion version={config.version} />
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

export { AboutDialog, LinkToVersion }
