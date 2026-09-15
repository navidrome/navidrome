import IconButton from '@material-ui/core/IconButton'
import Tooltip from '@material-ui/core/Tooltip'
import { alpha, makeStyles } from '@material-ui/core/styles'
import RecordVoiceOverIcon from '@material-ui/icons/RecordVoiceOver'
import TranslateIcon from '@material-ui/icons/Translate'
import clsx from 'clsx'
import React from 'react'

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
  labels = {},
  testId = 'lyrics-layer-controls',
}) => {
  const classes = useStyles()

  return (
    <div
      className={clsx(classes.controls, {
        [classes.mobilePlacement]: placement === 'mobile',
        [classes.sidebarPlacement]: placement !== 'mobile',
      })}
      data-testid={testId}
      onClick={(event) => event.stopPropagation()}
    >
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
