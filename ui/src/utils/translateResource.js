import inflection from 'inflection'

export const translatedResourceName = (resource, translate) => {
  if (resource && translate) {
    return translate(`resources.${resource.name}.name`, {
      smart_count: 2,
      _:
        resource.options && resource.options.label
          ? translate(resource.options.label, {
              smart_count: 2,
              _: resource.options.label,
            })
          : inflection.humanize(inflection.pluralize(resource.name)),
    })
  }
}
