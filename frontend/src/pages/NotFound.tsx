import { useTranslation } from 'react-i18next'
import { PageLayout } from '../components/layout/PageLayout'
import { PageHeader } from '../components/layout/PageHeader'
import { Card } from '../components/ui/Card'

export function NotFound() {
  const { t } = useTranslation()

  return (
    <PageLayout>
      <PageHeader title={t('errors.pageNotFound')} />
      
      <Card className="text-center py-12">
        <h2 className="text-4xl font-bold text-primary mb-4">404</h2>
        <p className="text-secondary mb-6">
          {t('errors.pageNotFoundDescription')}
        </p>
        <a
          href="/"
          className="text-primary hover:text-primary-hover transition-base"
        >
          {t('errors.goBackDashboard')}
        </a>
      </Card>
    </PageLayout>
  )
}
