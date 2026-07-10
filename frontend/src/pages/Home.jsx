import { Link } from 'react-router-dom'
import { useEffect, useState } from 'react'
import ServicesGrid from '../components/ServicesGrid.jsx'
import Reviews from '../components/Reviews.jsx'
import MaxButton from '../components/MaxButton.jsx'
import { useApi } from '../utils/api.js'

export default function Home() {
  const api = useApi()
  const [featured, setFeatured] = useState([])
  const [maxLink, setMaxLink] = useState('')

  useEffect(() => {
    api.get('/services?limit=6').then((r) => setFeatured(r.data || [])).catch(() => {})
    api.get('/max/deeplink').then((r) => setMaxLink(r.data?.deeplink || '')).catch(() => {})
  }, [])

  return (
    <>
      <section className="hero">
        <div className="container">
          <span className="pill">● Технологичный автосервис в Москве</span>
          <h1>Отмотаем время назад — вернём автомобилю лучшее состояние</h1>
          <p>Прозрачные цены, диагностика под ключ и запись онлайн за минуту. Принимаем заявки здесь и в мессенджере MAX.</p>
          <div className="hero-actions">
            <Link to="/booking" className="btn btn-primary lg">Записаться онлайн</Link>
            <MaxButton href={maxLink} />
            <Link to="/prices" className="btn btn-ghost lg">Смотреть прайс</Link>
          </div>
        </div>
      </section>

      <section className="section container">
        <div className="row-between">
          <div>
            <h2>Популярные услуги</h2>
            <div className="lead">Быстрый взгляд на то, что мы делаем лучше всего. Полный список — на странице услуг и в прайсе.</div>
          </div>
          <Link to="/services" className="btn btn-ghost">Все услуги →</Link>
        </div>
        <ServicesGrid services={featured} loading={featured.length === 0} />
      </section>

      <section className="section container">
        <h2>Почему «Машина времени»</h2>
        <div className="grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(240px,1fr))' }}>
          {ADVANTAGES.map((a) => (
            <div className="card" key={a.title}>
              <div style={{ fontSize: 28 }}>{a.icon}</div>
              <h3 style={{ margin: '12px 0 6px' }}>{a.title}</h3>
              <div className="muted">{a.body}</div>
            </div>
          ))}
        </div>
      </section>

      <section className="section container">
        <h2>Как это работает</h2>
        <div className="grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(220px,1fr))' }}>
          {STEPS.map((s, i) => (
            <div className="card" key={s}>
              <div className="badge-from" style={{ background: 'var(--accent)', color: '#04201d' }}>0{i + 1}</div>
              <div style={{ marginTop: 12 }}>{s}</div>
            </div>
          ))}
        </div>
        <Reviews />
      </section>
    </>
  )
}

const ADVANTAGES = [
  { icon: '🔧', title: 'Опыт мастеров', body: 'Сертифицированные механики с опытом 10+ лет по всем системам автомобиля.' },
  { icon: '📋', title: 'Прозрачные цены', body: 'Прайс в открытом доступе. Смету согласовываем до начала работ.' },
  { icon: '🛡️', title: 'Гарантия 6 мес.', body: 'На все работы — 6 месяцев, на расходники — по гарантии производителя.' },
  { icon: '💬', title: 'Связь в MAX', body: 'ИИ-ассистент поможет записаться и ответит по частым вопросам 24/7.' },
]
const STEPS = [
  'Оставляете заявку на сайте или в MAX — это занимает минуту.',
  'Согласуем дату, время и предварительную смету по телефону.',
  'Проводим диагностику и работы, держим вас в курсе.',
  'Принимаете работу с гарантией. Выдаём подменное авто при надобности.',
]