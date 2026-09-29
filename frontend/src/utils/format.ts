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

/** 分钟数 → 人类可读时长（90 → “1 小时 30 分”）；非正数返回空串 */
export function formatDuration(minutes: number): string {
  if (!minutes || minutes <= 0) return ''
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  if (h === 0) return `${m} 分钟`
  if (m === 0) return `${h} 小时`
  return `${h} 小时 ${m} 分`
}

const DIFFICULTY_LABELS: Record<string, string> = {
  beginner: '入门',
  intermediate: '进阶',
  advanced: '高阶',
}

/** 难度英文枚举 → 中文标签；未知值原样返回 */
export function difficultyLabel(value: string): string {
  return DIFFICULTY_LABELS[value] ?? value
}
