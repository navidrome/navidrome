import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Typography, Collapse } from '@material-ui/core'
import { makeStyles, alpha } from '@material-ui/core/styles'
import ExpandMore from '@material-ui/icons/ExpandMore'
import AnchorMe from './Linkify'
import clsx from 'clsx'

const COLLAPSED_HEIGHT_EM = 2

const useStyles = makeStyles(
  (theme) => {
    const fadeTo = theme.palette.background.paper
    return {
      commentBlock: {
        position: 'relative',
        display: 'inline-block',
        marginTop: '1em',
        float: 'left',
        wordBreak: 'break-word',
        maxWidth: '100%',
      },
      pointerCursor: {
        cursor: 'pointer',
      },
      overflowCue: {
        position: 'absolute',
        left: 0,
        right: 0,
        bottom: 0,
        height: '1em',
        pointerEvents: 'none',
        display: 'flex',
        alignItems: 'flex-end',
        justifyContent: 'center',
        zIndex: 1,
      },
      fadeGradient: {
        position: 'absolute',
        left: 0,
        right: 0,
        top: 0,
        bottom: 0,
        background: `linear-gradient(to bottom, ${alpha(
          fadeTo,
          0,
        )} 0%, ${alpha(fadeTo, 0.75)} 45%, ${fadeTo} 100%)`,
      },
      expandIcon: {
        position: 'relative',
        color: theme.palette.text.secondary,
        fontSize: '1.25rem',
        marginBottom: -2,
        filter: `drop-shadow(0 0 2px ${alpha(fadeTo, 0.9)})`,
      },
    }
  },
  {
    name: 'NDCollapsibleComment',
  },
)

export const CollapsibleComment = ({ record }) => {
  const classes = useStyles()
  const [expanded, setExpanded] = useState(false)
  const [overflows, setOverflows] = useState(false)
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
    const fontSize = parseFloat(getComputedStyle(el).fontSize) || 16
    const collapsedPx = COLLAPSED_HEIGHT_EM * fontSize
    setOverflows(el.scrollHeight > collapsedPx)
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

  const showOverflowCue = !expanded && overflows

  return (
    <div
      className={clsx(
        classes.commentBlock,
        (overflows || expanded) && classes.pointerCursor,
      )}
      onClick={handleExpandClick}
      data-testid="collapsible-comment"
    >
      <Collapse
        collapsedHeight={`${COLLAPSED_HEIGHT_EM}em`}
        in={expanded}
        timeout={'auto'}
      >
        <Typography variant={'h6'} ref={innerRef}>
          {formatted}
        </Typography>
      </Collapse>
      {showOverflowCue && (
        <div className={classes.overflowCue} aria-hidden="true">
          <div
            className={classes.fadeGradient}
            data-testid="collapsible-comment-fade"
          />
          <ExpandMore
            className={classes.expandIcon}
            data-testid="collapsible-comment-more"
          />
        </div>
      )}
    </div>
  )
}
