import MaxButton from '../components/MaxButton.jsx'
import { useApi } from '../utils/api.js'
import { useEffect, useState } from 'react'

export default function About() {
  const api = useApi()
  const [maxLink, setMaxLink] = useState('')
  useEffect(() => {
    api.get('/max/deeplink').then((r) => setMaxLink(r.data?.deeplink || '')).catch(() => {})
  }, [])
  return (
    <section className="section container">
      <h1>О компании</h1>
      <div className="lead">
        «Машина времени» — автомастерская, где инженерный опыт встречается с цифровым сервисом.
        Мы убеждены: ремонт должен быть прозрачным, понятным и предсказуемым по цене.
      </div>

      <div className="grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(240px,1fr))' }}>
        {VALUES.map((v) => (
          <div className="card" key={v.title}>
            <h3>{v.title}</h3>
            <div className="muted">{v.body}</div>
          </div>
        ))}
      </div>

      <div className="card" style={{ marginTop: 24, display: 'flex', gap: 24, alignItems: 'center', flexWrap: 'wrap' }}>
        <div style={{ flex: '1 1 320px' }}>
          <h2 style={{ marginTop: 0 }}>Свяжитесь с нами в MAX</h2>
          <p className="muted">ИИ-ассистент поможет сориентироваться, подберёт услугу и зафиксирует заявку. Для сложных случаев переключим на оператора.</p>
        </div>
        <MaxButton href={maxLink} />
      </div>
    </section>
  )
}

const VALUES = [
  { title: 'Прозрачность', body: 'Открытый прайс, согласованная смета до работ, отчёт по каждому этапу.' },
  { title: 'Технологичность', body: 'Диагностические стенды, 3D сход-развал, цифровые записи и напоминания.' },
  { title: 'Человечность', body: 'Объясняем простыми словами, без навязывания лишнего. Честно о сроках.' },
  { title: 'Гарантия результата', body: '6 месяцев на работы и подменное авто для длительного ремонта.' },
]