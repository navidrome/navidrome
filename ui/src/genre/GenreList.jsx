import { Filter, Loading, SearchInput, useTranslate } from 'react-admin'
import { useHistory } from 'react-router-dom'
import { makeStyles } from '@material-ui/core/styles'
import { List } from '../common'
import { genreGradient } from './genreColor'

const useStyles = makeStyles((theme) => ({
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))',
    gap: '1rem',
    padding: '0.5rem 1rem 1rem',
  },
  chip: {
    borderRadius: '12px',
    padding: '1.25rem',
    color: '#fff',
    cursor: 'pointer',
    minHeight: '96px',
    display: 'flex',
    flexDirection: 'column',
    justifyContent: 'space-between',
    boxShadow: '0 2px 6px rgba(0,0,0,0.25)',
    transition: 'transform 150ms ease, box-shadow 150ms ease',
    outline: 'none',
    '&:hover, &:focus-visible': {
      transform: 'translateY(-2px)',
      boxShadow: '0 4px 12px rgba(0,0,0,0.35)',
    },
  },
  name: {
    fontSize: '1.15rem',
    fontWeight: 600,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
    textShadow: '0 1px 2px rgba(0,0,0,0.35)',
  },
  counts: {
    fontSize: '0.85rem',
    opacity: 0.92,
    textShadow: '0 1px 2px rgba(0,0,0,0.35)',
  },
}))

const GenreFilter = (props) => (
  <Filter {...props} variant={'outlined'}>
    <SearchInput id="search" source="name" alwaysOn />
  </Filter>
)

const GenreChipGrid = ({ ids, data, loading }) => {
  const classes = useStyles()
  const history = useHistory()
  const translate = useTranslate()

  if (loading || !ids || !data) return <Loading />

  const goToGenre = (id) => history.push(`/genre/${id}/show`)

  return (
    <div className={classes.grid}>
      {ids.map((id) => {
        const genre = data[id]
        if (!genre) return null
        return (
          <div
            key={id}
            className={classes.chip}
            style={{ background: genreGradient(genre.name) }}
            onClick={() => goToGenre(id)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') goToGenre(id)
            }}
            role="button"
            tabIndex={0}
          >
            <div className={classes.name}>{genre.name}</div>
            <div className={classes.counts}>
              {translate('resources.genre.chipCounts', {
                songs: genre.songCount || 0,
                albums: genre.albumCount || 0,
              })}
            </div>
          </div>
        )
      })}
    </div>
  )
}

const GenreList = (props) => (
  <List
    {...props}
    sort={{ field: 'name', order: 'ASC' }}
    exporter={false}
    bulkActionButtons={false}
    filters={<GenreFilter />}
    perPage={50}
  >
    <GenreChipGrid />
  </List>
)

export default GenreList
