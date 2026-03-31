import { useEffect } from 'preact/hooks'

const APP_NAME = 'Isolate Panel'

interface MetaTagsOptions {
  title?: string
  description?: string
}

/**
 * Hook to dynamically update document title and meta description.
 * Title is formatted as "Page Title — Isolate Panel".
 */
export function useDocumentTitle(pageTitle: string) {
  useEffect(() => {
    const previousTitle = document.title
    document.title = pageTitle ? `${pageTitle} — ${APP_NAME}` : APP_NAME

    return () => {
      document.title = previousTitle
    }
  }, [pageTitle])
}

/**
 * Hook to set both title and meta description.
 */
export function useMetaTags({ title, description }: MetaTagsOptions) {
  useEffect(() => {
    const previousTitle = document.title

    if (title) {
      document.title = `${title} — ${APP_NAME}`
    }

    let metaDesc = document.querySelector<HTMLMetaElement>('meta[name="description"]')
    const previousDescription = metaDesc?.content

    if (description) {
      if (!metaDesc) {
        metaDesc = document.createElement('meta')
        metaDesc.name = 'description'
        document.head.appendChild(metaDesc)
      }
      metaDesc.content = description
    }

    return () => {
      document.title = previousTitle
      if (metaDesc && previousDescription !== undefined) {
        metaDesc.content = previousDescription || ''
      }
    }
  }, [title, description])
}
