import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useApi } from '../utils/api.js'

function Wizard({ services, cats }) {
  const api = useApi()
  const [step, setStep] = useState(1)
  const [done, setDone] = useState(false)
  const [err, setErr] = useState('')
  const [form, setForm] = useState({
    name: '', phone: '', car_make: '', car_model: '', gov_number: '', vin: '',
    service_id: '', desired_at: '', comment: ''
  })

  const set = (k, v) => setForm((f) => ({ ...f, [k]: v }))

  const submit = async () => {
    setErr('')
    try {
      await api.post('/appointments', {
        ...form,
        service_id: form.service_id ? Number(form.service_id) : null,
        desired_at: form.desired_at ? new Date(form.desired_at).toISOString() : null
      })
      setDone(true)
    } catch (e) {
      setErr(e.message)
    }
  }

  if (done) {
    return (
      <div className="card" style={{ textAlign: 'center', padding: 40 }}>
        <div style={{ fontSize: 48 }}>✅</div>
        <h2>Заявка принята!</h2>
        <p className="muted">Менеджер свяжется с вами в течение часа в рабочее время для подтверждения даты и времени.</p>
        <Link to="/" className="btn btn-primary">На главную</Link>
      </div>
    )
  }

  return (
    <div className="card" style={{ maxWidth: 620, margin: '0 auto' }}>
      <div className="row-between" style={{ marginBottom: 18 }}>
        {[1, 2, 3].map((n) => (
          <div key={n} style={{ flex: 1, height: 6, borderRadius: 6, marginRight: 8,
            background: step >= n ? 'var(--accent)' : 'var(--border)' }} />
        ))}
      </div>
      {err && <div className="error-banner">{err}</div>}

      {step === 1 && (
        <>
          <h2 style={{ marginTop: 0 }}>Контакт и автомобиль</h2>
          <div className="row-between">
            <div className="field" style={{ flex: 1 }}>
              <label>Имя *</label>
              <input value={form.name} onChange={(e) => set('name', e.target.value)} />
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>Телефон *</label>
              <input value={form.phone} onChange={(e) => set('phone', e.target.value)} placeholder="+7…" />
            </div>
          </div>
          <div className="row-between">
            <div className="field" style={{ flex: 1 }}>
              <label>Марка *</label>
              <input value={form.car_make} onChange={(e) => set('car_make', e.target.value)} placeholder="Toyota" />
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>Модель *</label>
              <input value={form.car_model} onChange={(e) => set('car_model', e.target.value)} placeholder="Corolla" />
            </div>
          </div>
          <div className="row-between">
            <div className="field" style={{ flex: 1 }}>
              <label>Госномер</label>
              <input value={form.gov_number} onChange={(e) => set('gov_number', e.target.value)} />
            </div>
            <div className="field" style={{ flex: 1 }}>
              <label>VIN (17 символов)</label>
              <input value={form.vin} onChange={(e) => set('vin', e.target.value)} />
              <div className="hint">Необязательно, но ускорит работу.</div>
            </div>
          </div>
          <button className="btn btn-primary" onClick={() => setStep(2)}>Далее</button>
        </>
      )}

      {step === 2 && (
        <>
          <h2 style={{ marginTop: 0 }}>Услуга и время</h2>
          <div className="field">
            <label>Категория / услуга</label>
            <select value={form.service_id} onChange={(e) => set('service_id', e.target.value)}>
              <option value="">— не выбрано —</option>
              {cats.map((c) => (
                <optgroup key={c.id} label={c.name}>
                  {services.filter((s) => s.category_id === c.id).map((s) => (
                    <option key={s.id} value={s.id}>{s.name}</option>
                  ))}
                </optgroup>
              ))}
            </select>
          </div>
          <div className="field">
            <label>Желаемая дата/время</label>
            <input type="datetime-local" value={form.desired_at} onChange={(e) => set('desired_at', e.target.value)} />
          </div>
          <div className="row-between">
            <button className="btn btn-ghost" onClick={() => setStep(1)}>Назад</button>
            <button className="btn btn-primary" onClick={() => setStep(3)}>Далее</button>
          </div>
        </>
      )}

      {step === 3 && (
        <>
          <h2 style={{ marginTop: 0 }}>Комментарий и отправка</h2>
          <div className="field">
            <label>Комментарий</label>
            <textarea value={form.comment} onChange={(e) => set('comment', e.target.value)}
              placeholder="Опишите симптомы, что беспокоит…" />
          </div>
          <div className="row-between">
            <button className="btn btn-ghost" onClick={() => setStep(2)}>Назад</button>
            <button className="btn btn-primary lg" onClick={submit}>Отправить заявку</button>
          </div>
          <div className="hint">Нажимая «Отправить», вы соглашаетесь на обработку персональных данных.</div>
        </>
      )}
    </div>
  )
}

export default function Booking() {
  const api = useApi()
  const [services, setServices] = useState([])
  const [cats, setCats] = useState([])

  useEffect(() => {
    api.get('/categories').then((r) => setCats(r.data || [])).catch(() => {})
    api.get('/services?limit=500').then((r) => setServices(r.data || [])).catch(() => {})
  }, [])

  return (
    <section className="section container">
      <h1 style={{ textAlign: 'center' }}>Запись на обслуживание</h1>
      <div className="lead" style={{ textAlign: 'center', margin: '8px auto 32px' }}>
        Заполните три шага — это займёт меньше минуты. Подтверждение придёт в MAX или по телефону.
      </div>
      <Wizard services={services} cats={cats} />
    </section>
  )
}