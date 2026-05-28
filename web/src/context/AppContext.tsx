import { createContext, useContext, useState, useCallback, type ReactNode } from 'react'

interface AppContextValue {
  refreshing: boolean
  lastRefresh: number
  triggerRefresh: () => void
}

const AppContext = createContext<AppContextValue | null>(null)

export function AppProvider({ children }: { children: ReactNode }) {
  const [refreshing, setRefreshing] = useState(false)
  const [lastRefresh, setLastRefresh] = useState(Date.now())

  const triggerRefresh = useCallback(() => {
    setRefreshing(true)
    setLastRefresh(Date.now())
    // Reset refreshing state after a short delay
    setTimeout(() => setRefreshing(false), 500)
  }, [])

  return (
    <AppContext.Provider value={{ refreshing, lastRefresh, triggerRefresh }}>
      {children}
    </AppContext.Provider>
  )
}

export function useAppContext(): AppContextValue {
  const ctx = useContext(AppContext)
  if (!ctx) {
    throw new Error('useAppContext must be used within AppProvider')
  }
  return ctx
}
