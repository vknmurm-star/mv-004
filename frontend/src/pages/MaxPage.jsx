import { useEffect, useState } from 'react'
import { useApi } from '../utils/api.js'
import MaxButton from '../components/MaxButton.jsx'

export default function MaxPage() {
  const api = useApi()
  const [maxLink, setMaxLink] = useState('')
  const [faq, setFaq] = useState([])
  useEffect(() => {
    api.get('/max/deeplink').then((r) => setMaxLink(r.data?.deeplink || '')).catch(() => {})
    api.get('/faq').then((r) => setFaq((r.data || []).slice(0, 4))).catch(() => {})
  }, [])

  return (
    <section className="section container" style={{ textAlign: 'center' }}>
      <span className="pill">Связь через MAX</span>
      <h1 style={{ marginTop: 16 }}>Напишите нам в MAX</h1>
      <p className="lead" style={{ margin: '8px auto 28px' }}>
        ИИ-ассистент «Машины времени» поможет: подобрать услугу, записаться, ответить по частым вопросам.
        Сложный случай или диагноз — переключим вас на оператора.
      </p>
      <MaxButton href={maxLink} label="Открыть чат в MAX" />

      <div className="grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(240px,1fr))', marginTop: 36, textAlign: 'left' }}>
        {SCENARIOS.map((s) => (
          <div className="card" key={s.title}>
            <div style={{ fontSize: 26 }}>{s.icon}</div>
            <h3 style={{ margin: '10px 0' }}>{s.title}</h3>
            <div className="muted">{s.body}</div>
          </div>
        ))}
      </div>

      <h2 style={{ marginTop: 40 }}>Ответы на частые вопросы</h2>
      <div className="card" style={{ textAlign: 'left', maxWidth: 720, margin: '0 auto' }}>
        {faq.map((f) => (
          <div key={f.id} style={{ padding: '10px 0', borderBottom: '1px solid var(--border)' }}>
            <strong>{f.question}</strong>
            <div className="muted">{f.answer}</div>
          </div>
        ))}
      </div>
    </section>
  )
}

const SCENARIOS = [
  { icon: '📝', title: 'Запись', body: 'Бот соберёт имя, телефон, услугу и симптомы — передаст менеджеру.' },
  { icon: '🧭', title: 'Подбор услуги', body: 'Поможем понять, к какому направлению обратиться по вашим симптомам.' },
  { icon: '❓', title: 'FAQ', body: 'Ответим по часам, гарантиям, оплате и подменному авто.' },
  { icon: '👤', title: 'Оператор', body: 'В любой момент напишите «оператор» — переключим на человека.' },
]