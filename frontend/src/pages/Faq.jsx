import { useEffect, useState } from 'react'
import { useApi } from '../utils/api.js'

export default function Faq() {
  const api = useApi()
  const [items, setItems] = useState([])
  const [open, setOpen] = useState(null)
  useEffect(() => {
    api.get('/faq').then((r) => setItems(r.data || [])).catch(() => {})
  }, [])
  return (
    <section className="section container">
      <h1>Часто задаваемые вопросы</h1>
      <div className="lead">Не нашли ответ? Напишите нам в MAX — поможем.</div>
      <div className="grid">
        {items.map((it, i) => (
          <div className="card" key={it.id} style={{ cursor: 'pointer' }} onClick={() => setOpen(open === i ? null : i)}>
            <div className="row-between">
              <strong>{it.question}</strong>
              <span className="muted">{open === i ? '−' : '+'}</span>
            </div>
            {open === i && <p style={{ marginTop: 10 }}>{it.answer}</p>}
          </div>
        ))}
      </div>
    </section>
  )
}