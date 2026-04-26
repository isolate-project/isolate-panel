# Isolate Panel

> Легковесная панель управления прокси-ядрами **Xray**, **Sing-box** и **Mihomo** — для VPS с ограниченными ресурсами.

**[Read in English](README.md)**

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![CI](https://github.com/isolate-project/isolate-panel/actions/workflows/test.yml/badge.svg)](https://github.com/isolate-project/isolate-panel/actions/workflows/test.yml)

---

## ✨ Ключевые возможности

| Категория | Описание |
|---|---|
| 🔄 **Мультиядро** | Xray, Sing-box, Mihomo — запуск / остановка / перезапуск через Supervisord |
| 👥 **Управление пользователями** | CRUD, квоты трафика, сроки действия, авто-блокировка при превышении, ссылки подписок |
| 📡 **25+ протоколов** | VLESS, VMess, Trojan, Shadowsocks, Hysteria2, TUIC v4/v5, Naive, AnyTLS, XHTTP, Snell, SSR и другие |
| 🔗 **Подписки** | Авто-детект, Clash, Sing-box, Isolate форматы; QR-коды; короткие ссылки |
| 📊 **Дашборд реального времени** | Статистика через WebSocket: активные соединения, трафик, статус ядер — с fallback на polling |
| 📈 **Аналитика трафика** | 7-дневная история, график топ-пользователей, часовая/дневная агрегация |
| 🔁 **Сброс трафика** | Автоматический еженедельный / ежемесячный сброс по расписанию |
| 🔒 **Сертификаты** | ACME / Let's Encrypt авто-выпуск, ручная загрузка, продление, отзыв |
| ☁️ **Cloudflare WARP** | Интеграция WARP с управлением маршрутами и пресетами (игры, стриминг и т.д.) |
| 🌍 **GeoIP / GeoSite** | Автообновление баз данных, маршрутизация по странам и категориям |
| 💾 **Шифрованные бэкапы** | Потоковое шифрование AES-256-GCM, настраиваемое хранение, cron-расписание |
| 🔔 **Уведомления** | Telegram-бот + Webhook (HMAC-подпись), настраиваемые триггеры событий |
| 📝 **Аудит-лог** | Неизменяемый журнал всех действий администратора |
| 🛡️ **2FA / TOTP** | Двухфакторная аутентификация на основе TOTP |
| 🖥️ **CLI** | CLI на Cobra: `isolate-panel user list`, `core start xray`, `backup create` |
| 🔐 **Безопасность** | JWT + Argon2id, CSP-заголовки, rate limiting, валидация запросов, доступ только через SSH-туннель |

---

## 🧭 Матрица протоколов

| Протокол | Sing-box | Xray | Mihomo | Транспорт |
|---|:---:|:---:|:---:|---|
| HTTP | ✅ | ✅ | ✅ | — |
| SOCKS5 | ✅ | ✅ | ✅ | — |
| Mixed (HTTP+SOCKS5) | ✅ | — | ✅ | — |
| Shadowsocks | ✅ | ✅ | ✅ | WS, gRPC |
| VMess | ✅ | ✅ | ✅ | WS, gRPC, HTTP, HTTPUpgrade |
| VLESS | ✅ | ✅ | ✅ | WS, gRPC, HTTP, HTTPUpgrade |
| Trojan | ✅ | ✅ | ✅ | WS, gRPC |
| Hysteria2 | ✅ | ✅ | ✅ | QUIC |
| TUIC v4 | ✅ | — | ✅ | QUIC |
| TUIC v5 | ✅ | — | ✅ | QUIC |
| Naive | ✅ | — | — | — |
| AnyTLS | ✅ | — | — | — |
| XHTTP | — | ✅ | — | — |
| Redirect | ✅ | — | ✅ | — |
| Mieru | — | — | ✅ | — |
| Sudoku | — | — | ✅ | — |
| TrustTunnel | — | — | ✅ | — |
| ShadowsocksR | — | — | ✅ | — |
| Snell | — | — | ✅ | — |
| MASQUE (исходящий) | — | — | ✅ | — |
| Tor (исходящий) | ✅ | — | — | — |

> Все протоколы поддерживают **TLS** и **REALITY**, где применимо.

---

## 🏗️ Технологический стек

| Слой | Технологии |
|---|---|
| Backend | Go 1.26, Fiber v3, GORM, SQLite (WAL) |
| Frontend | Preact 10, TypeScript 5.9, Vite 6, Tailwind CSS 4, Zustand |
| Аутентификация | JWT (access + refresh), Argon2id, TOTP (pquerna/otp) |
| Управление процессами | Supervisord (XML-RPC) |
| Развёртывание | Docker, Alpine Linux, multi-stage build |
| Логирование | Zerolog (структурированный JSON) |

---

## 🚀 Быстрый старт

### Требования

- Docker 20.10+
- Docker Compose 2.0+

### Установка одной командой (рекомендуется для VPS)

```bash
bash <(curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/install.sh)
```

### Ручная установка

```bash
mkdir -p /opt/isolate-panel && cd /opt/isolate-panel
curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/docker-compose.yml -o docker-compose.yml
curl -sL https://raw.githubusercontent.com/isolate-project/isolate-panel/main/docker/.env.example -o .env
nano .env   # задайте JWT_SECRET и ADMIN_PASSWORD
docker compose up -d
```

### Доступ через SSH-туннель

Панель слушает **только** на `localhost:8080` — она никогда не привязывается к публичному интерфейсу. Откройте туннель с вашего компьютера:

```bash
ssh -L 8080:localhost:8080 user@ip-вашего-сервера
```

Затем откройте <http://localhost:8080> в браузере.

**Логин по умолчанию:** `admin` / значение `ADMIN_PASSWORD` из `.env`

### Обновление

```bash
cd /opt/isolate-panel
docker compose pull && docker compose up -d
```

---

## 🔐 Доступ через SSH-туннель

Панель **принципиально не поддерживает** прямой доступ из интернета — это функция безопасности, а не ограничение.

| Сценарий | Команда |
|---|---|
| Прямой туннель | `ssh -L 8080:localhost:8080 user@host` |
| Фоновый туннель | `ssh -fNL 8080:localhost:8080 user@host` |
| Через jump-хост | `ssh -J jump-host -L 8080:localhost:8080 user@host` |

---

## 🛠️ Разработка

### Полный стек с hot reload (рекомендуется)

```bash
cd docker
docker compose -f docker-compose.dev.yml up --build
```

### Или запуск отдельно

```bash
# Backend (Go-сервер на :8080)
cd backend && make run

# Frontend (Vite dev-сервер на :5173, проксирует /api → :8080)
cd frontend && npm run dev
```

### Запуск тестов

```bash
# Backend
cd backend
make test                                                      # все тесты
go test -v -run TestFoo ./internal/api/...                     # один тест
go test ./... -coverprofile=coverage.out                       # с покрытием

# Frontend
cd frontend
npm test                                                       # Vitest unit-тесты
npm run test:e2e                                               # Playwright e2e-тесты
```

### Линтинг

```bash
cd backend && make lint           # golangci-lint
cd frontend && npm run lint       # ESLint
cd frontend && npm run typecheck  # tsc --noEmit
```

---

## 📁 Структура проекта

```
isolate-panel/
├── backend/
│   ├── cmd/
│   │   ├── server/main.go           # Точка входа приложения
│   │   └── migrate/main.go          # Инструмент миграций БД
│   ├── internal/
│   │   ├── api/                     # HTTP-обработчики Fiber (один файл на домен)
│   │   ├── services/                # Слой бизнес-логики
│   │   ├── models/                  # GORM-модели (16 доменных моделей)
│   │   ├── middleware/              # Auth, rate limit, audit, security headers
│   │   ├── scheduler/              # Cron-задачи (бэкап, сброс трафика)
│   │   ├── app/                     # DI-связывание, маршруты, фоновые воркеры
│   │   ├── auth/                    # JWT + Argon2id + TOTP
│   │   ├── cache/                   # Ristretto cache manager
│   │   ├── cores/                   # Адаптеры ядер (xray/, singbox/, mihomo/)
│   │   ├── protocol/               # Реестр схем протоколов (25+ протоколов)
│   │   ├── database/               # Миграции (39 шагов), сиды
│   │   └── config/                 # Загрузка конфигурации через Viper
│   ├── tests/                       # Интеграционные, e2e, edge-case, leak-тесты
│   └── Makefile
├── frontend/
│   ├── src/
│   │   ├── pages/                   # 19 страничных компонентов (lazy-loaded)
│   │   ├── components/             # UI-примитивы + feature-компоненты
│   │   ├── hooks/                  # Кастомные хуки (useUsers, useCores, useWebSocket, …)
│   │   ├── stores/                 # Zustand-сторы (auth, theme, toast)
│   │   └── api/                    # Axios-клиент + типизированные endpoints
│   └── e2e/                        # Playwright-тесты
├── cli/                            # CLI на Cobra (отдельный go.mod)
├── docker/                         # Dockerfile, Compose, Supervisord-конфиги
└── docs/                           # Архитектура, API-справка, руководства
```

---

## 🤝 Участие в проекте

Мы приветствуем вклад! Пожалуйста, прочитайте [Руководство для контрибьюторов](docs/CONTRIBUTING.md):

- **Настройка среды разработки** — как запустить проект локально
- **Стиль кода** — форматирование Go, TypeScript/ESLint, конвенции коммитов
- **Процесс Pull Request** — именование веток, шаблон PR, чеклист ревью
- **Репортинг проблем** — баг-репорты, запросы фич

### Краткие правила

1. **Форкните** репозиторий и создайте feature-ветку от `develop`
2. **Пишите тесты** для любой новой функциональности
3. **Запускайте линтеры** перед коммитом: `make lint` (backend), `npm run lint` (frontend)
4. **Одна фича — один PR** — не смешивайте разные изменения
5. **Conventional commits** — `feat:`, `fix:`, `docs:`, `chore:`

---

## 📚 Документация

| Документ | Описание |
|---|---|
| [Генеральный план](docs/MASTER_PLAN.md) | Roadmap проекта, архитектурное видение, фазы разработки |
| [Руководство пользователя](docs/USER_MANUAL.md) | Полное руководство для системных администраторов |
| [Для контрибьюторов](docs/CONTRIBUTING.md) | Настройка среды, стиль кода, процесс PR |
| [Архитектура](docs/ARCHITECTURE.md) | Подробное описание архитектуры системы |
| [API-справка](docs/API.md) | Документация REST API |
| [CLI-справка](docs/CLI.md) | Справка по командам Cobra CLI |

---

## 🛡️ Безопасность

- Панель доступна **только через SSH-туннель** — никогда не торчит в интернет
- Подписки отдаются на отдельном публичном порту с TLS
- **Argon2id**-хеширование паролей (отраслевой стандарт)
- **JWT** access-токены (15 мин) + refresh-токены (7 дней, хранятся хешированными)
- **TOTP** двухфакторная аутентификация
- Rate limiting: 5/мин (вход), 60/мин (стандарт), 10/мин (тяжёлые операции)
- Content Security Policy + security-заголовки
- Автоматическая генерация API-ключей для прокси-ядер
- Swagger UI **отключён** в продакшене
- Неизменяемый аудит-лог всех действий администратора
- Автоматическая очистка данных (просроченные токены, старые логи)

---

## ⚖️ Лицензия

[MIT](LICENSE) © 2026 isolate-project
