import baseUrl from './utils/baseUrl'
import throttle from 'lodash.throttle'

// TODO https://stackoverflow.com/a/20060461
let es = null
let dispatchFunc = null

const getEventStream = () => {
  if (es === null) {
    es = new EventSource(
      baseUrl(`/app/api/events?jwt=${localStorage.getItem('token')}`)
    )
  }
  return es
}

export const startEventStream = (func) => {
  const es = getEventStream()
  dispatchFunc = func
  es.onmessage = throttle(
    (msg) => {
      const data = JSON.parse(msg.data)
      if (data.name !== 'keepAlive') {
        dispatchFunc(data)
      }
    },
    100,
    { trailing: true }
  )
}
