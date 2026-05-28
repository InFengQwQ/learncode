import { useEffect, useRef, useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { languages, versions as versionsApi, type Language, type LanguageVersion, type ResearchResult } from '../api/client'
import DetailHeader from '../components/language/DetailHeader'
import ResourcePanel from '../components/language/ResourcePanel'
import VersionList from '../components/language/VersionList'

export default function LanguageDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [lang, setLang] = useState<Language | null>(null)
  const [vers, setVers] = useState<LanguageVersion[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [deleting, setDeleting] = useState(false)

  // Research state
  const [researching, setResearching] = useState(false)
  const [researchResult, setResearchResult] = useState<ResearchResult | null>(null)
  const [researchError, setResearchError] = useState('')
  const researchAbortRef = useRef<AbortController | null>(null)

  async function handleResearch() {
    if (!id) return
    const controller = new AbortController()
    researchAbortRef.current = controller
    setResearching(true)
    setResearchError('')
    setResearchResult(null)
    try {
      const res = await languages.research(id, controller.signal)
      if (res.ok && res.data) {
        setResearchResult(res.data)
        const langRes = await languages.get(id)
        if (langRes.ok && langRes.data) setLang(langRes.data)
      } else {
        setResearchError(res.error ?? '研究失败')
      }
    } catch (e) {
      if (controller.signal.aborted) { setResearching(false); return }
      setResearchError('网络错误')
    } finally {
      setResearching(false)
    }
  }

  function handleCancelResearch() {
    researchAbortRef.current?.abort()
    researchAbortRef.current = null
    setResearching(false)
  }

  function loadVersions() {
    if (!id) return
    Promise.all([languages.get(id), versionsApi.listByLanguage(id)]).then(
      ([langRes, verRes]) => {
        if (langRes.ok && langRes.data) {
          setLang(langRes.data)
          if (verRes.ok && verRes.data) setVers(verRes.data)
        } else {
          setError(langRes.error ?? '加载失败')
        }
        setLoading(false)
      },
    )
  }

  useEffect(() => {
    loadVersions()
  }, [id])

  async function handleDelete() {
    if (!id || !lang) return
    if (!confirm(`确定要删除 "${lang.name}"？此操作不可撤销。`)) return
    setDeleting(true)
    try {
      const res = await languages.delete(id)
      if (res.ok) navigate('/languages')
      else setError(res.error ?? '删除失败')
    } catch {
      setError('网络错误')
    } finally {
      setDeleting(false)
    }
  }

  if (loading) {
    return (
      <div className="flex animate-fade-in items-center justify-center py-20">
        <p className="text-sm text-text-muted">加载中…</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex animate-fade-in items-center justify-center py-20">
        <p className="text-sm text-danger">错误: {error}</p>
      </div>
    )
  }

  if (!lang) return null

  return (
    <div className="animate-fade-in">
      <Link
        to="/languages"
        className="text-sm text-text-secondary transition-colors hover:text-text-primary"
      >
        &larr; 语言列表
      </Link>

      <div className="mt-6">
        <DetailHeader
          language={lang}
          researching={researching}
          deleting={deleting}
          onResearch={handleResearch}
          onCancelResearch={handleCancelResearch}
          onDelete={handleDelete}
        />
      </div>

      {/* Research error */}
      {researchError && (
        <div className="mt-4 rounded-xl border border-danger-bg bg-danger-bg/20 px-6 py-3">
          <p className="text-sm text-danger">{researchError}</p>
        </div>
      )}

      {/* Research results */}
      {researchResult && <ResourcePanel result={researchResult} />}

      {/* Versions section */}
      <VersionList
        versions={vers}
        language={lang}
        onVersionsChange={setVers}
      />
    </div>
  )
}
