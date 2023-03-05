import { useCallback, useState } from 'react'
import { webhooks } from '../config'
import { WebhookDialog } from '../dialogs/WebhookDialog'
import { WebhookToggle } from './WebhookToggle'

export const Webhooks = () => {
  const [links, setLinked] = useState({})

  const updateLinked = useCallback((name, linked) => {
    setLinked((state) => ({
      ...state,
      [name]: linked,
    }))
  }, [])

  const hooks = webhooks.map((hook) => (
    <div key={hook.name}>
      <WebhookToggle
        setLinked={updateLinked}
        linked={links[hook.name]}
        {...hook}
      />
    </div>
  ))

  return (
    <>
      {hooks}
      {<WebhookDialog setLinked={updateLinked} />}
    </>
  )
}
