import { useAuth } from './auth.jsx'

const BASE = '/api/v1'

async function request(path, { method = 'GET', body, formData, token, raw } = {}) {
  const headers = {}
  if (token) headers.Authorization = `Bearer ${token}`
  let payload
  if (body) {
    headers['Content-Type'] = 'application/json'
    payload = JSON.stringify(body)
  } else if (formData) {
    payload = formData // browser sets multipart boundary
  }
  const res = await fetch(BASE + path, { method, headers, body: payload })
  if (res.status === 204) return null
  const ct = res.headers.get('content-type') || ''
  if (raw || ct.includes('text/csv')) return res
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    const msg = data?.error?.message || `Ошибка ${res.status}`
    throw new Error(msg)
  }
  return data
}

export function useApi() {
  const { token } = useAuth()
  return {
    get: (p, o = {}) => request(p, { ...o, token }),
    post: (p, body, o = {}) => request(p, { method: 'POST', ...(body instanceof FormData ? { formData: body } : { body }), ...o, token }),
    put: (p, body, o = {}) => request(p, { method: 'PUT', body, ...o, token }),
    patch: (p, body, o = {}) => request(p, { method: 'PATCH', body, ...o, token }),
    del: (p, o = {}) => request(p, { method: 'DELETE', ...o, token }),
    token,
    download: async (p, filename) => {
      const res = await request(p, { raw: true })
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      a.click()
      URL.revokeObjectURL(url)
    }
  }
}