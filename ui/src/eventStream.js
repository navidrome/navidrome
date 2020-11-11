import { baseUrl } from './utils'
import throttle from 'lodash.throttle'

let es = null
let onMessageHandler = null
let timeOut = null

const getEventStream = () => {
  if (es === null) {
    es = new EventSource(
      baseUrl(`/app/api/events?jwt=${localStorage.getItem('token')}`)
    )
  }
  return es
}

// Reestablish the event stream after 20 secs of inactivity
const setTimeout = () => {
  if (timeOut != null) {
    window.clearTimeout(timeOut)
  }
  timeOut = window.setTimeout(() => {
    if (es != null) {
      es.close()
    }
    es = null
    startEventStream(onMessageHandler)
  }, 20000)
}

export const startEventStream = (messageHandler) => {
  onMessageHandler = messageHandler
  setTimeout()
  if (!localStorage.getItem('token')) {
    console.log('Cannot create a unauthenticated EventSource')
    return
  }
  const es = getEventStream()
  es.onmessage = throttle(
    (msg) => {
      const data = JSON.parse(msg.data)
      if (data.name !== 'keepAlive') {
        onMessageHandler(data)
      }
      setTimeout() // Reset timeout on every received message
    },
    100,
    { trailing: true }
  )
}
