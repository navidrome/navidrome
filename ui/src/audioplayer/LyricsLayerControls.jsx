import IconButton from '@material-ui/core/IconButton'
import Popover from '@material-ui/core/Popover'
import Tooltip from '@material-ui/core/Tooltip'
import Typography from '@material-ui/core/Typography'
import { alpha, makeStyles } from '@material-ui/core/styles'
import InfoOutlinedIcon from '@material-ui/icons/InfoOutlined'
import RecordVoiceOverIcon from '@material-ui/icons/RecordVoiceOver'
import TranslateIcon from '@material-ui/icons/Translate'
import clsx from 'clsx'
import React, { useEffect, useState } from 'react'

const UPPERCASE_SOURCE_VALUES = new Set([
  'elrc',
  'lrc',
  'srt',
  'ttml',
  'txt',
  'yaml',
])

const formatSourceValue = (value) => {
  const normalized = typeof value === 'string' ? value.trim() : ''
  if (!normalized) return ''
  if (UPPERCASE_SOURCE_VALUES.has(normalized.toLowerCase())) {
    return normalized.toUpperCase()
  }
  if (normalized.toLowerCase() === 'plain') return 'Plain text'
  return normalized
    .replace(/[-_]+/g, ' ')
    .replace(/\b\w/g, (character) => character.toUpperCase())
}

const getSourceName = (source, labels) => {
  switch (source?.type) {
    case 'embedded':
      return labels.embeddedSource || 'Embedded lyrics'
    case 'sidecar':
      return labels.sidecarSource || 'Sidecar file'
    case 'plugin':
      return source.name || labels.pluginSource || 'Lyrics plugin'
    default:
      return source?.name || formatSourceValue(source?.type) || 'Lyrics'
  }
}

const useStyles = makeStyles(
  (theme) => ({
    controls: {
      position: 'absolute',
      bottom: theme.spacing(1),
      zIndex: 2,
      display: 'flex',
      alignItems: 'center',
      gap: theme.spacing(0.25),
      padding: theme.spacing(0.25),
      borderRadius: theme.shape.borderRadius * 2,
      backgroundColor: 'transparent',
    },
    sidebarPlacement: {
      right: theme.spacing(0.75),
    },
    mobilePlacement: {
      left: '50%',
      transform: 'translateX(-50%)',
    },
    controlButton: {
      padding: theme.spacing(0.75),
      color: alpha(theme.palette.text.primary, 0.58),
      backgroundColor: 'transparent',
      WebkitTapHighlightColor: 'transparent',
      transition:
        'color 160ms ease, background-color 160ms ease, transform 160ms cubic-bezier(0.23, 1, 0.32, 1)',
      '@media (hover: hover) and (pointer: fine)': {
        '&:hover': {
          color: theme.palette.text.primary,
          backgroundColor: alpha(theme.palette.primary.main, 0.08),
        },
        '&$controlActive:hover': {
          color: theme.palette.primary.main,
        },
      },
      '&:focus-visible': {
        color: theme.palette.text.primary,
        backgroundColor: alpha(theme.palette.primary.main, 0.1),
      },
      '&:active:not(:disabled)': {
        transform: 'scale(0.97)',
      },
      '&$controlActive': {
        color: theme.palette.primary.main,
      },
      '&$controlActive:focus-visible': {
        color: theme.palette.primary.main,
      },
      '&:disabled': {
        color: alpha(theme.palette.text.primary, 0.28),
      },
      '@media (prefers-reduced-motion: reduce)': {
        transition: 'none',
        '&:active:not(:disabled)': {
          transform: 'none',
        },
      },
    },
    controlActive: {},
    sourcePopover: {
      minWidth: 200,
      maxWidth: 280,
      padding: theme.spacing(1.5, 2),
    },
    sourceHeading: {
      display: 'block',
      color: alpha(theme.palette.text.primary, 0.58),
      lineHeight: 1.4,
    },
    sourceName: {
      marginTop: theme.spacing(0.25),
      color: theme.palette.text.primary,
      overflowWrap: 'anywhere',
    },
    sourceDetails: {
      display: 'grid',
      gridTemplateColumns: 'auto minmax(0, 1fr)',
      columnGap: theme.spacing(1.5),
      rowGap: theme.spacing(0.5),
      marginTop: theme.spacing(1),
    },
    sourceDetailLabel: {
      color: alpha(theme.palette.text.primary, 0.58),
    },
    sourceDetailValue: {
      color: theme.palette.text.primary,
      textAlign: 'right',
      overflowWrap: 'anywhere',
    },
  }),
  { name: 'NDLyricsLayerControls' },
)

const LayerButton = ({
  active,
  classes,
  disabled,
  label,
  onClick,
  testId,
  children,
}) => {
  return (
    <Tooltip title={label}>
      <span>
        <IconButton
          size="small"
          onClick={onClick}
          disabled={disabled}
          aria-label={label}
          aria-pressed={active}
          data-testid={testId}
          className={clsx(classes.controlButton, {
            [classes.controlActive]: active && !disabled,
          })}
        >
          {children}
        </IconButton>
      </span>
    </Tooltip>
  )
}

const LyricsLayerControls = ({
  placement = 'sidebar',
  showTranslation,
  showPronunciation,
  translationEnabled,
  pronunciationEnabled,
  onToggleTranslation,
  onTogglePronunciation,
  source,
  active = true,
  labels = {},
  testId = 'lyrics-layer-controls',
}) => {
  const classes = useStyles()
  const [sourceAnchor, setSourceAnchor] = useState(null)
  const sourcePopoverId = `${testId}-source-popover`
  const sourceOpen = Boolean(active && source && sourceAnchor)
  const sourceDetails = [
    {
      label: labels.provider || 'Provider',
      value: formatSourceValue(source?.provider),
    },
    {
      label: labels.format || 'Format',
      value: formatSourceValue(source?.format),
    },
  ].filter((detail) => detail.value)

  useEffect(() => {
    if (!active || !source) setSourceAnchor(null)
  }, [active, source])

  return (
    <div
      className={clsx(classes.controls, {
        [classes.mobilePlacement]: placement === 'mobile',
        [classes.sidebarPlacement]: placement !== 'mobile',
      })}
      data-testid={testId}
      onClick={(event) => event.stopPropagation()}
    >
      {source && (
        <>
          <Tooltip title={labels.viewSource || 'View lyrics source'}>
            <IconButton
              size="small"
              onClick={(event) => setSourceAnchor(event.currentTarget)}
              aria-label={labels.viewSource || 'View lyrics source'}
              aria-haspopup="dialog"
              aria-expanded={sourceOpen}
              aria-controls={sourceOpen ? sourcePopoverId : undefined}
              data-testid="lyrics-source-button"
              className={clsx(classes.controlButton, {
                [classes.controlActive]: sourceOpen,
              })}
            >
              <InfoOutlinedIcon fontSize="small" />
            </IconButton>
          </Tooltip>
          <Popover
            id={sourcePopoverId}
            open={sourceOpen}
            anchorEl={sourceAnchor}
            onClose={() => setSourceAnchor(null)}
            anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
            transformOrigin={{ vertical: 'bottom', horizontal: 'right' }}
            PaperProps={{ className: classes.sourcePopover }}
          >
            <div
              role="dialog"
              aria-label={labels.sourceTitle || 'Lyrics source'}
              onClick={(event) => event.stopPropagation()}
            >
              <Typography variant="overline" className={classes.sourceHeading}>
                {labels.sourceTitle || 'Lyrics source'}
              </Typography>
              <Typography variant="subtitle2" className={classes.sourceName}>
                {getSourceName(source, labels)}
              </Typography>
              {sourceDetails.length > 0 && (
                <div className={classes.sourceDetails}>
                  {sourceDetails.map((detail) => (
                    <React.Fragment key={detail.label}>
                      <Typography
                        variant="caption"
                        className={classes.sourceDetailLabel}
                      >
                        {detail.label}
                      </Typography>
                      <Typography
                        variant="caption"
                        className={classes.sourceDetailValue}
                      >
                        {detail.value}
                      </Typography>
                    </React.Fragment>
                  ))}
                </div>
              )}
            </div>
          </Popover>
        </>
      )}
      <LayerButton
        active={showPronunciation}
        classes={classes}
        disabled={!pronunciationEnabled}
        label={
          showPronunciation
            ? labels.hidePronunciation || 'Hide pronunciation'
            : labels.showPronunciation || 'Show pronunciation'
        }
        onClick={onTogglePronunciation}
        testId="toggle-pronunciation-button"
      >
        <RecordVoiceOverIcon fontSize="small" />
      </LayerButton>
      <LayerButton
        active={showTranslation}
        classes={classes}
        disabled={!translationEnabled}
        label={
          showTranslation
            ? labels.hideTranslation || 'Hide translation'
            : labels.showTranslation || 'Show translation'
        }
        onClick={onToggleTranslation}
        testId="toggle-translation-button"
      >
        <TranslateIcon fontSize="small" />
      </LayerButton>
    </div>
  )
}

export default LyricsLayerControls
