import { baseUrl } from './utils'
import throttle from 'lodash.throttle'
import { processEvent, serverDown } from './actions'

let es = null
let dispatch = null
let timeout = null
const defaultIntervalCheck = 20000
const reconnectIntervalCheck = 2000
let currentIntervalCheck = reconnectIntervalCheck

const getEventStream = () => {
  if (es === null) {
    es = new EventSource(
      baseUrl(`/app/api/events?jwt=${localStorage.getItem('token')}`)
    )
  }
  return es
}

// Reestablish the event stream after 20 secs of inactivity
const setTimeout = (value) => {
  currentIntervalCheck = value
  if (timeout != null) {
    window.clearTimeout(timeout)
  }
  timeout = window.setTimeout(() => {
    if (es != null) {
      es.close()
    }
    es = null
    startEventStream(dispatch)
  }, currentIntervalCheck)
}

export const startEventStream = (dispatchFunc) => {
  dispatch = dispatchFunc
  setTimeout(currentIntervalCheck)
  if (!localStorage.getItem('token')) {
    console.log('Cannot create a unauthenticated EventSource connection')
    return
  }
  const es = getEventStream()
  es.onmessage = throttle(
    (msg) => {
      const data = JSON.parse(msg.data)
      if (data.name !== 'keepAlive') {
        dispatch(processEvent(data))
      }
      setTimeout(defaultIntervalCheck) // Reset timeout on every received message
    },
    100,
    { trailing: true }
  )
  es.onerror = (e) => {
    setTimeout(reconnectIntervalCheck)
    dispatch(serverDown())
  }

  return es
}
