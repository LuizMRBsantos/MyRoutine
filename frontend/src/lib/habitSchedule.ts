const WEEKDAY = ['', 'seg', 'ter', 'qua', 'qui', 'sex', 'sáb', 'dom']

// "todo dia", "seg a sex", "ter, qui e sáb"…
export function scheduleLabel(days: number[]): string {
  const d = [...new Set(days)].sort((a, b) => a - b)
  if (d.length === 7) return 'todo dia'
  if (d.join() === '1,2,3,4,5') return 'seg a sex'
  if (d.join() === '6,7') return 'fim de semana'
  const names = d.map(n => WEEKDAY[n])
  return names.length > 1 ? `${names.slice(0, -1).join(', ')} e ${names[names.length - 1]}` : names.join('')
}
