const format = (opt: Intl.DateTimeFormatOptions) => new Intl.DateTimeFormat('da-DK', opt)

function isValidDate(d: Date): boolean {
  return !isNaN(d.getTime()) && d.getFullYear() > 1970
}

export function ddddhhmm(date: Date | string | number) {
  const d = new Date(date)
  if (!isValidDate(d)) return ''
  return format({ weekday: 'long', hour: '2-digit', minute: '2-digit' }).format(d)
}
export function dddhhmm(date: Date | string | number) {
  const d = new Date(date)
  if (!isValidDate(d)) return ''
  return format({ weekday: 'short', hour: '2-digit', minute: '2-digit' }).format(d)
}
export function hhmm(date: Date | string | number) {
  const d = new Date(date)
  if (!isValidDate(d)) return ''
  return format({ hour: '2-digit', minute: '2-digit' }).format(d)
}
