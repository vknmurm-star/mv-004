export default function formatPrice(price_cents, currency = 'RUB', is_from = false) {
  switch (currency) {
    case 'USD': return `${is_from ? 'от ' : ''}$${(price_cents / 100).toFixed(0)}`
    case 'EUR': return `${is_from ? 'от ' : ''}€${(price_cents / 100).toFixed(0)}`
    default: return `${is_from ? 'от ' : ''}${(price_cents / 100).toLocaleString('ru-RU')} ₽`
  }
}

export function formatDuration(min) {
  if (!min) return ''
  if (min < 60) return `${min} мин`
  const h = Math.floor(min / 60)
  const m = min % 60
  return m ? `${h} ч ${m} мин` : `${h} ч`
}