export const FormatFullDate = (date) => {
  const dashes = date.split("-").length - 1
  let options = {
    year: "numeric"
  }
  switch(dashes) {
    case 2:
      options = {
        year: "numeric",
        month: "long",
        day: "numeric"
      }
      return new Date(date).toLocaleDateString(undefined, options)
    case 1:
      options = {
        year: "numeric",
        month: "long"
      }
      return new Date(date).toLocaleDateString(undefined, options)
    case 0:
      if (date.length === 4) {
        return new Date(date).toLocaleDateString(undefined, options)
      } else {
        return ''
      }
    default:
      return ''
  }
}
