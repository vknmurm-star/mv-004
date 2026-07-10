import MaxButton from '../components/MaxButton.jsx'
import { useApi } from '../utils/api.js'
import { useEffect, useState } from 'react'

export default function Contacts() {
  const api = useApi()
  const [maxLink, setMaxLink] = useState('')
  useEffect(() => {
    api.get('/max/deeplink').then((r) => setMaxLink(r.data?.deeplink || '')).catch(() => {})
  }, [])
  return (
    <section className="section container">
      <h1>Контакты</h1>
      <div className="grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(280px,1fr))' }}>
        <div className="card">
          <h3>Адрес</h3>
          <p>Москва, ул. Гаражная 12</p>
          <h3>Часы работы</h3>
          <p>Пн–Сб 09:00–20:00<br />Воскресенье — выходной</p>
          <h3>Телефон</h3>
          <p>+7 (495) 000-00-00</p>
          <div style={{ marginTop: 12 }}><MaxButton href={maxLink} /></div>
        </div>
        <div className="card" style={{ minHeight: 360, padding: 0, overflow: 'hidden' }}>
          <iframe
            title="Карта"
            style={{ width: '100%', height: '100%', minHeight: 360, border: 0, filter: 'grayscale(.2) invert(.08)' }}
            loading="lazy"
            referrerPolicy="no-referrer-when-downgrade"
            src="https://www.openstreetmap.org/export/embed.html?bbox=37.5%2C55.7%2C37.7%2C55.8&layer=mapnik"
          />
        </div>
      </div>
    </section>
  )
}