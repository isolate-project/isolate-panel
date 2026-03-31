import { Component, ComponentChildren } from 'preact'
import { AlertCircle } from 'lucide-preact'

interface Props {
  children: ComponentChildren
  fallback?: (error: Error, reset: () => void) => ComponentChildren
}

interface State {
  hasError: boolean
  error?: Error
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: unknown) {
    console.error('ErrorBoundary caught an error:', error, errorInfo)
  }

  resetError = () => {
    this.setState({ hasError: false, error: undefined })
  }

  render() {
    if (this.state.hasError && this.state.error) {
      if (this.props.fallback) {
        return this.props.fallback(this.state.error, this.resetError)
      }
      return (
        <div className="flex h-screen w-full flex-col items-center justify-center p-8 text-center bg-bg-secondary">
          <div className="max-w-md w-full bg-bg-primary p-8 rounded-xl border border-color-danger/20 shadow-lg flex flex-col items-center">
            <div className="mb-6 rounded-full bg-danger/10 p-4 text-color-danger">
              <AlertCircle className="w-12 h-12" />
            </div>
            <h2 className="mb-2 text-2xl font-bold text-text-primary">Something went wrong</h2>
            <p className="mb-6 text-sm text-text-secondary">
              {this.state.error.message || "An unexpected error occurred while rendering this component."}
            </p>
            <button
              onClick={this.resetError}
              className="rounded-lg bg-color-primary px-6 py-2.5 text-sm font-medium text-white hover:bg-color-primary/90 transition-colors shadow-sm"
            >
              Try again
            </button>
          </div>
        </div>
      )
    }

    return this.props.children
  }
}
