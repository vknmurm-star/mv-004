import { Routes, Route, Link, useLocation } from 'react-router-dom'
import { useEffect, useState } from 'react'
import Home from './pages/Home.jsx'
import Services from './pages/Services.jsx'
import Prices from './pages/Prices.jsx'
import Booking from './pages/Booking.jsx'
import About from './pages/About.jsx'
import Faq from './pages/Faq.jsx'
import Contacts from './pages/Contacts.jsx'
import MaxPage from './pages/MaxPage.jsx'
import AdminLogin from './pages/admin/AdminLogin.jsx'
import AdminLayout from './pages/admin/AdminLayout.jsx'
import Dashboard from './pages/admin/Dashboard.jsx'
import AdminServices from './pages/admin/AdminServices.jsx'
import PriceImport from './pages/admin/PriceImport.jsx'
import AdminAppointments from './pages/admin/AdminAppointments.jsx'
import AdminLeads from './pages/admin/AdminLeads.jsx'
import AdminFaq from './pages/admin/AdminFaq.jsx'
import AdminKnowledge from './pages/admin/AdminKnowledge.jsx'
import AdminReviews from './pages/admin/AdminReviews.jsx'
import AdminAudit from './pages/admin/AdminAudit.jsx'
import { ThemeProvider } from './utils/theme.jsx'
import { TokenProvider } from './utils/auth.jsx'

function PublicLayout({ children }) {
  return (
    <>
      <header className="site-header">
        <div className="container nav">
          <Link to="/" className="brand">
            <span className="brand-mark">🕒</span>
            <span className="brand-name">Машина времени</span>
          </Link>
          <nav className="nav-links">
            <Link to="/services">Услуги</Link>
            <Link to="/prices">Прайс</Link>
            <Link to="/booking">Запись</Link>
            <Link to="/faq">FAQ</Link>
            <Link to="/about">О нас</Link>
            <Link to="/contacts">Контакты</Link>
            <Link to="/max" className="btn btn-primary sm">MAX</Link>
          </nav>
        </div>
      </header>
      <main className="site-main">{children}</main>
      <footer className="site-footer">
        <div className="container">
          <div>© {new Date().getFullYear()} Автомастерская «Машина времени»</div>
          <div>Москва, ул. Гаражная 12 · Пн–Сб 09:00–20:00</div>
        </div>
      </footer>
    </>
  )
}

function ScrollToTop() {
  const { pathname } = useLocation()
  useEffect(() => { window.scrollTo(0, 0) }, [pathname])
  return null
}

export default function App() {
  return (
    <ThemeProvider>
      <TokenProvider>
        <ScrollToTop />
        <Routes>
          <Route path="/" element={<PublicLayout><Home /></PublicLayout>} />
          <Route path="/services" element={<PublicLayout><Services /></PublicLayout>} />
          <Route path="/prices" element={<PublicLayout><Prices /></PublicLayout>} />
          <Route path="/booking" element={<PublicLayout><Booking /></PublicLayout>} />
          <Route path="/about" element={<PublicLayout><About /></PublicLayout>} />
          <Route path="/faq" element={<PublicLayout><Faq /></PublicLayout>} />
          <Route path="/contacts" element={<PublicLayout><Contacts /></PublicLayout>} />
          <Route path="/max" element={<PublicLayout><MaxPage /></PublicLayout>} />

          <Route path="/admin/login" element={<AdminLogin />} />
          <Route path="/admin" element={<AdminLayout />}>
            <Route index element={<Dashboard />} />
            <Route path="services" element={<AdminServices />} />
            <Route path="prices/import" element={<PriceImport />} />
            <Route path="appointments" element={<AdminAppointments />} />
            <Route path="leads" element={<AdminLeads />} />
            <Route path="faq" element={<AdminFaq />} />
            <Route path="knowledge" element={<AdminKnowledge />} />
            <Route path="reviews" element={<AdminReviews />} />
            <Route path="audit" element={<AdminAudit />} />
          </Route>
        </Routes>
      </TokenProvider>
    </ThemeProvider>
  )
}