import React, { useEffect, useState } from 'react'
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
import FileCopyIcon from '@material-ui/icons/FileCopy'
import Button from '@material-ui/core/Button'
import { humanize, underscore } from 'inflection'
import { useGetOne, usePermissions, useTranslate, useNotify } from 'react-admin'
import { Tabs, Tab } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import config from '../config'
import { DialogTitle } from './DialogTitle'
import { DialogContent } from './DialogContent'
import { INSIGHTS_DOC_URL } from '../consts.js'
import subsonic from '../subsonic/index.js'
import { Typography } from '@material-ui/core'
import TableHead from '@material-ui/core/TableHead'
import { configToToml, separateAndSortConfigs } from './aboutUtils'

const useStyles = makeStyles((theme) => ({
  configNameColumn: {
    maxWidth: '200px',
    width: '200px',
    wordWrap: 'break-word',
    overflowWrap: 'break-word',
  },
  envVarColumn: {
    maxWidth: '250px',
    width: '250px',
    fontFamily: 'monospace',
    wordWrap: 'break-word',
    overflowWrap: 'break-word',
  },
  copyButton: {
    marginBottom: theme.spacing(2),
    marginTop: theme.spacing(1),
  },
  devSectionHeader: {
    '& td': {
      paddingTop: theme.spacing(2),
      paddingBottom: theme.spacing(2),
      borderTop: `2px solid ${theme.palette.divider}`,
      borderBottom: `1px solid ${theme.palette.divider}`,
      textAlign: 'left',
      fontWeight: 600,
    },
  },
  configContainer: {
    paddingTop: theme.spacing(1),
  },
  tableContainer: {
    maxHeight: '60vh',
    overflow: 'auto',
  },
  devFlagsTitle: {
    fontWeight: 600,
  },
  expandableDialog: {
    transition: 'max-width 300ms ease',
  },
}))

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
    return <>{version}</>
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
    <>
      <Link href={url} target="_blank" rel="noopener noreferrer">
        {parts[0]}
      </Link>
      {' (' + commitID + ')'}
    </>
  )
}

const ShowVersion = ({ uiVersion, serverVersion }) => {
  const translate = useTranslate()
  const showRefresh = uiVersion !== serverVersion

  return (
    <>
      <TableRow>
        <TableCell align="right" component="th" scope="row">
          {translate('menu.version')}:
        </TableCell>
        <TableCell align="left">
          <LinkToVersion version={serverVersion} />
        </TableCell>
      </TableRow>
      {showRefresh && (
        <TableRow>
          <TableCell align="right" component="th" scope="row">
            UI {translate('menu.version')}:
          </TableCell>
          <TableCell align="left">
            <div>
              <LinkToVersion version={uiVersion} />
            </div>
            <div>
              <Link onClick={() => window.location.reload()}>
                <Typography variant={'caption'}>
                  {translate('ra.notification.new_version')}
                </Typography>
              </Link>
            </div>
          </TableCell>
        </TableRow>
      )}
    </>
  )
}

const AboutTabContent = ({
  uiVersion,
  serverVersion,
  insightsData,
  loading,
  permissions,
}) => {
  const translate = useTranslate()

  const lastRun = !loading && insightsData?.lastRun
  let insightsStatus = 'N/A'
  if (lastRun === 'disabled') {
    insightsStatus = translate('about.links.insights.disabled')
  } else if (lastRun && lastRun?.startsWith('1969-12-31')) {
    insightsStatus = translate('about.links.insights.waiting')
  } else if (lastRun) {
    insightsStatus = lastRun
  }

  return (
    <Table aria-label={translate('menu.about')} size="small">
      <TableBody>
        <ShowVersion uiVersion={uiVersion} serverVersion={serverVersion} />
        {Object.keys(links).map((key) => {
          return (
            <TableRow key={key}>
              <TableCell align="right" component="th" scope="row">
                {translate(`about.links.${key}`, {
                  _: humanize(underscore(key)),
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
        {permissions === 'admin' ? (
          <TableRow>
            <TableCell align="right" component="th" scope="row">
              {translate(`about.links.lastInsightsCollection`)}:
            </TableCell>
            <TableCell align="left">
              <Link href={INSIGHTS_DOC_URL}>{insightsStatus}</Link>
            </TableCell>
          </TableRow>
        ) : null}
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
  )
}

const ConfigTabContent = ({ configData }) => {
  const classes = useStyles()
  const translate = useTranslate()
  const notify = useNotify()

  if (!configData || !configData.config) {
    return null
  }

  // Use the shared separation and sorting logic
  const { regularConfigs, devConfigs } = separateAndSortConfigs(
    configData.config,
  )

  const handleCopyToml = async () => {
    try {
      const tomlContent = configToToml(configData, translate)
      await navigator.clipboard.writeText(tomlContent)
      notify(translate('about.config.exportSuccess'), 'info')
    } catch (err) {
      // eslint-disable-next-line no-console
      console.error('Failed to copy TOML:', err)
      notify(translate('about.config.exportFailed'), 'error')
    }
  }

  return (
    <div className={classes.configContainer}>
      <Button
        variant="outlined"
        startIcon={<FileCopyIcon />}
        onClick={handleCopyToml}
        className={classes.copyButton}
        disabled={!configData}
        size="small"
      >
        {translate('about.config.exportToml')}
      </Button>
      <TableContainer className={classes.tableContainer}>
        <Table size="small" stickyHeader>
          <TableHead>
            <TableRow>
              <TableCell
                align="left"
                component="th"
                scope="col"
                className={classes.configNameColumn}
              >
                {translate('about.config.configName')}
              </TableCell>
              <TableCell align="left" component="th" scope="col">
                {translate('about.config.environmentVariable')}
              </TableCell>
              <TableCell align="left" component="th" scope="col">
                {translate('about.config.currentValue')}
              </TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {configData?.configFile && (
              <TableRow>
                <TableCell
                  align="left"
                  component="th"
                  scope="row"
                  className={classes.configNameColumn}
                >
                  {translate('about.config.configurationFile')}
                </TableCell>
                <TableCell align="left" className={classes.envVarColumn}>
                  ND_CONFIGFILE
                </TableCell>
                <TableCell align="left">{configData.configFile}</TableCell>
              </TableRow>
            )}
            {regularConfigs.map(({ key, envVar, value }) => (
              <TableRow key={key}>
                <TableCell
                  align="left"
                  component="th"
                  scope="row"
                  className={classes.configNameColumn}
                >
                  {key}
                </TableCell>
                <TableCell align="left" className={classes.envVarColumn}>
                  {envVar}
                </TableCell>
                <TableCell align="left">{String(value)}</TableCell>
              </TableRow>
            ))}
            {devConfigs.length > 0 && (
              <TableRow className={classes.devSectionHeader}>
                <TableCell colSpan={3}>
                  <Typography
                    variant="subtitle1"
                    component="div"
                    className={classes.devFlagsTitle}
                  >
                    🚧 {translate('about.config.devFlagsHeader')}
                  </Typography>
                </TableCell>
              </TableRow>
            )}
            {devConfigs.map(({ key, envVar, value }) => (
              <TableRow key={key}>
                <TableCell
                  align="left"
                  component="th"
                  scope="row"
                  className={classes.configNameColumn}
                >
                  {key}
                </TableCell>
                <TableCell align="left" className={classes.envVarColumn}>
                  {envVar}
                </TableCell>
                <TableCell align="left">{String(value)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </div>
  )
}

const TabContent = ({
  tab,
  setTab,
  showConfigTab,
  uiVersion,
  serverVersion,
  insightsData,
  loading,
  permissions,
  configData,
}) => {
  const translate = useTranslate()

  return (
    <TableContainer component={Paper}>
      {showConfigTab && (
        <Tabs value={tab} onChange={(_, value) => setTab(value)}>
          <Tab
            label={translate('about.tabs.about')}
            id="about-tab"
            aria-controls="about-panel"
          />
          <Tab
            label={translate('about.tabs.config')}
            id="config-tab"
            aria-controls="config-panel"
          />
        </Tabs>
      )}
      <div
        id="about-panel"
        role="tabpanel"
        aria-labelledby="about-tab"
        hidden={showConfigTab && tab === 1}
      >
        <AboutTabContent
          uiVersion={uiVersion}
          serverVersion={serverVersion}
          insightsData={insightsData}
          loading={loading}
          permissions={permissions}
        />
      </div>
      {showConfigTab && (
        <div
          id="config-panel"
          role="tabpanel"
          aria-labelledby="config-tab"
          hidden={tab === 0}
        >
          <ConfigTabContent configData={configData} />
        </div>
      )}
    </TableContainer>
  )
}

const AboutDialog = ({ open, onClose }) => {
  const classes = useStyles()
  const { permissions } = usePermissions()
  const { data: insightsData, loading } = useGetOne(
    'insights',
    'insights_status',
  )
  const [serverVersion, setServerVersion] = useState('')
  const showConfigTab = permissions === 'admin' && config.devUIShowConfig
  const [tab, setTab] = useState(0)
  const { data: configData } = useGetOne('config', 'config', {
    enabled: showConfigTab,
  })
  const expanded = showConfigTab && tab === 1
  const uiVersion = config.version

  useEffect(() => {
    subsonic
      .ping()
      .then((resp) => resp.json['subsonic-response'])
      .then((data) => {
        if (data.status === 'ok') {
          setServerVersion(data.serverVersion)
        }
      })
      .catch((e) => {
        // eslint-disable-next-line no-console
        console.error('error pinging server', e)
      })
  }, [setServerVersion])

  return (
    <Dialog
      onClose={onClose}
      aria-labelledby="about-dialog-title"
      open={open}
      fullWidth={true}
      maxWidth={expanded ? 'lg' : 'sm'}
      className={classes.expandableDialog}
    >
      <DialogTitle id="about-dialog-title" onClose={onClose}>
        Navidrome Music Server
      </DialogTitle>
      <DialogContent dividers>
        <TabContent
          tab={tab}
          setTab={setTab}
          showConfigTab={showConfigTab}
          uiVersion={uiVersion}
          serverVersion={serverVersion}
          insightsData={insightsData}
          loading={loading}
          permissions={permissions}
          configData={configData}
        />
      </DialogContent>
    </Dialog>
  )
}

AboutDialog.propTypes = {
  open: PropTypes.bool.isRequired,
  onClose: PropTypes.func.isRequired,
}

export { AboutDialog, LinkToVersion }
