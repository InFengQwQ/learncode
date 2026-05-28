import { Component, type ReactNode, type ErrorInfo } from 'react'
import Button from './ui/Button'

interface ErrorBoundaryProps {
  children: ReactNode
  fallback?: ReactNode
}

interface ErrorBoundaryState {
  hasError: boolean
  error: Error | null
}

export default class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('ErrorBoundary caught:', error, info.componentStack)
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null })
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback
      }

      return (
        <div className="flex min-h-screen items-center justify-center bg-bg-base p-8">
          <div className="max-w-md text-center">
            <div className="mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-danger-bg">
              <span className="text-2xl text-danger">!</span>
            </div>
            <h1 className="mb-2 text-xl font-semibold text-text-primary">
              页面出现错误
            </h1>
            <p className="mb-2 text-sm text-text-secondary">
              发生了意外错误，请尝试重新加载。
            </p>
            {this.state.error && (
              <pre className="mb-6 max-h-24 overflow-auto rounded-lg bg-bg-elevated p-3 text-xs text-text-muted">
                {this.state.error.message}
              </pre>
            )}
            <div className="flex justify-center gap-3">
              <Button onClick={this.handleReset}>重试</Button>
              <Button
                variant="secondary"
                onClick={() => window.location.reload()}
              >
                重新加载页面
              </Button>
            </div>
          </div>
        </div>
      )
    }

    return this.props.children
  }
}
