import React from 'react'
import { Link } from 'react-router-dom'
import { Breadcrumbs as MuiBreadcrumbs, Typography } from '@material-ui/core'
import NavigateNextIcon from '@material-ui/icons/NavigateNext'
import { makeStyles } from '@material-ui/core/styles'
import { useTranslate } from 'react-admin'

const useStyles = makeStyles((theme) => ({
  root: {
    marginBottom: theme.spacing(2),
  },
  link: {
    cursor: 'pointer',
    color: theme.palette.text.secondary,
    textDecoration: 'none',
    '&:hover': {
      textDecoration: 'underline',
    },
  },
  current: {
    color: theme.palette.text.primary,
  },
}))

const Breadcrumbs = ({ breadcrumbs }) => {
  const classes = useStyles()
  const translate = useTranslate()

  if (!breadcrumbs || !Array.isArray(breadcrumbs)) return null

  return (
    <MuiBreadcrumbs
      separator={<NavigateNextIcon fontSize="small" />}
      aria-label="breadcrumb"
      className={classes.root}
    >
      <Link to="/folder" className={classes.link}>
        {translate('menu.folders')}
      </Link>
      {breadcrumbs.map((b, index) => {
        if (!b || !b.id || !b.name) return null
        const isLast = index === breadcrumbs.length - 1
        const isLibrary = index === 0

        if (isLast) {
          return (
            <Typography key={b.id} className={classes.current}>
              {b.name}
            </Typography>
          )
        }

        // If it's the library, just go back to the main list
        const url = isLibrary ? '/folder' : `/folder/${b.id}/show`

        return (
          <Link key={b.id} to={url} className={classes.link}>
            {b.name}
          </Link>
        )
      })}
    </MuiBreadcrumbs>
  )
}

export default Breadcrumbs
