import { useEffect } from 'react'
import { IconButton, Paper, Typography } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import ChevronLeftIcon from '@material-ui/icons/ChevronLeft'
import ChevronRightIcon from '@material-ui/icons/ChevronRight'
import { useHistory, useLocation } from 'react-router-dom'

const variants = ['A', 'B', 'C']

const useStyles = makeStyles((theme) => ({
  root: {
    position: 'fixed',
    left: '50%',
    bottom: theme.spacing(2),
    zIndex: 1500,
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(0.5),
    padding: theme.spacing(0.5, 1),
    border: `1px solid ${theme.palette.divider}`,
    borderRadius: 999,
    background: theme.palette.background.paper,
    boxShadow: theme.shadows[8],
  },
  label: {
    minWidth: 150,
    textAlign: 'center',
    fontWeight: 700,
    letterSpacing: '0.02em',
  },
}))

export const PrototypeSwitcher = ({ names, variant }) => {
  const classes = useStyles()
  const history = useHistory()
  const location = useLocation()

  const select = (offset) => {
    const index = variants.indexOf(variant)
    const next = variants[(index + offset + variants.length) % variants.length]
    const search = new URLSearchParams(location.search)
    search.set('variant', next)
    history.replace({ ...location, search: search.toString() })
  }

  useEffect(() => {
    const handleKeyDown = (event) => {
      const tag = event.target?.tagName?.toLowerCase()
      if (
        tag === 'input' ||
        tag === 'textarea' ||
        event.target?.isContentEditable
      ) {
        return
      }
      if (event.key === 'ArrowLeft') select(-1)
      if (event.key === 'ArrowRight') select(1)
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  })

  return (
    <Paper className={classes.root} aria-label="UI prototype variants">
      <IconButton
        size="small"
        onClick={() => select(-1)}
        aria-label="Previous variant"
      >
        <ChevronLeftIcon />
      </IconButton>
      <Typography className={classes.label} variant="body2">
        {variant} · {names[variant]}
      </Typography>
      <IconButton
        size="small"
        onClick={() => select(1)}
        aria-label="Next variant"
      >
        <ChevronRightIcon />
      </IconButton>
    </Paper>
  )
}
