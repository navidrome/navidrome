const subsonicUrl = (command, id, options) => {
  const username = localStorage.getItem('username')
  const token = localStorage.getItem('subsonic-token')
  const salt = localStorage.getItem('subsonic-salt')
  const timeStamp = new Date().getTime()
  const url = `rest/${command}?u=${username}&f=json&v=1.8.0&c=NavidromeUI&t=${token}&s=${salt}&id=${id}&_=${timeStamp}`
  if (options) {
    return url + '&' + options
  }
  return url
}

export { subsonicUrl }
