import dayjs from 'dayjs'

/**
 * 格式化日期时间：YYYY-MM-DD HH:mm:ss
 */
export function formatDateTime(date: string | null | undefined): string {
  if (!date) return ''
  const d = dayjs(date)
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm:ss') : String(date)
}

/**
 * 格式化日期时间：YYYY-MM-DD HH:mm
 */
export function formatDate(date: string | null | undefined): string {
  if (!date) return ''
  const d = dayjs(date)
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm') : String(date)
}

/**
 * 格式化日期：YYYY-MM-DD
 */
export function formatDateOnly(date: string | null | undefined): string {
  if (!date) return ''
  const d = dayjs(date)
  return d.isValid() ? d.format('YYYY-MM-DD') : String(date)
}
