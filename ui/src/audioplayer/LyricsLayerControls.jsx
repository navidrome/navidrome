import Button from '@material-ui/core/Button'
import CircularProgress from '@material-ui/core/CircularProgress'
import IconButton from '@material-ui/core/IconButton'
import Menu from '@material-ui/core/Menu'
import MenuItem from '@material-ui/core/MenuItem'
import Popover from '@material-ui/core/Popover'
import Tooltip from '@material-ui/core/Tooltip'
import Typography from '@material-ui/core/Typography'
import { alpha, makeStyles } from '@material-ui/core/styles'
import AvTimerIcon from '@material-ui/icons/AvTimer'
import GetAppIcon from '@material-ui/icons/GetApp'
import LibraryMusicIcon from '@material-ui/icons/LibraryMusic'
import RecordVoiceOverIcon from '@material-ui/icons/RecordVoiceOver'
import RotateLeftIcon from '@material-ui/icons/RotateLeft'
import TranslateIcon from '@material-ui/icons/Translate'
import WavesIcon from '@material-ui/icons/Waves'
import clsx from 'clsx'
import React, { useCallback, useEffect, useState } from 'react'
import {
  LYRICS_OFFSET_COARSE_MS,
  LYRICS_OFFSET_FINE_MS,
  formatLyricsOffset,
  stepLyricsOffset,
} from './lyrics'
import { formatSourceValue, getVariantLabel } from './lyricsSource'

const noop = () => {}

const syncLabelFor = (sync, labels) => {
  switch (sync) {
    case 'word':
      return labels.wordSynced || 'Word-synced'
    case 'line':
      return labels.lineSynced || 'Line-synced'
    default:
      return labels.plainSynced || 'Plain text'
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
    sourceMenu: {
      minWidth: 210,
      maxWidth: 320,
    },
    sourceHeader: {
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
      gap: theme.spacing(1),
      padding: theme.spacing(1, 2, 0.5),
    },
    sourceHeading: {
      display: 'block',
      color: alpha(theme.palette.text.primary, 0.58),
      lineHeight: 1.4,
    },
    sourceSpinner: {
      color: alpha(theme.palette.text.primary, 0.58),
    },
    sourceItem: {
      display: 'flex',
      justifyContent: 'space-between',
      gap: theme.spacing(2),
    },
    variantLabel: {
      overflowWrap: 'anywhere',
    },
    variantMeta: {
      color: alpha(theme.palette.text.primary, 0.58),
      whiteSpace: 'nowrap',
    },
    offsetPopover: {
      padding: theme.spacing(1, 1.5),
    },
    offsetRow: {
      display: 'flex',
      alignItems: 'center',
      gap: theme.spacing(0.5),
    },
    offsetStep: {
      minWidth: 0,
      padding: theme.spacing(0.25, 0.75),
      color: alpha(theme.palette.text.primary, 0.78),
      whiteSpace: 'nowrap',
    },
    offsetValue: {
      minWidth: 64,
      textAlign: 'center',
      fontVariantNumeric: 'tabular-nums',
      color: theme.palette.text.primary,
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
  letterWave = true,
  translationEnabled,
  pronunciationEnabled,
  letterWaveEnabled = false,
  onToggleTranslation,
  onTogglePronunciation,
  onToggleLetterWave = noop,
  variants = [],
  selectedSourceKey = null,
  onSelectSource = noop,
  onSourceMenuOpenChange = noop,
  refreshing = false,
  offsetMs = 0,
  offsetEnabled = false,
  onOffsetChange = noop,
  onDownload = noop,
  active = true,
  labels = {},
  testId = 'lyrics-layer-controls',
}) => {
  const classes = useStyles()
  const [sourceAnchor, setSourceAnchor] = useState(null)
  const [offsetAnchor, setOffsetAnchor] = useState(null)
  const sourceMenuId = `${testId}-source-menu`
  const offsetPopoverId = `${testId}-offset-popover`
  const hasVariants = variants.length > 0
  const sourceOpen = Boolean(active && hasVariants && sourceAnchor)
  const offsetOpen = Boolean(active && offsetEnabled && offsetAnchor)
  const offsetLabel = labels.offset || 'Lyrics timing offset'

  useEffect(() => {
    if (!sourceOpen) return undefined
    onSourceMenuOpenChange(true)
    return () => onSourceMenuOpenChange(false)
  }, [onSourceMenuOpenChange, sourceOpen])

  useEffect(() => {
    if (!active || !hasVariants) setSourceAnchor(null)
  }, [active, hasVariants])

  useEffect(() => {
    if (!active || !offsetEnabled) setOffsetAnchor(null)
  }, [active, offsetEnabled])

  const selectSource = useCallback(
    (sourceKey) => {
      setSourceAnchor(null)
      onSelectSource(sourceKey)
    },
    [onSelectSource],
  )

  const stepOffset = (delta) =>
    onOffsetChange(stepLyricsOffset(offsetMs, delta))

  return (
    <div
      className={clsx(classes.controls, {
        [classes.mobilePlacement]: placement === 'mobile',
        [classes.sidebarPlacement]: placement !== 'mobile',
      })}
      data-testid={testId}
      onClick={(event) => event.stopPropagation()}
    >
      {hasVariants && (
        <>
          <Tooltip title={labels.viewSource || 'View lyrics source'}>
            <IconButton
              size="small"
              onClick={(event) => setSourceAnchor(event.currentTarget)}
              aria-label={labels.viewSource || 'View lyrics source'}
              aria-haspopup="menu"
              aria-expanded={sourceOpen}
              aria-controls={sourceOpen ? sourceMenuId : undefined}
              data-testid="lyrics-source-button"
              className={clsx(classes.controlButton, {
                [classes.controlActive]: sourceOpen,
              })}
            >
              <LibraryMusicIcon fontSize="small" />
            </IconButton>
          </Tooltip>
          <Menu
            id={sourceMenuId}
            anchorEl={sourceAnchor}
            open={sourceOpen}
            onClose={() => setSourceAnchor(null)}
            anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
            transformOrigin={{ vertical: 'bottom', horizontal: 'right' }}
            PaperProps={{ className: classes.sourceMenu }}
            onClick={(event) => event.stopPropagation()}
          >
            <li className={classes.sourceHeader} role="presentation">
              <Typography variant="overline" className={classes.sourceHeading}>
                {labels.sourceTitle || 'Lyrics source'}
              </Typography>
              {refreshing && (
                <CircularProgress
                  size={12}
                  className={classes.sourceSpinner}
                  aria-label={
                    labels.refreshingSources || 'Looking for more sources'
                  }
                />
              )}
            </li>
            <MenuItem
              role="menuitemradio"
              aria-checked={selectedSourceKey == null}
              selected={selectedSourceKey == null}
              onClick={() => selectSource(null)}
              className={classes.sourceItem}
              data-testid="lyrics-source-auto"
            >
              {labels.autoSource || 'Auto (best match)'}
            </MenuItem>
            {variants.map((variant) => (
              <MenuItem
                key={variant.key}
                role="menuitemradio"
                aria-checked={selectedSourceKey === variant.key}
                selected={selectedSourceKey === variant.key}
                onClick={() => selectSource(variant.key)}
                className={classes.sourceItem}
                data-testid={`lyrics-source-${variant.key}`}
              >
                <span className={classes.variantLabel}>
                  {getVariantLabel(variant.source, labels)}
                </span>
                <span className={classes.variantMeta}>
                  {[
                    formatSourceValue(variant.source?.format),
                    syncLabelFor(variant.sync, labels),
                  ]
                    .filter(Boolean)
                    .join(' · ')}
                </span>
              </MenuItem>
            ))}
          </Menu>
        </>
      )}
      <Tooltip title={offsetLabel}>
        <span>
          <IconButton
            size="small"
            disabled={!offsetEnabled}
            onClick={(event) => setOffsetAnchor(event.currentTarget)}
            aria-label={`${offsetLabel}: ${formatLyricsOffset(offsetMs)}`}
            aria-haspopup="dialog"
            aria-expanded={offsetOpen}
            aria-controls={offsetOpen ? offsetPopoverId : undefined}
            data-testid="lyrics-offset-button"
            className={clsx(classes.controlButton, {
              [classes.controlActive]: offsetOpen || offsetMs !== 0,
            })}
          >
            <AvTimerIcon fontSize="small" />
          </IconButton>
        </span>
      </Tooltip>
      <Popover
        id={offsetPopoverId}
        open={offsetOpen}
        anchorEl={offsetAnchor}
        onClose={() => setOffsetAnchor(null)}
        anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
        transformOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        PaperProps={{ className: classes.offsetPopover }}
        onClick={(event) => event.stopPropagation()}
      >
        <div
          className={classes.offsetRow}
          role="group"
          aria-label={offsetLabel}
        >
          <Button
            size="small"
            aria-label={labels.offsetCoarseEarlier}
            onClick={() => stepOffset(-LYRICS_OFFSET_COARSE_MS)}
            className={classes.offsetStep}
          >
            -0.5 s
          </Button>
          <Button
            size="small"
            aria-label={labels.offsetFineEarlier}
            onClick={() => stepOffset(-LYRICS_OFFSET_FINE_MS)}
            className={classes.offsetStep}
          >
            -0.05 s
          </Button>
          <Typography
            variant="body2"
            component="span"
            className={classes.offsetValue}
          >
            {formatLyricsOffset(offsetMs)}
          </Typography>
          <Button
            size="small"
            aria-label={labels.offsetFineLater}
            onClick={() => stepOffset(LYRICS_OFFSET_FINE_MS)}
            className={classes.offsetStep}
          >
            +0.05 s
          </Button>
          <Button
            size="small"
            aria-label={labels.offsetCoarseLater}
            onClick={() => stepOffset(LYRICS_OFFSET_COARSE_MS)}
            className={classes.offsetStep}
          >
            +0.5 s
          </Button>
          <Tooltip title={labels.offsetReset || 'Reset timing offset'}>
            <span>
              <IconButton
                size="small"
                disabled={offsetMs === 0}
                onClick={() => onOffsetChange(0)}
                aria-label={labels.offsetReset || 'Reset timing offset'}
                data-testid="lyrics-offset-reset"
                className={classes.controlButton}
              >
                <RotateLeftIcon fontSize="small" />
              </IconButton>
            </span>
          </Tooltip>
        </div>
      </Popover>
      <Tooltip title={labels.download || 'Download lyrics'}>
        <span>
          <IconButton
            size="small"
            disabled={!hasVariants}
            onClick={onDownload}
            aria-label={labels.download || 'Download lyrics'}
            data-testid="lyrics-download-button"
            className={classes.controlButton}
          >
            <GetAppIcon fontSize="small" />
          </IconButton>
        </span>
      </Tooltip>
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
        active={letterWave}
        classes={classes}
        disabled={!letterWaveEnabled}
        label={
          letterWave
            ? labels.hideLetterWave || 'Turn off letter wave (experimental)'
            : labels.showLetterWave || 'Turn on letter wave (experimental)'
        }
        onClick={onToggleLetterWave}
        testId="toggle-letter-wave-button"
      >
        <WavesIcon fontSize="small" />
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
