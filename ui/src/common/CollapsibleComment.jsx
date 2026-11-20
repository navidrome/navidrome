import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Typography, Collapse } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import ExpandLess from '@material-ui/icons/ExpandLess'
import ExpandMore from '@material-ui/icons/ExpandMore'
import { useTranslate } from 'react-admin'
import AnchorMe from './Linkify'
import clsx from 'clsx'

const useStyles = makeStyles(
  (theme) => ({
    commentBlock: {
      display: 'inline-block',
      marginTop: '1em',
      float: 'left',
      wordBreak: 'break-word',
      maxWidth: '100%',
    },
    interactive: {
      cursor: 'pointer',
    },
    cueRow: {
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'flex-end',
      gap: theme.spacing(0.25),
      marginTop: theme.spacing(0.5),
      color: theme.palette.text.secondary,
      fontSize: theme.typography.caption.fontSize,
      lineHeight: 1.2,
      userSelect: 'none',
    },
    cueLabel: {
      fontSize: 'inherit',
    },
    cueIcon: {
      fontSize: '1.25rem',
    },
  }),
  {
    name: 'NDCollapsibleComment',
  },
)

const lineHeightPx = (el) => {
  const style = getComputedStyle(el)
  const parsed = parseFloat(style.lineHeight)
  if (!Number.isNaN(parsed)) {
    return parsed
  }
  const fontSize = parseFloat(style.fontSize) || 16
  return fontSize * 1.334
}

export const CollapsibleComment = ({ record }) => {
  const classes = useStyles()
  const translate = useTranslate()
  const [expanded, setExpanded] = useState(false)
  const [overflows, setOverflows] = useState(false)
  const [collapsedHeight, setCollapsedHeight] = useState('1.334em')
  const innerRef = useRef(null)

  const lines = useMemo(
    () => record.comment?.split('\n') || [],
    [record.comment],
  )

  const formatted = useMemo(() => {
    return lines.map((line, idx) => (
      <span key={record.id + '-comment-' + idx}>
        <AnchorMe text={line} />
        <br />
      </span>
    ))
  }, [lines, record.id])

  const checkOverflow = useCallback(() => {
    const el = innerRef.current
    if (!el) return
    const oneLinePx = lineHeightPx(el)
    setCollapsedHeight(`${oneLinePx}px`)
    setOverflows(el.scrollHeight > oneLinePx + 1)
  }, [])

  useEffect(() => {
    checkOverflow()
    const el = innerRef.current
    if (!el) return
    const observer = new ResizeObserver(checkOverflow)
    observer.observe(el)
    return () => observer.disconnect()
  }, [checkOverflow, lines])

  const handleExpandClick = useCallback(() => {
    if (overflows || expanded) {
      setExpanded((prev) => !prev)
    }
  }, [overflows, expanded])

  if (lines.length === 0) {
    return null
  }

  const isInteractive = overflows || expanded

  return (
    <div className={classes.commentBlock} data-testid="collapsible-comment">
      <Collapse
        collapsedHeight={collapsedHeight}
        in={expanded}
        timeout={'auto'}
        className={clsx(isInteractive && classes.interactive)}
        onClick={isInteractive ? handleExpandClick : undefined}
      >
        <Typography variant={'h6'} ref={innerRef}>
          {formatted}
        </Typography>
      </Collapse>
      {overflows && !expanded && (
        <div
          className={clsx(classes.cueRow, classes.interactive)}
          onClick={handleExpandClick}
          data-testid="collapsible-comment-cue"
        >
          <span className={classes.cueLabel}>
            {translate('ra.action.expand')}
          </span>
          <ExpandMore
            className={classes.cueIcon}
            data-testid="collapsible-comment-more"
            fontSize="inherit"
          />
        </div>
      )}
      {overflows && expanded && (
        <div
          className={clsx(classes.cueRow, classes.interactive)}
          onClick={handleExpandClick}
          data-testid="collapsible-comment-cue"
        >
          <ExpandLess
            className={classes.cueIcon}
            data-testid="collapsible-comment-less"
            fontSize="inherit"
          />
        </div>
      )}
    </div>
  )
}
