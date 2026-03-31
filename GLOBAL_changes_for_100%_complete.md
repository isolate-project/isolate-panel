# 🎯 GLOBAL: План доработок до 100% реализации

**Дата создания:** 30 марта 2026 (Обновлено на основе валидации)
**Версия проекта:** 0.1.0 → **1.0.0**  
**Статус:** Активный план

---

## 📊 Общая оценка по фазам

| Фаза | Название | Текущая готовность | Цель |
|------|----------|-------------------|------|
| Phase 0 | Setup & Infrastructure | ✅ 100% | ✅ 100% |
| Phase 1 | MVP Backend | ⚠️ 92% | ✅ 100% |
| Phase 2 | MVP Frontend | ✅ 100% | ✅ 100% |
| Phase 3 | Inbound/Outbound Management | ✅ 100% | ✅ 100% |
| Phase 4 | Subscriptions | ✅ 100% | ✅ 100% |
| Phase 5 | Certificates | ✅ 100% | ✅ 100% |
| Phase 6 | Monitoring & Statistics | ⚠️ 75% | ✅ 100% |
| Phase 7 | Xray + Mihomo Cores | ✅ 100% | ✅ 100% |
| Phase 8 | WARP + GeoIP | ⚠️ 50% | ✅ 100% |
| Phase 9 | Backup System | ⚠️ 87% | ✅ 100% |
| Phase 10 | Notifications | ⚠️ 70% | ✅ 100% |
| Phase 11 | CLI Interface | ⚠️ 82% | ✅ 100% |
| Phase 12 | Docker Deployment | ⚠️ 82% | ✅ 100% |
| Phase 13 | Testing | ⚠️ 58% | ✅ 100% |
| Phase 14 | Optimization & Polish | ⚠️ 35% | ✅ 100% |

---

## Phase 0: Setup & Infrastructure ✅ 100%

**Статус:** Полностью завершена.

- [x] Структура проекта
- [x] Docker multi-stage build
- [x] Supervisord configuration
- [x] Go modules
- [x] Frontend Vite + Preact setup

**Нет доработок.**

---

## Phase 1: MVP Backend — 100???% → 100%

**Реализовано:** Fiber API, GORM/SQLite, JWT auth, core lifecycle, middleware, database, запись `last_login_at`.

### Что осталось:

- [x] **1.1** Разрешить дублирование SQL схемы. Убрать raw SQL инициализацию из `docker-entrypoint.sh` и использовать `cmd/migrate` (`migrate up`), чтобы не дублировать код миграций.
- [x] **1.2** Добавить graceful shutdown с обработкой SIGTERM/SIGINT
  - Остановить traffic collector, connection tracker, data aggregator
  - Закрыть database connection
  - Завершить backup scheduler
  - Вызвать `CertificateService.Stop()`
- [x] **1.3** Использовать `ADMIN_PASSWORD` из `.env` в `docker-entrypoint.sh` (вместо hardcoded `admin`)
- [x] **1.4** Добавить тесты для пакетов без тестов:
  - `internal/api`
  - `internal/cache`
  - `internal/config`
  - `internal/database`
  - `internal/logger`
  - `internal/models`
  - `internal/scheduler`
  - `internal/stats`
  - `internal/cores/singbox`
  - `internal/acme`
- [ ] **1.5** (Отложено) Создать явную CLI команду `isolate-migrate seed-dev` для опционального засева базы фейковыми development юзерами.

---

## Phase 2: MVP Frontend — ✅ 100%

**Реализовано:** 17 страниц, 25 UI компонентов, Zustand stores, i18n, Tailwind v4. Частичные юнит-тесты (hooks, UI components).

### Что осталось:

- [x] **2.1** Исправить все TypeScript ошибки (восстановить `tsc --noEmit` check)
- [x] **2.2** Ужесточить ESLint конфигурацию (сейчас не strict, разрешены `any` и `ban-ts-comment`)
- [x] **2.3** Добавить error boundaries для всех страниц
- [x] **2.4** Frontend unit тесты (Vitest) для непокрытых частей:
  - Tесты для сторов (`authStore`, `themeStore`)
  - Тесты для страниц (page render tests)
- [x] **2.5** Accessibility (a11y):
  - ARIA labels где они отсутствуют (сейчас есть только в ~4 компонентах)
  - Keyboard navigation
  - Focus management в модальных окнах
- [x] **2.6** Обновить мета-теги и SEO для Login/Dashboard

---

## Phase 3: Inbound/Outbound Management — ✅ 100%

**Реализовано:** Protocol Schema Registry (25 протоколов), dynamic form generation, port manager, outbound service tests, bulk assign API (`/bulk`).

### Что осталось (Выполнено):

- [x] **3.1** Unit тесты для Protocol Schema Registry
- [x] **3.2** Unit тесты для Port Manager (уже частично: `port_manager_test.go`)
- [x] **3.3** Валидация конфликтов портов при создании inbound через UI (показать ошибку, а не crash)
- [x] **3.4** Интегрировать Bulk assign/unassign пользователей к inbound со стороны UI (в бекенде API `/bulk` уже готово)

---

## Phase 4: Subscriptions — ✅ 100%

**Реализовано:** V2Ray/Clash/Sing-box форматы, short URL, rate limiting, access logging, отображение QR кода в UI, выдача заголовков `Profile-Update-Interval`.

### Что осталось (Выполнено):

- [x] **4.1** Интегрировать кэширование подписок (`internal/cache/manager.go` написан, но это мёртвый код, он не прокинут в зависимости сервисов и `main.go`)
- [x] **4.2** User-Agent auto-detection для формата подписки (сейчас формат задаётся только при явном указании в URL: `/sub/:token/clash` или `/sub/:token/singbox`)
- [x] **4.3** Unit тесты для subscription форматов (V2Ray, Clash, Sing-box)
- [x] **4.4** Integration тесты: создать user → создать inbound → assign → get subscription → verify format

---

## Phase 5: Certificates — ✅ 100%

**Реализовано:** `certificate_service.go`, Certificates UI, автоматическое обновление (time.Ticker scheduler работает), ручная загрузка (`POST /upload`), Cloudflare DNS-01 challenge, привязка сертификата к inbound (БД + UI selector), unit и integration тесты.

### Что выполнено:

- [x] **5.1** Проверить и довести до рабочего состояния DNS-01 challenge (Cloudflare)
  - Добавлена поддержка `CLOUDFLARE_API_TOKEN` (помимо `API_KEY + EMAIL`)
  - Graceful degradation: при отсутствии credentials ACME не инициализируется, но панель работает
  - ACME init failure стала non-fatal (manual upload продолжает работать)
- [x] **5.2** Привязка сертификатов к inbound
  - Backend: валидация `tls_cert_id` при Create/Update inbound (проверка существования и статуса)
  - Frontend: dropdown выбора сертификата в InboundForm при `tls_enabled = true`
  - Автоочистка `tls_cert_id` при отключении TLS
  - **Примечание:** подстановка cert paths в конфиг-генераторы ядер — задача Phase 7.6
- [x] **5.3** Wildcard сертификат поддержка через DNS-01 (код уже был, покрыто тестами)
- [x] **5.4** Unit тесты для certificate service (15 тестов) и ACME (14 тестов)
- [x] **5.5** Integration тесты: lifecycle (create→list→get→status→delete) + cert-inbound binding

---

## Phase 6: Monitoring & Statistics — ✅ 100%

**Реализовано:** `traffic_collector`, `connection_tracker`, `data_aggregator`, `data_retention_service`, `quota_enforcer`, Dashboard UI с Chart.js графиками.

### Выполнено:

- [x] **6.1** Smoke-тесты Stats API всех ядер (Sing-box 10 тестов, Mihomo 11 тестов)
- [x] **6.2** Smart Quota Enforcement: threshold warnings (80%/90%), targeted reload, warning dedup. (**Примечание:** Auto-reset трафика по расписанию отложен на Phase 14.16)
- [x] **6.3** `DataRetentionService`: вынесен в отдельный файл, настраиваемые retention periods через Settings, структурированное zerolog логирование
- [x] **6.4** DisconnectUser + KickUser API: реальное закрытие соединений через ядра, удаление пользователей из inbounds. Добавлены эндпоинты: `GET /traffic/overview`, `GET /traffic/top-users`, `POST /user/:id/kick`
- [x] **6.5** Dashboard графики: **Chart.js** + `react-chartjs-2`, компоненты `TrafficChart` (Line) и `TopUsersChart` (Bar)
- [x] **6.6** Unit тесты: connection_tracker(4), quota_enforcer(4), data_aggregator(1), data_retention(3) = 12 тестов
- [x] **6.7** Integration тест: full monitoring flow (create → assign → record → aggregate → quota → reset → retention)

---

## Phase 7: Xray + Mihomo Cores — ✅ 100%

**Реализовано:** Config generators для Xray/Mihomo/Sing-box, stats clients, unified cores package, TLS/Reality/Transport.

### Выполнено:

- [x] **7.1** Консолидация пакетов `internal/core/` → `internal/cores/`. Перенесены `manager.go` и `supervisor.go`, удалены дубликаты, обновлены импорты (5 файлов).
- [x] **7.2** Sing-box config generator: `cores/singbox/config.go` (700+ строк), 12 протоколов, DB-driven user mapping, Clash API для stats. 13 unit-тестов.
- [x] **7.3** Protocol sweep: все протоколы покрыты по ядрам (Xray: 7 + socks/http, Sing-box: 12, Mihomo: 14). Добавлены socks/http accounts для Xray.
- [x] **7.4** Transport options: WebSocket, gRPC, H2, XHTTP для Xray; ws/grpc/http для Sing-box; ws-opts/grpc-opts/h2-opts для Mihomo — все из ConfigJSON.
- [x] **7.5** Reality settings: Xray (realitySettings), Sing-box (TLS.Reality), Mihomo (reality-opts) — из RealityConfigJSON.
- [x] **7.6** TLS конфигурация: все 3 генератора загружают cert_path/key_path из Certificate через TLSCertID.


---

## Phase 8: WARP + GeoIP — ✅ 100%

**Реализовано:** WARP registration API, GeoIP/GeoSite management backend/UI.

### Что осталось:

- [x] **8.1** Интеграция WARP с ядрами (маршрутизация, добавление WARP outbound в генераторы конфигов!)
- [x] **8.2** WARP token auto-refresh
- [x] **8.3** Интеграция GeoIP/GeoSite баз с конфигурациями ядер
- [x] **8.4** Автоматическое скачивание и кэширование Geo данных
- [x] **8.5** Unit тесты для WARP registration и rule generation
- [x] **8.6** UI: статус подключения WARP (online/offline) в интерфейсе (Бэкенд готов, IP отображается)

---

## Phase 9: Backup System — 87% → 100%

**Реализовано:** backup API, Backup Scheduler, UI страница со списком бэкапов, restore API (`/restore`).

### Что осталось:

- [ ] **9.1** Реализовать заявленное шифрование бэкапов (AES-256-GCM) — сейчас данные сжимаются в tar/gz, но не шифруются.
- [ ] **9.2** Восстановление из бэкапа через UI (API готово, нужен dialog confirmation на фронтенде)
- [ ] **9.3** Retention policy (настроить автоудаление старых бэкапов)
- [ ] **9.4** Integration тест: backup → restore → verify data integrity
- [ ] **9.5** Документация по процедуре восстановления в DEPLOYMENT.md

---

## Phase 10: Notifications — 70% → 100%

**Реализовано:** `notification_service`, Telegram Webhook integration, UI.

### Что осталось:

- [ ] **10.1** Проверить Telegram bot integration (real token tests, quota thresholds, cert validation errors)
- [ ] **10.2** Уведомления через Webhooks: реализовать механизм Retry (exponential backoff)
- [ ] **10.3** Тестовая отправка нотификаций через кнопку "Test notification" в UI и триггеры на core crash
- [ ] **10.4** Unit тесты для Telegram и Webhook notifiers
- [ ] **10.5** Email уведомления (отложено в v1.1.0)

---

## Phase 11: CLI Interface — 82% → 100%

**Реализовано:** Cobra framework, команды (user, core, system), shell completions (bash, zsh, fish).

### Что осталось:

- [ ] **11.1** Ручная проверка команд (list/add/delete/start/stop/restore)
- [ ] **11.2** Добавить Unit-тесты для команд директории `cli/cmd/`
- [ ] **11.3** Внедрить CLI output formatters (table, json, yaml)
- [ ] **11.4** Integration тесты: вызов из CLI → вызов API → проверка результата
- [ ] **11.5** Man pages / help documentation

---

## Phase 12: Docker Deployment — 82% → 100%

**Реализовано:** Dockerfile (production multi-stage), docker-compose.yml, entrypoint.sh, Dockerfile.dev c hot reload (Air/Vite).

### Что осталось:

- [ ] **12.1** Обновить версии зависимостей контеинеров (Go 1.25, новые релизы Sing-box и Mihomo)
- [ ] **12.2** Разместить `supervisord.dev.conf` (файл отсутствует) для корректной работы hot-reload окружения
- [ ] **12.3** Настроить Health check для процессов ядер (сейчас он проверяет только панель)
- [ ] **12.4** Лог ротация в контейнере
- [ ] **12.5** Выполнение от имени non-root user (security hardening у supervisord/процессов)
- [ ] **12.6** Убрать `version: '3.8'` из compose-файлов ввиду deprecation

---

## Phase 13: Testing — 58% → 100%

**Реализовано:** Backend API и Services частично покрыты, протокольные схемы, Frontend UI компоненты и hooks (vitest), E2E UI flow tests (Playwright), базовый пайплайн GH Actions.

### Что осталось:

- [ ] **13.1** Увеличить покрытие бекенда (документированные папки без тестов в 1.4)
- [ ] **13.2** Расширить E2E тесты (добавить core lifecycle start/stop, бэкапы create/restore)
- [ ] **13.3** Починить и обновить CI/CD pipeline в `.github/workflows/` (Golang/Node.js validation)
- [ ] **13.4** Golangci-lint: объединить конфиги (убрать мусорные `.golangci.bck.yml`), включить в пайплайн, исправить warnings
- [ ] **13.5** Очистить репозиторий: удалить артефакты-логи сборки (`build.log`, `docker-build.log`) и занести их в `.gitignore`

---

## Phase 14: Optimization & Polish — 35% → 100%

**Реализовано:** Performance индексы БД созданы через миграции.

### Что осталось:

#### Performance & Code Quality:
- [ ] **14.1** Добавить индекс на поле `users.subscription_token` (упущено из прошлой миграции)
- [ ] **14.2** Устранить мёртвый код: добавить `CacheManager` в провайдеры зависимостей (DI)
- [ ] **14.3** Использовать хуки WebSockets на Dashboard для real-time отзывчивости
- [ ] **14.4** Рефакторинг `main.go`. Разбить тяжеловесный код инициализации (460 строк) на модули
- [ ] **14.5** Вынести хардкодинг версию (`v0.1.0`) в флаги линкера `ldflags` при компиляции
- [ ] **14.6** Frontend code splitting: настроить React lazy loading для тяжелых страниц / роутов
- [ ] **14.16** Автоматический reset трафика по расписанию (monthly/weekly) — отложено из Phase 6.2

#### Security & Limits:
- [ ] **14.7** Разработать API Rate Limiting для *authenticated* эндпоинтов (сейчас он только на public endpoints)
- [ ] **14.8** Добавить централизованный Audit Log для критических действий
- [ ] **14.9** Создать общий Request Validation Middleware (перестать валидировать запросы внутри каждого хендлера отдельно)
- [ ] **14.10** Security Audit: внедрить CSP headers, выполнить `govulncheck` сканирование
- [ ] **14.11** 2FA / TOTP для админ-логина

#### Documentation:
- [ ] **14.12** Обновить README.md, чтобы он отражал текущую архитектуру
- [ ] **14.13** Сгенерировать API документацию (Swagger/OpenAPI)
- [ ] **14.14** Написать Architecture documentation и Contributing guide
- [ ] **14.15** Подготовить CHANGELOG.md к релизу v1.0.0

---

## 📌 Общий roadmap к v1.0.0

### Sprint 1 (Неделя 1-2): Рефакторинг и критические пропуски ← **ТЕКУЩИЙ**
- [ ] Устранение дублирования `internal/core` и `internal/cores`
- [ ] Миграция схемы БД прочь из `docker-entrypoint.sh`
- [ ] Phase 12 Fixes (добавить missing dev configs, почистить logs)
- [ ] Graceful shutdown бекенда
- [ ] Fix TypeScript errors, ESLint rules

### Sprint 2 (Неделя 3-4): Завершение Core функционала
- [ ] Активация системы кэширования для подписок (`CacheManager`)
- [ ] Внедрить шифрование для бэкапов (AES-256-GCM)
- [ ] DataRetentionService и Smart Quota Enforcement
- [ ] Frontend Dashboard: внедрение Chart.js исторических графиков

### Sprint 3 (Неделя 5-6): Ядра и Интеграция 
- [ ] WARP + GeoIP: сквозная трансляция настроек в ядра
- [ ] Telegram нотификации (end-to-end webhook integrations)
- [ ] Security fix: API Rate Limiting, Audit logs, Request validation

### Sprint 4 (Неделя 7-8): Качество и Покрытие (Testing)
- [ ] Поднять target coverage до 80% (написать падающие/недостающие unit и e2e тесты)
- [ ] Пофиксить и настроить CI pipeline в GitHub
- [ ] Устранить golangci-lint предупреждения

### Sprint 5 (Неделя 9-10): Полировка и Релиз
- [ ] Performance and bundle optimization (React Lazy)
- [ ] Написание доков (Swagger, Architecture, CHANGELOG)
- [ ] Security audit (CSP, HTTP Strict Transport Security, vuln validation)
- [ ] **Tag v1.0.0**
