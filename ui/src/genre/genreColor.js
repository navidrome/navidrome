// Deterministic two-tone gradient per genre name, so the same genre always renders the same way
// for every user, with no artwork or backend data needed - just the name.
export const genreGradient = (name) => {
  let hash = 0
  for (let i = 0; i < name.length; i++) {
    hash = (hash * 31 + name.charCodeAt(i)) >>> 0
  }
  const hue1 = hash % 360
  const hue2 = (hue1 + 40) % 360
  return `linear-gradient(135deg, hsl(${hue1}, 65%, 45%), hsl(${hue2}, 70%, 32%))`
}
