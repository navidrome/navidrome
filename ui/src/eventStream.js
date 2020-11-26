import { baseUrl } from './utils'
import throttle from 'lodash.throttle'
import { processEvent, serverDown } from './actions'
import { httpClient } from './dataProvider'
import { REST_URL } from './consts'

const defaultIntervalCheck = 20000
const reconnectIntervalCheck = 2000
let currentIntervalCheck = reconnectIntervalCheck
let es = null
let dispatch = null
let timeout = null

const getEventStream = async () => {
  if (es === null) {
    return httpClient(`${REST_URL}/keepalive/eventSource`).then(() => {
      es = new EventSource(
        baseUrl(`/app/api/events?jwt=${localStorage.getItem('token')}`)
      )
      return es
    })
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

const stopEventStream = () => {
  if (es) {
    es.close()
  }
  es = null
  if (timeout != null) {
    window.clearTimeout(timeout)
  }
  timeout = null
}

const setDispatch = (dispatchFunc) => {
  dispatch = dispatchFunc
}

const eventHandler = throttle(
  (event) => {
    if (event.type !== 'keepAlive') {
      dispatch(processEvent(event.type, event.data))
    }
    setTimeout(defaultIntervalCheck) // Reset timeout on every received message
  },
  100,
  { trailing: true }
)

const startEventStream = async () => {
  setTimeout(currentIntervalCheck)
  if (!localStorage.getItem('token')) {
    console.log('Cannot create a unauthenticated EventSource connection')
    return Promise.reject()
  }
  getEventStream().then((newStream) => {
    newStream.addEventListener('serverStart', eventHandler)
    newStream.addEventListener('scanStatus', eventHandler)
    newStream.addEventListener('keepAlive', eventHandler)
    newStream.onerror = (e) => {
      console.log('EventStream error', e)
      setTimeout(reconnectIntervalCheck)
      dispatch(serverDown())
    }
    es = newStream
    return es
  })
}

export { setDispatch, startEventStream, stopEventStream }
