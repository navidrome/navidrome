function getParts(object, fragments) {
  if (!object) {
    return object
  }
  const [part, ...rest] = fragments.split('.')

  return Object.assign(
    {},
    ...Object.entries(object)
      .filter(([key]) => key.toLowerCase().includes(part))
      .map(([k, v]) => {
        if (!rest.length) return { [k]: v }
        const parts = v && typeof v === 'object' && getParts(v, rest.join('.'))
        if (parts) return { [k]: parts }
        return undefined
      })
  )
}

const pruneResource = (resource) => ({
  props: {},
  list: {
    param: {
      perPage: getParts(resource.list.param, 'perPage'),
      filter: {},
    },
    selectedIds: [],
  },
})

export const pruneResources = (state) => {
  return Object.keys(state.admin.resources).reduce(
    (acc, cur) => ({
      ...acc,
      [cur]: pruneResource(state.admin.resources[cur]),
    }),
    {}
  )
}
