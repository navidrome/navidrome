import { useCallback, useMemo, useState } from 'react'
import { Typography, Collapse } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import AnchorMe from './Linkify'
import clsx from 'clsx'

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
  }),
  {
    name: 'NDCollapsibleComment',
  },
)

export const CollapsibleComment = ({ record }) => {
  const classes = useStyles()
  const [expanded, setExpanded] = useState(false)

  const lines = record.comment.split('\n')
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
