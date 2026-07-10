import { useRef, useState } from 'react'
import { useApi } from '../../utils/api.js'

const ACTION_CLASS = { create: 'row-create', update: 'row-update', skip: 'row-skip', error: 'row-error' }

export default function PriceImport() {
  const api = useApi()
  const inputRef = useRef(null)
  const [drag, setDrag] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [result, setResult] = useState(null)
  const [jobs, setJobs] = useState([])
  const [err, setErr] = useState('')
  const [applying, setApplying] = useState(false)
  const [applied, setApplied] = useState(null)

  async function handleFile(file) {
    setResult(null); setErr(''); setApplied(null)
    if (!file) return
    setUploading(true)
    try {
      const fd = new FormData()
      fd.append('file', file)
      const r = await api.post('/prices/import', fd)
      setResult(r); loadJobs()
    } catch (e) {
      setErr(e.message)
    } finally {
      setUploading(false)
    }
  }

  function loadJobs() {
    api.get('/prices/jobs').then((r) => setJobs(r.data || [])).catch(() => {})
  }

  async function apply(jobId) {
    setApplying(true); setApplied(null)
    try {
      const r = await api.post(`/prices/jobs/${jobId}/apply`)
      setApplied({ jobId, summary: r.data.summary })
      loadJobs()
    } catch (e) {
      setErr(e.message)
    } finally {
      setApplying(false)
    }
  }

  return (
    <>
      <div className="admin-top">
        <div>
          <h1 style={{ margin: 0 }}>Импорт прайса</h1>
          <div className="muted">CSV или XLSX. Сначала валидация и предпросмотр, затем подтверждение.</div>
        </div>
        <div className="row-between">
          <button className="btn" onClick={() => api.download('/prices/template', 'timemachine-price-template.csv')}>⬇ Шаблон CSV</button>
          <button className="btn" onClick={() => api.download('/prices/export', 'timemachine-prices.csv')}>⬇ Экспорт текущих</button>
        </div>
      </div>

      {err && <div className="error-banner">{err}</div>}
      {applied && (
        <div className="ok-banner">
          Применено: добавлено {applied.summary.added}, обновлено {applied.summary.updated}, пропущено {applied.summary.skipped}, ошибок {applied.summary.errors}.
        </div>
      )}

      {!result && (
        <div
          className={`dropzone ${drag ? 'drag' : ''}`}
          onDragOver={(e) => { e.preventDefault(); setDrag(true) }}
          onDragLeave={() => setDrag(false)}
          onDrop={(e) => { e.preventDefault(); setDrag(false); handleFile(e.dataTransfer.files[0]) }}
          onClick={() => inputRef.current?.click()}
        >
          <input ref={inputRef} type="file" accept=".csv,.xlsx" hidden onChange={(e) => handleFile(e.target.files[0])} />
          {uploading ? <><div className="spinner" style={{ margin: '0 auto 12px' }} /><p>Разбираем файл…</p></>
            : <>
              <p style={{ fontSize: 40 }}>📥</p>
              <p><strong>Перетащите файл сюда</strong> или нажмите для выбора</p>
              <p>Поддерживаются .csv и .xlsx · макс. 5 МБ</p>
            </>}
        </div>
      )}

      {result && (
        <div className="card">
          <div className="row-between">
            <h3 style={{ margin: 0 }}>Предпросмотр · {result.meta.file_name}</h3>
            <button className="btn btn-ghost sm" onClick={() => setResult(null)}>Сбросить</button>
          </div>
          <div className="stat-grid" style={{ margin: '14px 0' }}>
            <div className="stat"><div className="num">{result.meta.summary.total}</div><div className="lbl">Всего строк</div></div>
            <div className="stat"><div className="num" style={{ color: 'var(--ok)' }}>{result.meta.summary.added}</div><div className="lbl">Будет создано</div></div>
            <div className="stat"><div className="num" style={{ color: 'var(--accent-2)' }}>{result.meta.summary.updated}</div><div className="lbl">Будет обновлено</div></div>
            <div className="stat"><div className="num" className="muted">{result.meta.summary.skipped}</div><div className="lbl">Пропущено</div></div>
            <div className="stat"><div className="num" style={{ color: 'var(--danger)' }}>{result.meta.summary.errors}</div><div className="lbl">С ошибками</div></div>
          </div>

          <div className="table-wrap">
            <table className="data">
              <thead><tr><th>#</th><th>Действие</th><th>Название</th><th>Цена</th><th>Ошибки</th></tr></thead>
              <tbody>
                {(result.meta.preview_rows || []).map((r) => (
                  <tr key={r.row_number} className={ACTION_CLASS[r.action] || ''}>
                    <td>{r.row_number}</td>
                    <td><span className="tag">{r.action}</span></td>
                    <td>{r.name}</td>
                    <td>{(r.price_cents / 100).toLocaleString('ru-RU')}</td>
                    <td style={{ color: 'var(--danger)' }}>
                      {(r.errors || []).map((e) => e.message).join('; ')}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {(result.meta.preview_rows?.length < result.meta.row_count) && (
            <div className="hint">Показаны первые {result.meta.preview_rows.length} из {result.meta.row_count} строк. Остальные сохранены в job.</div>
          )}

          <div className="row-between" style={{ marginTop: 16 }}>
            <span className="muted">Job #{result.data.job_id} · статус «validated»</span>
            <button
              className="btn btn-primary lg"
              disabled={applying || result.meta.summary.total === 0}
              onClick={() => apply(result.data.job_id)}
            >
              {applying ? 'Применяем…' : 'Подтвердить и применить'}
            </button>
          </div>
        </div>
      )}

      <div className="card" style={{ marginTop: 18 }}>
        <h3>Журнал импортов</h3>
        <button className="btn sm" onClick={loadJobs} style={{ marginBottom: 10 }}>Обновить</button>
        {jobs.length > 0 && (
          <div className="table-wrap">
            <table className="data">
              <thead><tr><th>#</th><th>Файл</th><th>Статус</th><th>Всего</th><th>Добавлено</th><th>Обновлено</th><th>Пропущено</th><th>Дата</th></tr></thead>
              <tbody>
                {jobs.map((j) => (
                  <tr key={j.id}>
                    <td>{j.id}</td><td>{j.file_name}</td><td><span className="tag">{j.status}</span></td>
                    <td>{j.total}</td><td>{j.added}</td><td>{j.updated}</td><td>{j.skipped}</td>
                    <td className="muted">{j.created_at}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </>
  )
}