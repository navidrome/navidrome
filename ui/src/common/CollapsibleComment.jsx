import { useCallback, useMemo, useState } from 'react'
import { Typography, Collapse } from '@mui/material'
import { useTheme } from '@mui/material/styles'
import AnchorMe from './Linkify'
import clsx from 'clsx'

// TODO(v6): migrate makeStyles → sx prop / styled()
// const useStyles = makeStyles(
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
  }),
  {
    name: 'NDCollapsibleComment',
  },
)

export const CollapsibleComment = ({ record }) => {
  const classes = useStyles()
  const [expanded, setExpanded] = useState(false)

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

  const handleExpandClick = useCallback(() => {
    setExpanded(!expanded)
  }, [expanded, setExpanded])

  if (lines.length === 0) {
    return null
  }

  return (
    <Collapse
      collapsedHeight={'2em'}
      in={expanded}
      timeout={'auto'}
      className={clsx(
        classes.commentBlock,
        lines.length > 1 && classes.pointerCursor,
      )}
    >
      <Typography variant={'h6'} onClick={handleExpandClick}>
        {formatted}
      </Typography>
    </Collapse>
  )
}
