import { Button, Link } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import PropTypes from 'prop-types'
import React, { useCallback, useMemo, useState } from 'react'

const useStyles = makeStyles(
  (theme) => ({
    link: {
      textDecoration: 'none',
      color: theme.palette.primary.main,
    },
    textWrapper: {
      display: '-webkit-box',
      '-webkit-line-clamp': 3,
      '-webkit-box-orient': 'vertical',
      overflow: 'hidden',
    },
    contentWrapper: {
      position: 'relative',
    },
    viewBtn: {
      whiteSpace: 'nowrap',
      position: 'absolute',
      right: -90,
      bottom: 0,
      padding: '0px 8px',
    },
  }),
  { name: 'RaLink' }
)

const Linkify = ({ text, clampText = false, ...rest }) => {
  const classes = useStyles()
  const [viewDetailedText, setViewDetailedText] = useState(
    clampText ? false : true
  )
  const linkify = useCallback((text) => {
    const urlRegex =
      /(\b(https?|ftp|file):\/\/[-A-Z0-9+&@#/%?=~_|!:,.;]*[-A-Z0-9+&@#/%=~_|])/gi
    return [...text.matchAll(urlRegex)]
  }, [])

  const parse = useCallback(() => {
    const matches = linkify(text)
    if (matches.length === 0) return text

    const elements = []
    let lastIndex = 0
    matches.forEach((match, index) => {
      // Push text located before matched string
      if (match.index > lastIndex) {
        elements.push(text.substring(lastIndex, match.index))
      }

      const href = match[0]
      // Push Link component
      elements.push(
        <Link
          {...rest}
          target="_blank"
          className={classes.link}
          rel="noopener noreferrer"
          key={index}
          href={href}
        >
          {href}
        </Link>
      )

      lastIndex = match.index + href.length
    })

    // Push remaining text
    if (text.length > lastIndex) {
      elements.push(
        <span
          key={'last-span-key'}
          dangerouslySetInnerHTML={{ __html: text.substring(lastIndex) }}
        />
      )
    }

    return elements.length === 1 ? elements[0] : elements
  }, [linkify, text, rest, classes.link])

  const parsedText = useMemo(() => parse(), [parse])

  return (
    <>
      <div className={classes.contentWrapper}>
        <div className={viewDetailedText ? '' : classes.textWrapper}>
          {parsedText}
        </div>
        {clampText && (
          <Button
            className={classes.viewBtn}
            onClick={() => setViewDetailedText((val) => !val)}
          >
            VIEW {viewDetailedText ? 'LESS' : 'MORE'}
          </Button>
        )}
      </div>
    </>
  )
}

Linkify.propTypes = {
  text: PropTypes.string,
  clampText: PropTypes.bool,
}

export default React.memo(Linkify)
