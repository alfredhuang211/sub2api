export function formatMoney(amount: number | null | undefined, currency = 'CNY') {
  const value = Number(amount || 0)
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency,
    minimumFractionDigits: 2
  }).format(value)
}

export function formatMinorMoney(amount: number | null | undefined, currency = 'CNY') {
  return formatMoney(Number(amount || 0) / 100, currency)
}

export function formatPercent(rateBps: number | null | undefined) {
  return `${(Number(rateBps || 0) / 100).toFixed(2)}%`
}

export function formatDateTime(value: string | null | undefined) {
  if (!value) return '-'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(new Date(value))
}
