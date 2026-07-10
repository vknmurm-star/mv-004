import { createContext, useContext, useState } from 'react'

const AuthCtx = createContext({ token: null, setToken: () => {}, clear: () => {} })

const KEY = 'tm-admin-token'

export function TokenProvider({ children }) {
  const [token, setTokenState] = useState(() => localStorage.getItem(KEY))

  const setToken = (t) => {
    if (t) localStorage.setItem(KEY, t)
    else localStorage.removeItem(KEY)
    setTokenState(t)
  }
  const clear = () => setToken(null)

  return <AuthCtx.Provider value={{ token, setToken, clear }}>{children}</AuthCtx.Provider>
}

export const useAuth = () => useContext(AuthCtx)