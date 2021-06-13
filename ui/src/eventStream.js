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
  if (!es) {
    // Call `keepalive` to refresh the jwt token
    await httpClient(`${REST_URL}/keepalive/keepalive`)
    let url = baseUrl(`${REST_URL}/events`)
    if (localStorage.getItem('token')) {
      url = url + `?jwt=${localStorage.getItem('token')}`
    }
    es = new EventSource(url)
  }
  return es
}

// Reestablish the event stream after 20 secs of inactivity
const setTimeout = (value) => {
  currentIntervalCheck = value
  if (timeout) {
    window.clearTimeout(timeout)
  }
  timeout = window.setTimeout(async () => {
    if (es) {
      es.close()
    }
    es = null
    await startEventStream()
  }, currentIntervalCheck)
}

const stopEventStream = () => {
  if (es) {
    es.close()
  }
  es = null
  if (timeout) {
    window.clearTimeout(timeout)
  }
  timeout = null
}

const setDispatch = (dispatchFunc) => {
  dispatch = dispatchFunc
}

const eventHandler = (event) => {
  const data = JSON.parse(event.data)
  if (event.type !== 'keepAlive') {
    dispatch(processEvent(event.type, data))
  }
  setTimeout(defaultIntervalCheck) // Reset timeout on every received message
}

const throttledEventHandler = throttle(eventHandler, 100, { trailing: true })

const startEventStream = async () => {
  setTimeout(currentIntervalCheck)
  if (!localStorage.getItem('is-authenticated')) {
    console.log('Cannot create a unauthenticated EventSource connection')
    return Promise.reject()
  }
  return getEventStream()
    .then((newStream) => {
      newStream.addEventListener('serverStart', eventHandler)
      newStream.addEventListener('scanStatus', throttledEventHandler)
      newStream.addEventListener('refreshResource', eventHandler)
      newStream.addEventListener('keepAlive', eventHandler)
      newStream.onerror = (e) => {
        console.log('EventStream error', e)
        setTimeout(reconnectIntervalCheck)
        dispatch(serverDown())
      }
      return newStream
    })
    .catch((e) => {
      console.log(`Error connecting to server:`, e)
    })
}

export { setDispatch, startEventStream, stopEventStream }
