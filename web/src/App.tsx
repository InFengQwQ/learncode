import { Routes, Route } from 'react-router-dom'
import Layout from './components/Layout'
import HomePage from './pages/HomePage'
import LanguagesPage from './pages/LanguagesPage'
import LanguageDetailPage from './pages/LanguageDetailPage'
import AddLanguagePage from './pages/AddLanguagePage'
import SettingsPage from './pages/SettingsPage'
import NotFoundPage from './pages/NotFoundPage'

function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<HomePage />} />
        <Route path="/languages" element={<LanguagesPage />} />
        <Route path="/languages/add" element={<AddLanguagePage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="/languages/:id" element={<LanguageDetailPage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  )
}

export default App
