import { ComponentChildren } from 'preact'
import { useState } from 'preact/hooks'
import { Sidebar } from './Sidebar'
import { Header } from './Header'

interface PageLayoutProps {
  children: ComponentChildren
}

export function PageLayout({ children }: PageLayoutProps) {
  const [isSidebarOpen, setIsSidebarOpen] = useState(false)

  return (
    <div className="min-h-screen bg-bg-secondary text-text-primary selection:bg-color-primary/20">
      {/* Sidebar */}
      <Sidebar
        isOpen={isSidebarOpen}
        onClose={() => setIsSidebarOpen(false)}
      />

      {/* Main Content */}
      <div className="flex flex-col min-h-screen transition-all duration-300 lg:pl-72">
        <Header onMenuClick={() => setIsSidebarOpen(!isSidebarOpen)} />
        <main className="flex-1 p-4 sm:p-6 lg:p-8 max-w-[1600px] w-full mx-auto">
          {children}
        </main>
      </div>
    </div>
  )
}
