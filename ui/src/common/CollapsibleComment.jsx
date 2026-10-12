import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Typography, Collapse } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import AnchorMe from './Linkify'
import clsx from 'clsx'

const COLLAPSED_HEIGHT_EM = 2

const useStyles = makeStyles(
  (theme) => ({
    commentBlock: {
      display: 'inline-block',
      marginTop: '1em',
      float: 'left',
      wordBreak: 'break-word',
    },
    pointerCursor: {
      cursor: 'pointer',
    },
    fadeOverlay: {
      position: 'relative',
      '&::after': {
        content: '""',
        position: 'absolute',
        bottom: 0,
        left: 0,
        right: 0,
        height: '1.2em',
        background: `linear-gradient(transparent, ${theme.palette.background.default})`,
        pointerEvents: 'none',
      },
    },
  }),
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

  return (
    <Collapse
      collapsedHeight={`${COLLAPSED_HEIGHT_EM}em`}
      in={expanded}
      timeout={'auto'}
      className={clsx(
        classes.commentBlock,
        (overflows || expanded) && classes.pointerCursor,
        !expanded && overflows && classes.fadeOverlay,
      )}
    >
      <Typography variant={'h6'} onClick={handleExpandClick} ref={innerRef}>
        {formatted}
      </Typography>
    </Collapse>
  )
}
