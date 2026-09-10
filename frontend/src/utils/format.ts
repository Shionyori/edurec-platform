const pad = (n: number) => String(n).padStart(2, '0')

export function formatDate(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/** Unix 秒 → YYYY-MM-DD。metadata 里的时间戳字段（如 B 站视频发布时间）用它渲染 */
export function formatUnixDate(seconds: number): string {
  const d = new Date(seconds * 1000)
  if (Number.isNaN(d.getTime())) return String(seconds)
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}
