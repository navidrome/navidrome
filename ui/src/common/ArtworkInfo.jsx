import React, { useEffect, useState } from 'react'
import PropTypes from 'prop-types'
import { Chip, Link, TableCell, TableRow } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import { useDataProvider, usePermissions, useTranslate } from 'react-admin'
import { DateField } from './DateField'

const OUTCOME_COLORS = {
  hit: '#4caf50',
  miss: '#9e9e9e',
  skipped: '#9e9e9e',
  error: '#f44336',
  unreadable: '#f44336',
}

const useStyles = makeStyles({
  chip: { color: '#fff', height: 20 },
  toggle: { cursor: 'pointer' },
})

const OutcomeChip = ({ outcome }) => {
  const classes = useStyles()
  return (
    <Chip
      size="small"
      label={outcome}
      className={classes.chip}
      style={{ backgroundColor: OUTCOME_COLORS[outcome] || '#9e9e9e' }}
    />
  )
}

OutcomeChip.propTypes = {
  outcome: PropTypes.string.isRequired,
}

const StepTable = ({ title, steps }) => {
  if (!steps?.length) return null
  return (
    <>
      <TableRow>
        <TableCell colSpan={2}>
          <strong>{title}</strong>
        </TableCell>
      </TableRow>
      {steps.map((s, i) => (
        <TableRow key={`${s.candidate}-${i}`}>
          <TableCell>{s.candidate}</TableCell>
          <TableCell>
            <OutcomeChip outcome={s.outcome} /> {s.detail}
          </TableCell>
        </TableRow>
      ))}
    </>
  )
}

StepTable.propTypes = {
  title: PropTypes.string.isRequired,
  steps: PropTypes.array,
}

const Row = ({ label, children }) => (
  <TableRow>
    <TableCell component="th" scope="row">
      {label}:
    </TableCell>
    <TableCell align="left">{children}</TableCell>
  </TableRow>
)

Row.propTypes = {
  label: PropTypes.node.isRequired,
  children: PropTypes.node,
}

export const ArtworkInfo = ({ resource, id }) => {
  const classes = useStyles()
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const { permissions } = usePermissions()
  const [report, setReport] = useState(null)
  const [expanded, setExpanded] = useState(false)
  const isAdmin = permissions === 'admin'

  useEffect(() => {
    if (!isAdmin) return undefined
    let live = true
    dataProvider
      .explainArtwork(resource, id)
      .then(({ data }) => live && setReport(data))
      .catch(() => live && setReport(null))
    return () => {
      live = false
    }
  }, [dataProvider, resource, id, isAdmin])

  if (!isAdmin || !report) return null

  const recorded = !!report.stored
  return (
    <>
      <TableRow>
        <TableCell colSpan={2}>
          <strong>{translate('artwork.title')}</strong>
        </TableCell>
      </TableRow>
      <Row label={translate('artwork.result')}>{report.result}</Row>
      {recorded ? (
        <>
          <Row label={translate('artwork.source')}>{report.stored?.source}</Row>
          <Row label={translate('artwork.attemptedAt')}>
            <DateField record={report.stored} source="attemptedAt" showTime />
          </Row>
        </>
      ) : (
        <Row label={translate('artwork.source')}>
          {translate('artwork.notRecorded')}
        </Row>
      )}
      <TableRow>
        <TableCell colSpan={2}>
          <Link
            component="button"
            className={classes.toggle}
            onClick={() => setExpanded(!expanded)}
          >
            {translate(
              expanded ? 'artwork.hideDetails' : 'artwork.showDetails',
            )}
          </Link>
        </TableCell>
      </TableRow>
      {expanded && (
        <>
          <StepTable title={translate('artwork.chain')} steps={report.steps} />
          <StepTable
            title={translate('artwork.lastAttemptFailed')}
            steps={report.lastAttemptFailed}
          />
          <StepTable
            title={translate('artwork.gaveUpAfter')}
            steps={report.gaveUpAfter}
          />
          {report.stored?.sourcePath && (
            <Row label={translate('artwork.sourcePath')}>
              {report.stored.sourcePath}
            </Row>
          )}
          {report.queued && (
            <>
              <Row label={translate('artwork.priority')}>
                {report.queued.priorityName}
              </Row>
              <Row label={translate('artwork.attempts')}>
                {report.queued.attempts}
              </Row>
              <Row label={translate('artwork.retryAt')}>
                <DateField record={report.queued} source="retryAt" showTime />
              </Row>
            </>
          )}
          {report.config && (
            <Row label={report.config.setting}>{report.config.value}</Row>
          )}
          {report.agents && (
            <Row label={translate('artwork.agents')}>{report.agents}</Row>
          )}
        </>
      )}
    </>
  )
}

ArtworkInfo.propTypes = {
  resource: PropTypes.oneOf(['album', 'artist']).isRequired,
  id: PropTypes.string.isRequired,
}
