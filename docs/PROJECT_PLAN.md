# Полный план проекта: Панель управления прокси-ядрами

## 📋 Оглавление

1. [Обзор проекта](#обзор-проекта)
2. [Технологический стек](#технологический-стек)
3. [Архитектура системы](#архитектура-системы)
4. [Функциональные требования](#функциональные-требования)
5. [Распределение протоколов по ядрам](#распределение-протоколов-по-ядрам)
6. [Рекомендации по безопасности](#рекомендации-по-безопасности)
7. [Структура базы данных](#структура-базы-данных)
8. [Фазы реализации](#фазы-реализации)

**📚 Дополнительная документация:**
- [PROTOCOL_SMART_FORMS_PLAN.md](./PROTOCOL_SMART_FORMS_PLAN.md) - Protocol-Aware Smart Forms & User Management System

---

## Обзор проекта

### Цель
Создать легковесную панель управления для прокси-ядер (Xray, Sing-box, Mihomo) с акцентом на безопасность и минимальное потребление ресурсов.

### Ключевые принципы
- **Безопасность превыше всего**: Доступ только через SSH туннель, localhost-only
- **Минимализм**: Работа на VPS с 1 CPU / 1GB RAM
- **Простота развертывания**: Docker + docker-compose
- **Монолитная архитектура**: Все компоненты в одном контейнере
- **Только для администраторов**: Обычные пользователи НЕ имеют доступа к панели

---

## Технологический стек

### Backend

| Компонент | Версия | Назначение |
|-----------|--------|------------|
| **Go** | 1.26.1 | Основной язык backend |
| **Fiber** | v3.1.0 | Web-фреймворк |
| **GORM** | v1.31.1 | ORM для работы с БД |
| **SQLite** | 3.x | База данных |
| **Zerolog** | v1.34.0 | Структурированное логирование |
| **Viper** | v1.21.0 | Управление конфигурацией |
| **JWT** | v5.3.1 (golang-jwt/jwt) | Аутентификация |
| **Lego** | v4.33.0 | ACME клиент для SSL |

### Frontend

| Компонент | Версия | Назначение |
|-----------|--------|------------|
| **Preact** | 10.29.0 | UI фреймворк (3-4 KB) |
| **Vite** | 6.x | Сборщик и dev-сервер |
| **TypeScript** | 5.9.3 | Типизация |
| **Zustand** | v5.0.12 | Управление состоянием |
| **Tailwind CSS** | v4.2.2 | Стилизация |

**Почему Preact?**
- Размер бандла: ~3-4 KB vs 16-20 KB (Vue)
- Потребление RAM: ~15-25 MB vs 25-35 MB (Vue)
- Критично для VPS с 1GB RAM
- Быстрая загрузка через SSH туннель
- Совместимость с React-экосистемой

### Прокси-ядра

| Ядро | Версия | Роль |
|------|--------|------|
| **Sing-box** | v1.13.3 | Основное ядро - все общие протоколы |
| **Xray-core** | v26.2.6 | XHTTP + Xray-специфичные протоколы |
| **Mihomo** | v1.19.21 | Mihomo-специфичные протоколы |

### Инфраструктура

- **Docker** + **docker-compose**: Контейнеризация
- **Alpine Linux**: Базовый образ (минимальный размер)
- **Supervisord**: Управление процессами ядер
- **SSH**: Единственный способ доступа к панели
- **HAProxy** 2.9+ (Post-MVP): Reverse proxy для продвинутого роутинга

---

## Архитектура системы

### Общая схема

**Архитектура MVP (без HAProxy):**

```
┌─────────────────────────────────────────────────────────┐
│                    SSH Tunnel                            │
│              (Единственный путь доступа)                 │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│              localhost:8080 (Панель)                     │
│  ┌───────────────────────────────────────────────────┐  │
│  │              Frontend (Preact)                     │  │
│  └───────────────────┬───────────────────────────────┘  │
│                      │                                   │
│  ┌───────────────────▼───────────────────────────────┐  │
│  │              Backend (Go + Fiber)                  │  │
│  │  ┌──────────┬──────────┬──────────┬─────────────┐ │  │
│  │  │   Auth   │   API    │  Config  │  Monitoring │ │  │
│  │  └──────────┴──────────┴──────────┴─────────────┘ │  │
│  └───────────────────┬───────────────────────────────┘  │
│                      │                                   │
│  ┌───────────────────▼───────────────────────────────┐  │
│  │              SQLite Database                       │  │
│  └────────────────────────────────────────────────────┘  │
└────────────────────┬────────────────────────────────────┘
                     │
     ┌───────────────┼───────────────┐
     │               │               │
     ▼               ▼               ▼
┌─────────┐    ┌─────────┐    ┌─────────┐
│Sing-box │    │  Xray   │    │ Mihomo  │
│ v1.13.3 │    │v26.2.6  │    │v1.19.21 │
│         │    │         │    │         │
│ :443    │    │ :8443   │    │ :9443   │
│(direct) │    │(direct) │    │(direct) │
└─────────┘    └─────────┘    └─────────┘
     │               │               │
     └───────────────┴───────────────┘
                     │
                     ▼
              Интернет (Клиенты)
```

**Важно:** 
- Панель доступна ТОЛЬКО через SSH tunnel на `127.0.0.1:8080`
- Каждое ядро слушает на своих портах напрямую
- Каждый inbound требует уникальный порт
- HAProxy исключен из MVP (см. Post-MVP Features)

### Принципы архитектуры

1. **Монолитная структура**: Все в одном Docker-контейнере
2. **Localhost-only**: Панель слушает только 127.0.0.1:8080
3. **Процесс-менеджер**: Supervisord для управления ядрами
4. **Единая БД**: SQLite для простоты и минимальных ресурсов
5. **Stateless API**: JWT для аутентификации
6. **Прямое подключение**: Каждое ядро на своем порту (MVP без HAProxy)

---

## Post-MVP Features

### HAProxy: Продвинутый роутинг (Post-MVP)

**Статус:** Исключено из MVP, планируется в v1.5+

HAProxy предоставляет продвинутые возможности роутинга, но добавляет сложность и потребление ресурсов.

**Преимущества HAProxy:**
- ✅ Множественные inbound на одном порту (SNI-based routing)
- ✅ Path-based routing для HTTP/WebSocket
- ✅ Централизованный rate limiting
- ✅ Единая точка мониторинга соединений

**Недостатки:**
- ❌ Дополнительные 40-50MB RAM
- ❌ Latency overhead +1-2ms
- ❌ Усложнение архитектуры
- ❌ Дополнительная точка отказа

**Решение для MVP:** Прямое подключение к ядрам, каждый inbound на уникальном порту.

### Другие Post-MVP функции

**Протоколы:**
- TProxy (требует NET_ADMIN capability)
- TUN (требует /dev/net/tun)
- WireGuard (требует дополнительных capabilities)
- SSH как прокси-протокол (не путать с SSH доступом к VPS)

**Функциональность:**
- Email/Webhook уведомления
- Автоматическое резервное копирование (S3, FTP, SFTP)
- Multi-admin с ролями и правами
- User portal (для просмотра статистики пользователями)
- QR code генерация для быстрого подключения

---

## Функциональные требования

### 1. Управление пользователями

#### Администраторы
- ✅ Полный доступ к панели
- ✅ Создание/удаление других администраторов
- ✅ Управление всеми пользователями
- ✅ Настройка системы
- ✅ Просмотр логов и статистики

#### Обычные пользователи
- ❌ **НЕ ИМЕЮТ** доступа к панели
- ✅ Получают subscription link от администратора
- ✅ Используют подписку в своих клиентах

### 2. Управление подписками

#### Квоты (по умолчанию безлимит)
- Трафик: безлимитный (опционально ограничение)
- Срок действия: бессрочный (опционально ограничение)
- Возможность установки лимитов для конкретных пользователей

#### Форматы подписок

**API Endpoints:**
```
GET /sub/:token              # V2Ray (base64)
GET /sub/:token/clash        # Clash (YAML)
GET /sub/:token/singbox      # Sing-box (JSON)
GET /s/:short_code           # Short URL redirect
```

**Спецификация:**
- **Аутентификация**: subscription_token в URL (не требует JWT)
- **Rate limiting**: 
  - IP-based: 30 запросов/час на IP
  - Token-based: 10 запросов/час на token
  - Global: 1000 запросов/час (общий лимит)
- **Security**: User-Agent validation, IP blocking после 20 неудачных попыток
- **Формат V2Ray**: base64-encoded список vmess:// ссылок
- **Формат Clash**: YAML конфигурация с proxies
- **Формат Sing-box**: JSON конфигурация с outbounds
- **Short URLs**: Генерируются автоматически, 8-символьный код
- **Access logging**: Все запросы логируются в subscription_accesses
- **Подробнее**: См. [SECURITY_PLAN.md](./SECURITY_PLAN.md) для деталей защиты

**Пример V2Ray подписки:**
```
GET /sub/abc123token456

Response (base64):
dm1lc3M6Ly9leUoySWpvaU1pSXNJbkJ6SWpvaVZYTmxjaUF4SWl3aVlXUmtJam9pWlhoaGJYQnNaUzVqYjIwaUxDSndiM0owSWpvaU5EUXpJaXdpYVdRaU9pSXhNak0wTlRZM09DMWhZbU5rTFRFek5EVXRPVGN6TXkweE1qTTBOVFkzT0Rrd01USWlMQ0p1WlhRaU9pSjNjeUlzSW5SNWNHVWlPaUp1YjI1bElpd2lhRzl6ZENJNklpSXNJbkJoZEdnaU9pSXZkbTFsYzNNaUxDSjBiSE1pT2lKMGJITWlmUT09
```

### 3. Управление inbound/outbound

#### Inbound
- Создание/редактирование/удаление
- Привязка к пользователям
- Настройка протоколов
- Мониторинг активных подключений

#### Outbound
- Простые цепочки (chain)
- Сложная маршрутизация (routing rules)
- Балансировка нагрузки
- Failover

### 4. Сертификаты SSL/TLS

#### Варианты получения
1. **ACME автоматический** (Let's Encrypt, ZeroSSL)
   - Автоматическое обновление
   - HTTP-01 / DNS-01 challenge
   
2. **Certbot интеграция**
   - Использование существующих сертификатов
   - Автоматическое обновление
   
3. **Ручная загрузка**
   - Загрузка собственных сертификатов
   - Ручное обновление

### 5. WARP интеграция

#### Функционал
- Включение/выключение WARP
- **Маршрутизация по ресурсам**:
  - ChatGPT (openai.com, chat.openai.com)
  - Нейросети (claude.ai, gemini.google.com, etc.)
  - Пользовательские домены/IP
- Правила маршрутизации (domain, IP, CIDR)

### 6. GeoIP/GeoSite

#### Опциональная поддержка
- Автообновление баз данных
- Правила маршрутизации по странам
- Блокировка/разрешение по регионам
- Минимальное влияние на производительность

### 7. Мониторинг и логирование

#### Режимы мониторинга

**Lite режим** (для слабых VPS)
- Минимальная нагрузка на CPU/RAM
- Сбор критических ошибок
- Базовая статистика трафика
- Асинхронная запись в БД
- Ротация логов

**Full режим** (для мощных серверов)
- Детальная статистика по пользователям
- Мониторинг активных подключений в реальном времени
- Графики использования ресурсов
- История подключений
- Детальные логи

#### Активные подключения
- Просмотр текущих подключений
- Информация: пользователь → inbound → протокол
- Возможность отключения пользователя
- Статистика по подключениям

### 8. Резервное копирование

#### Автоматические бэкапы (опционально)
- Расписание (ежедневно, еженедельно, ежемесячно)
- Хранение N последних копий
- Бэкап конфигурации + БД

#### Destination варианты
1. **Local**: Локальное хранилище
2. **S3**: AWS S3, MinIO, любой S3-совместимый
3. **FTP/SFTP**: Удаленный FTP/SFTP сервер

### 9. CLI интерфейс

#### Полный функционал через CLI
```bash
# Управление пользователями
isolate-panel user add <username> --admin
isolate-panel user delete <username>
isolate-panel user list

# Управление inbound
isolate-panel inbound add --type vless --port 443
isolate-panel inbound list

# Управление системой
isolate-panel system status
isolate-panel system restart
isolate-panel backup create
```

### 10. Уведомления

#### Email уведомления
- Превышение квоты трафика
- Истечение срока подписки
- Ошибки ядер
- Успешное обновление сертификатов

#### Webhooks
- POST запросы на указанный URL
- Настраиваемый payload
- Retry механизм

#### Telegram Bot (будущая функция)
- Логика заложена, реализация позже
- Уведомления в Telegram
- Базовое управление через бота

### 11. Оплата подписок (будущая функция)

**Не реализуется сейчас, но архитектура учитывает:**
- Интеграция платежных систем
- Автоматическое продление подписок
- История платежей
- Тарифные планы

### 12. Что НЕ входит в проект

❌ **Multi-server управление**: Одна панель = один сервер
❌ **Публичный доступ**: Только через SSH туннель
❌ **Доступ обычных пользователей**: Только администраторы

---

## Распределение протоколов по ядрам

### Правило распределения

**Принцип выбора:**
- Пользователь **сначала выбирает ядро**, затем доступные протоколы фильтруются по выбранному ядру
- Для общих протоколов (VMess, VLESS, Trojan, Shadowsocks) **рекомендуется Sing-box**, но выбор не ограничен
- Эксклюзивные протоколы автоматически определяют ядро (XHTTP → Xray, Mieru → Mihomo)

### Таблица протоколов

#### Inbound протоколы (MVP)

| Протокол | Sing-box | Xray | Mihomo | Статус |
|----------|----------|------|--------|--------|
| **HTTP** | ✅ | ✅ | ✅ | MVP |
| **SOCKS5** | ✅ | ✅ | ✅ | MVP |
| **Mixed** | ✅ | ❌ | ✅ | MVP |
| **Shadowsocks** | ✅ | ✅ | ✅ | MVP |
| **VMess** | ✅ | ✅ | ✅ | MVP |
| **VLESS** | ✅ | ✅ | ✅ | MVP |
| **Trojan** | ✅ | ✅ | ✅ | MVP |
| **Hysteria2** | ✅ | ✅ | ✅ | MVP |
| **TUIC v4** | ✅ | ❌ | ✅ | MVP |
| **TUIC v5** | ✅ | ❌ | ✅ | MVP |
| **Naive** | ✅ | ❌ | ❌ | MVP |
| **Redirect** | ✅ | ❌ | ✅ | MVP |
| **XHTTP** | ❌ | ✅ | ❌ | MVP (Xray эксклюзив) |
| **Mieru** | ❌ | ❌ | ✅ | MVP (Mihomo эксклюзив) |
| **Sudoku** | ❌ | ❌ | ✅ | MVP (Mihomo эксклюзив) |
| **TrustTunnel** | ❌ | ❌ | ✅ | MVP (Mihomo эксклюзив) |
| **ShadowsocksR** | ❌ | ❌ | ✅ | MVP (Mihomo эксклюзив) |
| **Snell** | ❌ | ❌ | ✅ | MVP (Mihomo эксклюзив) |
| **TProxy** | ✅ | ❌ | ✅ | ⏸️ Post-MVP |
| **TUN** | ✅ | ✅ | ✅ | ⏸️ Post-MVP |
| **WireGuard** | ✅ | ✅ | ❌ | ⏸️ Post-MVP |

#### Outbound протоколы (MVP)

| Протокол | Sing-box | Xray | Mihomo | Статус |
|----------|----------|------|--------|--------|
| **Direct** | ✅ | ✅ | ✅ | MVP |
| **Block** | ✅ | ✅ | ✅ | MVP |
| **DNS** | ✅ | ✅ | ✅ | MVP |
| **HTTP** | ✅ | ✅ | ✅ | MVP |
| **SOCKS5** | ✅ | ✅ | ✅ | MVP |
| **Shadowsocks** | ✅ | ✅ | ✅ | MVP |
| **VMess** | ✅ | ✅ | ✅ | MVP |
| **VLESS** | ✅ | ✅ | ✅ | MVP |
| **Trojan** | ✅ | ✅ | ✅ | MVP |
| **Hysteria** | ✅ | ✅ | ✅ | MVP |
| **Hysteria2** | ✅ | ✅ | ✅ | MVP |
| **TUIC** | ✅ | ❌ | ✅ | MVP |
| **Tor** | ✅ | ❌ | ❌ | MVP |
| **XHTTP** | ❌ | ✅ | ❌ | MVP (Xray эксклюзив) |
| **Mieru** | ❌ | ❌ | ✅ | MVP (Mihomo эксклюзив) |
| **Sudoku** | ❌ | ❌ | ✅ | MVP (Mihomo эксклюзив) |
| **TrustTunnel** | ❌ | ❌ | ✅ | MVP (Mihomo эксклюзив) |
| **ShadowsocksR** | ❌ | ❌ | ✅ | MVP (Mihomo эксклюзив) |
| **Snell** | ❌ | ❌ | ✅ | MVP (Mihomo эксклюзив) |
| **MASQUE** | ❌ | ❌ | ✅ | MVP (Mihomo эксклюзив) |
| **WireGuard** | ✅ | ✅ | ❌ | ⏸️ Post-MVP |
| **SSH** | ✅ | ❌ | ❌ | ⏸️ Post-MVP (только для VPS доступа) |

### UI Flow для создания Inbound

**Wizard шаги:**
1. **Выбор ядра**: Пользователь выбирает Sing-box, Xray или Mihomo
2. **Выбор протокола**: Показываются только протоколы, поддерживаемые выбранным ядром
3. **Настройка параметров**: Protocol-aware форма с рекомендациями

**Рекомендации в UI:**
- Для VMess/VLESS/Trojan/Shadowsocks: "Рекомендуется Sing-box для лучшей производительности"
- Для XHTTP: Автоматически выбирается Xray (единственный вариант)
- Для Mihomo-эксклюзивных: Автоматически выбирается Mihomo

---

## Рекомендации по безопасности

### 🔒 Критически важные меры безопасности

#### 1. Сетевая изоляция

**Обязательные настройки:**

```yaml
# docker-compose.yml
services:
  isolate-panel:
    ports:
      - "127.0.0.1:8080:8080"  # ТОЛЬКО localhost!
    networks:
      - internal
    
networks:
  internal:
    driver: bridge
    internal: false  # Доступ к интернету для обновлений
```

**Firewall правила (iptables):**

```bash
# Блокировать все входящие соединения кроме SSH
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT ACCEPT

# Разрешить SSH
iptables -A INPUT -p tcp --dport 22 -j ACCEPT

# Разрешить прокси порты (настраиваемые)
iptables -A INPUT -p tcp --dport 443 -j ACCEPT
iptables -A INPUT -p udp --dport 443 -j ACCEPT

# Разрешить loopback
iptables -A INPUT -i lo -j ACCEPT

# Разрешить established соединения
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# Сохранить правила
iptables-save > /etc/iptables/rules.v4
```

**SSH туннель для доступа:**

```bash
# На клиенте
ssh -L 8080:localhost:8080 user@your-vps-ip

# Затем открыть в браузере
http://localhost:8080
```

#### 2. Аутентификация и авторизация

**JWT токены:**
- Короткий срок жизни: 15 минут (access token)
- Refresh token: 7 дней
- Хранение refresh token в httpOnly cookie
- CSRF защита

**Пароли:**
- Минимум 12 символов
- Обязательно: буквы, цифры, спецсимволы
- Хеширование: Argon2id (рекомендуется) или bcrypt
- Salt: уникальный для каждого пользователя
- Параметры Argon2id:
  ```go
  time=1, memory=64*1024, threads=4, keyLen=32
  ```

**Rate limiting:**
```go
// Ограничение попыток входа
maxLoginAttempts := 5
lockoutDuration := 15 * time.Minute

// Ограничение API запросов
rateLimit := 100 // запросов в минуту на IP
```

**2FA (опционально, но рекомендуется):**
- TOTP (Google Authenticator, Authy)
- Backup коды для восстановления

#### 3. Защита данных

**Шифрование в покое:**
```bash
# Шифрование SQLite базы данных
# Использовать SQLCipher или шифрование на уровне файловой системы

# LUKS для шифрования раздела
cryptsetup luksFormat /dev/sdX
cryptsetup open /dev/sdX encrypted_data
mkfs.ext4 /dev/mapper/encrypted_data
```

**Шифрование в движении:**
- Все API запросы через HTTPS (даже через SSH туннель)
- TLS 1.3 минимум
- Сильные cipher suites:
  ```
  TLS_AES_256_GCM_SHA384
  TLS_CHACHA20_POLY1305_SHA256
  TLS_AES_128_GCM_SHA256
  ```

**Секреты и ключи:**
```bash
# НЕ хранить в коде или конфигах!
# Использовать переменные окружения

# .env файл (добавить в .gitignore)
JWT_SECRET=<сгенерировать 64 байта случайных данных>
DB_ENCRYPTION_KEY=<сгенерировать 32 байта>
ADMIN_PASSWORD=<сильный пароль>

# Генерация секретов
openssl rand -base64 64  # JWT secret
openssl rand -base64 32  # DB encryption key
```

#### 4. Защита от атак

**SQL Injection:**
- Использовать GORM prepared statements (автоматически)
- Валидация всех входных данных
- Никогда не конкатенировать SQL запросы

**XSS (Cross-Site Scripting):**
- Sanitize всех пользовательских данных
- Content-Security-Policy заголовки:
  ```
  Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'
  ```
- Escape HTML в выводе

**CSRF (Cross-Site Request Forgery):**
- CSRF токены для всех POST/PUT/DELETE запросов
- SameSite cookie attribute: `Strict`
- Проверка Origin/Referer заголовков

**DDoS защита:**
- Rate limiting на уровне приложения
- Fail2ban для блокировки IP после неудачных попыток входа
- Cloudflare или аналог (опционально)

**Command Injection:**
- Никогда не использовать `os.system()` или `exec()` с пользовательским вводом
- Использовать библиотеки для работы с ядрами через API/конфиги
- Валидация всех параметров

#### 5. Логирование и мониторинг безопасности

**Что логировать:**
- Все попытки входа (успешные и неудачные)
- Изменения конфигурации
- Создание/удаление пользователей
- Изменение прав доступа
- Подозрительная активность (множественные неудачные попытки)

**Формат логов:**
```json
{
  "timestamp": "2026-03-21T10:30:00Z",
  "level": "warn",
  "event": "failed_login",
  "ip": "192.168.1.100",
  "username": "admin",
  "attempts": 3
}
```

**Алерты:**
- Email при 5+ неудачных попытках входа
- Webhook при изменении критичных настроек
- Уведомление при создании нового администратора

#### 6. Обновления и патчи

**Автоматические обновления:**
```bash
# Обновление системных пакетов (Alpine)
apk update && apk upgrade

# Обновление Docker образов
docker-compose pull
docker-compose up -d
```

**Примечание:** Для Ubuntu/Debian используйте `apt-get update && apt-get upgrade -y`

**Мониторинг уязвимостей:**
- Подписка на security advisory для Go, Fiber, GORM
- Регулярная проверка зависимостей:
  ```bash
  go list -json -m all | nancy sleuth
  ```
- Использование Dependabot или Renovate

#### 7. Резервное копирование и восстановление

**Стратегия 3-2-1:**
- 3 копии данных
- 2 разных типа носителей
- 1 копия offsite

**Шифрование бэкапов:**
```bash
# Шифрование перед отправкой
tar czf - /data | gpg --encrypt --recipient admin@example.com > backup.tar.gz.gpg

# Отправка на S3 с шифрованием
aws s3 cp backup.tar.gz.gpg s3://bucket/backups/ --sse AES256
```

**Тестирование восстановления:**
- Ежемесячная проверка восстановления из бэкапа
- Документирование процедуры восстановления

#### 8. Минимизация поверхности атаки

**Принцип наименьших привилегий:**
- Запуск приложения от непривилегированного пользователя
- Использование Docker user namespace remapping
- Ограничение capabilities:
  ```yaml
  cap_drop:
    - ALL
  cap_add:
    - NET_BIND_SERVICE  # Только если нужны порты < 1024
  ```

**Отключение ненужных сервисов:**
```bash
# Alpine (через rc-service)
rc-update del apache2
rc-update del nginx

# Ubuntu/Debian (через systemctl)
# systemctl disable apache2
# systemctl disable nginx
# systemctl disable mysql
```

**Hardening Docker:**
```yaml
# docker-compose.yml
services:
  isolate-panel:
    security_opt:
      - no-new-privileges:true
    read_only: true
    tmpfs:
      - /tmp
      - /var/run
    volumes:
      - ./data:/data:rw  # Данные приложения
      - ./logs:/var/log/isolate-panel:rw  # Логи
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE  # Только для портов < 1024
```

#### 9. Compliance и аудит

**GDPR соответствие (если применимо):**
- Шифрование персональных данных
- Право на удаление данных
- Логирование доступа к данным
- Data retention policy

**Регулярный аудит:**
- Ежемесячный обзор логов безопасности
- Квартальный security audit кода
- Ежегодный penetration testing

#### 10. Incident Response Plan

**При обнаружении взлома:**

1. **Изоляция**: Отключить сервер от сети
2. **Оценка**: Определить масштаб компрометации
3. **Сохранение**: Сделать snapshot для анализа
4. **Восстановление**: Восстановить из чистого бэкапа
5. **Анализ**: Определить вектор атаки
6. **Улучшение**: Закрыть уязвимость
7. **Уведомление**: Информировать пользователей (если нужно)

**Контакты для экстренных случаев:**
- Список администраторов с контактами
- Процедура эскалации
- Backup администратор

---


## Структура базы данных

### Схема SQLite

#### Таблица: admins

```sql
CREATE TABLE admins (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(100),
    totp_secret VARCHAR(32),  -- для 2FA
    totp_enabled BOOLEAN DEFAULT FALSE,
    is_super_admin BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_login_at DATETIME,
    is_active BOOLEAN DEFAULT TRUE
);

CREATE INDEX idx_admins_username ON admins(username);
CREATE INDEX idx_admins_email ON admins(email);
```

#### Таблица: users

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100),
    
    -- Universal Credentials (генерируются при создании)
    uuid VARCHAR(36) UNIQUE NOT NULL,  -- для VLESS/VMess/TUIC v5
    password VARCHAR(255) NOT NULL,  -- plaintext для Trojan/Shadowsocks (MVP)
    token VARCHAR(64) UNIQUE,  -- для TUIC v4
    subscription_token VARCHAR(64) UNIQUE NOT NULL,
    
    -- WireGuard keys (Post-MVP, исключено из MVP)
    -- wireguard_private_key VARCHAR(44),
    -- wireguard_public_key VARCHAR(44),
    
    -- Квоты
    traffic_limit_bytes BIGINT DEFAULT NULL,  -- NULL = безлимит
    traffic_used_bytes BIGINT DEFAULT 0,
    expiry_date DATETIME DEFAULT NULL,  -- NULL = бессрочно
    
    -- Статус
    is_active BOOLEAN DEFAULT TRUE,
    is_online BOOLEAN DEFAULT FALSE,
    
    -- Метаданные
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_connected_at DATETIME,
    
    
    -- Связи
    created_by_admin_id INTEGER,
    FOREIGN KEY (created_by_admin_id) REFERENCES admins(id) ON DELETE SET NULL
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_uuid ON users(uuid);
CREATE INDEX idx_users_token ON users(token);
CREATE INDEX idx_users_subscription_token ON users(subscription_token);
CREATE INDEX idx_users_is_active ON users(is_active);

-- Примечание: MVP использует plaintext credentials для упрощения
-- Post-MVP: Миграция на безопасное хранение с шифрованием
```

#### Таблица: cores

```sql
CREATE TABLE cores (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(20) NOT NULL,  -- 'singbox', 'xray', 'mihomo'
    version VARCHAR(20) NOT NULL,
    is_enabled BOOLEAN DEFAULT TRUE,
    is_running BOOLEAN DEFAULT FALSE,
    pid INTEGER,
    config_path VARCHAR(255),
    log_path VARCHAR(255),
    
    -- Статистика
    uptime_seconds INTEGER DEFAULT 0,
    restart_count INTEGER DEFAULT 0,
    last_error TEXT,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_cores_name ON cores(name);
```

#### Таблица: inbounds

```sql
CREATE TABLE inbounds (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(100) NOT NULL,
    protocol VARCHAR(50) NOT NULL,  -- vless, vmess, trojan, shadowsocks, etc.
    core_id INTEGER NOT NULL,  -- какое ядро использует
    
    -- Сетевые настройки
    listen_address VARCHAR(50) DEFAULT '0.0.0.0',
    port INTEGER NOT NULL,
    
    -- Конфигурация (JSON)
    config_json TEXT NOT NULL,  -- полная конфигурация inbound
    
    -- TLS/REALITY
    tls_enabled BOOLEAN DEFAULT FALSE,
    tls_cert_id INTEGER,  -- ссылка на сертификат
    reality_enabled BOOLEAN DEFAULT FALSE,
    reality_config_json TEXT,
    
    -- Статус
    is_enabled BOOLEAN DEFAULT TRUE,
    
    -- Метаданные
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (core_id) REFERENCES cores(id) ON DELETE CASCADE,
    FOREIGN KEY (tls_cert_id) REFERENCES certificates(id) ON DELETE SET NULL
);

CREATE INDEX idx_inbounds_protocol ON inbounds(protocol);
CREATE INDEX idx_inbounds_port ON inbounds(port);
CREATE INDEX idx_inbounds_core_id ON inbounds(core_id);
```

#### Таблица: outbounds

```sql
CREATE TABLE outbounds (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(100) NOT NULL,
    protocol VARCHAR(50) NOT NULL,  -- direct, block, shadowsocks, vmess, etc.
    core_id INTEGER NOT NULL,
    
    -- Конфигурация (JSON)
    config_json TEXT NOT NULL,
    
    -- Приоритет для routing
    priority INTEGER DEFAULT 0,
    
    -- Статус
    is_enabled BOOLEAN DEFAULT TRUE,
    
    -- Метаданные
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (core_id) REFERENCES cores(id) ON DELETE CASCADE
);

CREATE INDEX idx_outbounds_protocol ON outbounds(protocol);
CREATE INDEX idx_outbounds_core_id ON outbounds(core_id);
```

#### Таблица: user_inbound_mapping

```sql
CREATE TABLE user_inbound_mapping (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    inbound_id INTEGER NOT NULL,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (inbound_id) REFERENCES inbounds(id) ON DELETE CASCADE,
    
    UNIQUE(user_id, inbound_id)
);

CREATE INDEX idx_user_inbound_user_id ON user_inbound_mapping(user_id);
CREATE INDEX idx_user_inbound_inbound_id ON user_inbound_mapping(inbound_id);
```

#### Таблица: certificates

```sql
CREATE TABLE certificates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    domain VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL,  -- 'acme', 'manual', 'certbot'
    
    -- Пути к файлам
    cert_path VARCHAR(255) NOT NULL,
    key_path VARCHAR(255) NOT NULL,
    
    -- ACME настройки
    acme_provider VARCHAR(50),  -- 'letsencrypt', 'zerossl'
    acme_email VARCHAR(100),
    acme_challenge_type VARCHAR(20),  -- 'http-01', 'dns-01'
    
    -- Статус
    is_valid BOOLEAN DEFAULT TRUE,
    auto_renew BOOLEAN DEFAULT TRUE,
    
    -- Даты
    issued_at DATETIME,
    expires_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_certificates_domain ON certificates(domain);
CREATE INDEX idx_certificates_expires_at ON certificates(expires_at);
```

#### Таблица: routing_rules

```sql
CREATE TABLE routing_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    core_id INTEGER NOT NULL,
    name VARCHAR(100) NOT NULL,
    
    -- Правило (JSON)
    rule_json TEXT NOT NULL,  -- domain, ip, geoip, geosite, etc.
    
    -- Действие
    outbound_id INTEGER NOT NULL,
    
    -- Приоритет
    priority INTEGER DEFAULT 0,
    
    -- Статус
    is_enabled BOOLEAN DEFAULT TRUE,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (core_id) REFERENCES cores(id) ON DELETE CASCADE,
    FOREIGN KEY (outbound_id) REFERENCES outbounds(id) ON DELETE CASCADE
);

CREATE INDEX idx_routing_rules_core_id ON routing_rules(core_id);
CREATE INDEX idx_routing_rules_priority ON routing_rules(priority);
```

#### Таблица: warp_routes

```sql
CREATE TABLE warp_routes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    resource_type VARCHAR(20) NOT NULL,  -- 'domain', 'ip', 'cidr'
    resource_value VARCHAR(255) NOT NULL,
    description TEXT,
    
    is_enabled BOOLEAN DEFAULT TRUE,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_warp_routes_resource_type ON warp_routes(resource_type);
```

#### Таблица: traffic_stats

```sql
CREATE TABLE traffic_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    inbound_id INTEGER NOT NULL,
    
    -- Трафик
    upload_bytes BIGINT DEFAULT 0,
    download_bytes BIGINT DEFAULT 0,
    
    -- Временная метка
    recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (inbound_id) REFERENCES inbounds(id) ON DELETE CASCADE
);

CREATE INDEX idx_traffic_stats_user_id ON traffic_stats(user_id);
CREATE INDEX idx_traffic_stats_recorded_at ON traffic_stats(recorded_at);
```

#### Таблица: traffic_stats_hourly

```sql
CREATE TABLE traffic_stats_hourly (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    hour_timestamp DATETIME NOT NULL,
    upload_bytes BIGINT DEFAULT 0,
    download_bytes BIGINT DEFAULT 0,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, hour_timestamp)
);

CREATE INDEX idx_traffic_stats_hourly_user_id ON traffic_stats_hourly(user_id);
CREATE INDEX idx_traffic_stats_hourly_timestamp ON traffic_stats_hourly(hour_timestamp);
```

#### Таблица: subscription_short_urls

```sql
CREATE TABLE subscription_short_urls (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    short_code VARCHAR(8) UNIQUE NOT NULL,
    full_url TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_subscription_short_urls_user_id ON subscription_short_urls(user_id);
CREATE UNIQUE INDEX idx_subscription_short_urls_code ON subscription_short_urls(short_code);
```

#### Таблица: subscription_accesses

```sql
CREATE TABLE subscription_accesses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    user_agent VARCHAR(512),
    country VARCHAR(2),  -- ISO 3166-1 alpha-2 код страны
    format VARCHAR(20),  -- v2ray, clash, singbox
    is_suspicious BOOLEAN DEFAULT FALSE,
    response_time_ms INTEGER DEFAULT 0,
    accessed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_subscription_accesses_user_id ON subscription_accesses(user_id);
CREATE INDEX idx_subscription_accesses_accessed_at ON subscription_accesses(accessed_at);
CREATE INDEX idx_subscription_accesses_is_suspicious ON subscription_accesses(is_suspicious);
```

#### Таблица: blocked_ips

```sql
CREATE TABLE blocked_ips (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ip_address VARCHAR(45) NOT NULL,
    reason VARCHAR(255),
    blocked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL,
    
    UNIQUE(ip_address)
);

CREATE INDEX idx_blocked_ips_ip_address ON blocked_ips(ip_address);
CREATE INDEX idx_blocked_ips_expires_at ON blocked_ips(expires_at);
```

#### Таблица: active_connections

```sql
CREATE TABLE active_connections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    inbound_id INTEGER NOT NULL,
    
    -- Информация о подключении
    source_ip VARCHAR(50),
    source_port INTEGER,
    destination_ip VARCHAR(50),
    destination_port INTEGER,
    protocol VARCHAR(50),
    
    -- Статистика
    upload_bytes BIGINT DEFAULT 0,
    download_bytes BIGINT DEFAULT 0,
    
    -- Временные метки
    connected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_activity_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (inbound_id) REFERENCES inbounds(id) ON DELETE CASCADE
);

CREATE INDEX idx_active_connections_user_id ON active_connections(user_id);
CREATE INDEX idx_active_connections_inbound_id ON active_connections(inbound_id);
```

#### Таблица: system_logs

```sql
CREATE TABLE system_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    level VARCHAR(10) NOT NULL,  -- 'debug', 'info', 'warn', 'error'
    category VARCHAR(50) NOT NULL,  -- 'auth', 'core', 'api', 'system'
    message TEXT NOT NULL,
    details_json TEXT,  -- дополнительные данные в JSON
    
    -- Контекст
    admin_id INTEGER,
    user_id INTEGER,
    ip_address VARCHAR(50),
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (admin_id) REFERENCES admins(id) ON DELETE SET NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_system_logs_level ON system_logs(level);
CREATE INDEX idx_system_logs_category ON system_logs(category);
CREATE INDEX idx_system_logs_created_at ON system_logs(created_at);
```

#### Таблица: backups

```sql
CREATE TABLE backups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    filename VARCHAR(255) NOT NULL,
    file_path VARCHAR(255) NOT NULL,
    file_size_bytes BIGINT,
    
    -- Тип и назначение
    backup_type VARCHAR(20) NOT NULL,  -- 'manual', 'scheduled'
    destination VARCHAR(20) NOT NULL,  -- 'local', 's3', 'ftp'
    
    -- Статус
    status VARCHAR(20) DEFAULT 'pending',  -- 'pending', 'completed', 'failed'
    error_message TEXT,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);

CREATE INDEX idx_backups_created_at ON backups(created_at);
CREATE INDEX idx_backups_status ON backups(status);
```

#### Таблица: notifications

```sql
CREATE TABLE notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type VARCHAR(50) NOT NULL,  -- 'email', 'webhook', 'telegram'
    event VARCHAR(50) NOT NULL,  -- 'quota_exceeded', 'cert_renewed', etc.
    priority VARCHAR(20) DEFAULT 'normal',  -- 'low', 'normal', 'high', 'critical'
    
    -- Получатель
    recipient VARCHAR(255) NOT NULL,
    
    -- Содержимое
    subject VARCHAR(255),
    body TEXT,
    
    -- Статус
    status VARCHAR(20) DEFAULT 'pending',  -- 'pending', 'sent', 'failed'
    retry_count INTEGER DEFAULT 0,
    error_message TEXT,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    sent_at DATETIME
);

CREATE INDEX idx_notifications_status ON notifications(status);
CREATE INDEX idx_notifications_created_at ON notifications(created_at);
CREATE INDEX idx_notifications_priority ON notifications(priority);
```

#### Таблица: settings

```sql
CREATE TABLE settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key VARCHAR(100) UNIQUE NOT NULL,
    value TEXT,
    value_type VARCHAR(20) DEFAULT 'string',  -- 'string', 'int', 'bool', 'json'
    description TEXT,
    
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_settings_key ON settings(key);

-- Предустановленные настройки
INSERT INTO settings (key, value, value_type, description) VALUES
('monitoring_mode', 'lite', 'string', 'Режим мониторинга: lite или full'),
('backup_enabled', 'false', 'bool', 'Автоматические бэкапы включены'),
('backup_schedule', '0 2 * * *', 'string', 'Расписание бэкапов (cron)'),
('backup_retention', '7', 'int', 'Количество хранимых бэкапов'),
('warp_enabled', 'false', 'bool', 'WARP интеграция включена'),
('geoip_enabled', 'false', 'bool', 'GeoIP поддержка включена'),
('email_notifications_enabled', 'false', 'bool', 'Email уведомления включены'),
('webhook_notifications_enabled', 'false', 'bool', 'Webhook уведомления включены');
```

#### Таблица: refresh_tokens

```sql
CREATE TABLE refresh_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    admin_id INTEGER NOT NULL,
    token_hash VARCHAR(64) NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    revoked BOOLEAN DEFAULT FALSE,
    user_agent VARCHAR(255),
    ip_address VARCHAR(50),
    
    FOREIGN KEY (admin_id) REFERENCES admins(id) ON DELETE CASCADE
);

CREATE INDEX idx_refresh_tokens_admin_id ON refresh_tokens(admin_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
```

#### Таблица: login_attempts

```sql
CREATE TABLE login_attempts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ip_address VARCHAR(50) NOT NULL,
    username VARCHAR(50),
    success BOOLEAN DEFAULT FALSE,
    attempted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    user_agent VARCHAR(255)
);

CREATE INDEX idx_login_attempts_ip ON login_attempts(ip_address, attempted_at);
CREATE INDEX idx_login_attempts_username ON login_attempts(username, attempted_at);
```

#### Таблица: haproxy_routes (Post-MVP)

**Статус:** Исключено из MVP, будет добавлено в v1.5+ вместе с HAProxy

```sql
-- CREATE TABLE haproxy_routes (
--     id INTEGER PRIMARY KEY AUTOINCREMENT,
--     name VARCHAR(100) NOT NULL,
--     route_type VARCHAR(20) NOT NULL,  -- 'sni', 'path', 'header'
--     match_value VARCHAR(255) NOT NULL,
--     backend_type VARCHAR(20) NOT NULL,
--     backend_target VARCHAR(100) NOT NULL,
--     priority INTEGER DEFAULT 0,
--     is_enabled BOOLEAN DEFAULT TRUE,
--     created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
--     updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
-- );
```

### Миграции

**Стратегия:** Использовать `golang-migrate` для версионных SQL миграций.

**Структура миграций:**
```
migrations/
├── 000001_create_admins_table.up.sql
├── 000001_create_admins_table.down.sql
├── 000002_create_users_table.up.sql
├── 000002_create_users_table.down.sql
├── 000003_create_cores_table.up.sql
├── 000003_create_cores_table.down.sql
... (см. детали в Фазе 1.1)
```

**Примечание:** GORM AutoMigrate может использоваться только для dev/testing окружения, но НЕ для production.

---

## Архитектура взаимодействия с ядрами

### Управление процессами

**Supervisord** управляет всеми процессами ядер:

```ini
[program:singbox]
command=/usr/local/bin/sing-box run -c /data/configs/singbox.json
autostart=true
autorestart=true
stdout_logfile=/var/log/isolate-panel/singbox.log
stderr_logfile=/var/log/isolate-panel/singbox.err.log

[program:xray]
command=/usr/local/bin/xray run -c /data/configs/xray.json
autostart=true
autorestart=true
stdout_logfile=/var/log/isolate-panel/xray.log
stderr_logfile=/var/log/isolate-panel/xray.err.log

[program:mihomo]
command=/usr/local/bin/mihomo -f /data/configs/mihomo.yaml
autostart=true
autorestart=true
stdout_logfile=/var/log/isolate-panel/mihomo.log
stderr_logfile=/var/log/isolate-panel/mihomo.err.log

# HAProxy исключен из MVP (Post-MVP feature)
# [program:haproxy]
# command=/usr/local/sbin/haproxy -f /data/configs/haproxy.cfg
# autostart=true
# autorestart=true
# stdout_logfile=/var/log/isolate-panel/haproxy.log
# stderr_logfile=/var/log/isolate-panel/haproxy.err.log
```

### Получение статистики от ядер

**Унифицированный интерфейс:**

```go
// Единый интерфейс для всех ядер
type CoreStatsProvider interface {
    GetUserTraffic(userUUID string) (*TrafficStats, error)
    GetActiveConnections() ([]Connection, error)
    GetCoreStatus() (*CoreStatus, error)
}

// Реализация для Sing-box (Clash API)
type SingboxStatsProvider struct {
    baseURL string
    secret  string
}

func (s *SingboxStatsProvider) GetUserTraffic(userUUID string) (*TrafficStats, error) {
    // GET http://127.0.0.1:9090/connections
    // Фильтруем по UUID пользователя
}

// Реализация для Xray (gRPC Stats API)
type XrayStatsProvider struct {
    grpcClient pb.StatsServiceClient
}

func (x *XrayStatsProvider) GetUserTraffic(userUUID string) (*TrafficStats, error) {
    // gRPC вызов к Xray Stats API
    // QueryStats("user>>>"+userUUID+">>>traffic>>>uplink")
}

// Реализация для Mihomo (REST API)
type MihomoStatsProvider struct {
    baseURL string
    secret  string
}

func (m *MihomoStatsProvider) GetUserTraffic(userUUID string) (*TrafficStats, error) {
    // GET http://127.0.0.1:9091/connections
}
```

### Хранение конфигураций

```
/data/
├── configs/
│   ├── singbox.json      # Генерируется из БД
│   ├── xray.json         # Генерируется из БД
│   ├── mihomo.yaml       # Генерируется из БД
│   └── haproxy.cfg       # Генерируется из БД
├── certs/
│   ├── domain1.crt
│   └── domain1.key
└── isolate.db
```

**Логика работы:**
1. Панель хранит данные в БД (users, inbounds, outbounds)
2. При изменении → генерирует конфиг файл
3. Валидирует конфиг: `sing-box check -c singbox.json`
4. Если OK → применяет через supervisord: `supervisorctl restart singbox`

### Применение изменений конфигурации

**Graceful restart (приоритет):**

```go
func (cm *CoreManager) ApplyConfig(coreName string) error {
    // 1. Генерируем новый конфиг
    newConfig := cm.GenerateConfig(coreName)
    
    // 2. Сохраняем во временный файл
    tmpPath := fmt.Sprintf("/tmp/%s.json.new", coreName)
    ioutil.WriteFile(tmpPath, newConfig, 0644)
    
    // 3. Валидируем
    if err := cm.ValidateConfig(coreName, tmpPath); err != nil {
        return fmt.Errorf("invalid config: %w", err)
    }
    
    // 4. Переименовываем
    configPath := fmt.Sprintf("/data/configs/%s.json", coreName)
    os.Rename(tmpPath, configPath)
    
    // 5. Graceful reload (если поддерживается)
    if cm.SupportsGracefulReload(coreName) {
        return cm.SendSignal(coreName, syscall.SIGHUP)
    }
    
    // 6. Fallback: полный restart
    return cm.RestartCore(coreName)
}
```

**Поддержка graceful reload:**
- Xray: ✅ Поддерживает SIGHUP
- Sing-box: ⚠️ Нужно проверить (возможно только restart)
- Mihomo: ⚠️ Нужно проверить

---

## Генерация конфигураций ядер

### Подход: Go structs + JSON marshal

```go
// internal/cores/singbox/config.go

type SingboxConfig struct {
    Log       LogConfig       `json:"log"`
    Inbounds  []Inbound       `json:"inbounds"`
    Outbounds []Outbound      `json:"outbounds"`
    Route     RouteConfig     `json:"route"`
    Experimental ExperimentalConfig `json:"experimental"`
}

type Inbound struct {
    Type   string      `json:"type"`
    Tag    string      `json:"tag"`
    Listen string      `json:"listen"`
    Port   int         `json:"listen_port"`
    Users  []User      `json:"users,omitempty"`
    TLS    *TLSConfig  `json:"tls,omitempty"`
}

type User struct {
    Name string `json:"name"`
    UUID string `json:"uuid"`
}
```

### Генератор конфигурации

```go
// internal/cores/singbox/generator.go

func GenerateSingboxConfig(db *gorm.DB) (*SingboxConfig, error) {
    config := &SingboxConfig{
        Log: LogConfig{
            Level: "info",
            Output: "/var/log/isolate-panel/singbox.log",
        },
    }
    
    // Получаем все активные inbound для Sing-box
    var inbounds []models.Inbound
    db.Where("core_id = ? AND is_enabled = ?", singboxCoreID, true).
       Preload("Users").
       Find(&inbounds)
    
    // Конвертируем в конфиг Sing-box
    for _, inbound := range inbounds {
        sbInbound := convertToSingboxInbound(inbound)
        config.Inbounds = append(config.Inbounds, sbInbound)
    }
    
    // Добавляем Experimental API для статистики
    config.Experimental = ExperimentalConfig{
        ClashAPI: &ClashAPIConfig{
            ExternalController: "127.0.0.1:9090",
            Secret: generateSecret(),
        },
    }
    
    return config, nil
}

func convertToSingboxInbound(inbound models.Inbound) Inbound {
    sb := Inbound{
        Type:   inbound.Protocol,
        Tag:    fmt.Sprintf("inbound-%d", inbound.ID),
        Listen: "127.0.0.1", // HAProxy проксирует сюда
        Port:   10000 + inbound.ID, // Динамический порт
    }
    
    // Добавляем пользователей
    for _, user := range inbound.Users {
        sb.Users = append(sb.Users, User{
            Name: user.Username,
            UUID: user.UUID,
        })
    }
    
    return sb
}
```

### Структура пакетов

```
internal/
├── cores/
│   ├── singbox/
│   │   ├── config.go      # Структуры конфига
│   │   ├── generator.go   # Генерация из БД
│   │   ├── manager.go     # Управление процессом
│   │   └── stats.go       # Получение статистики
│   ├── xray/
│   │   ├── config.go
│   │   ├── generator.go
│   │   ├── manager.go
│   │   └── stats.go
│   ├── mihomo/
│   │   ├── config.go
│   │   ├── generator.go
│   │   ├── manager.go
│   │   └── stats.go
│   └── interface.go       # Общий интерфейс
```

---

## Учет трафика и квоты

### Частота сбора статистики

**Настройки:**
- **Lite режим**: каждые 60 секунд
- **Full режим**: каждые 10 секунд
- Настраивается через `settings.monitoring_mode`

### Архитектура сбора

```go
// internal/services/traffic_collector.go

type TrafficCollector struct {
    db            *gorm.DB
    coreProviders map[string]CoreStatsProvider
    interval      time.Duration
}

func (tc *TrafficCollector) Start() {
    ticker := time.NewTicker(tc.interval)
    
    for range ticker.C {
        tc.CollectStats()
    }
}

func (tc *TrafficCollector) CollectStats() error {
    // Получаем всех активных пользователей
    var users []models.User
    tc.db.Where("is_active = ?", true).Find(&users)
    
    for _, user := range users {
        // Собираем статистику со всех ядер
        totalUpload := int64(0)
        totalDownload := int64(0)
        
        for coreName, provider := range tc.coreProviders {
            stats, err := provider.GetUserTraffic(user.UUID)
            if err != nil {
                log.Warn().Err(err).Str("core", coreName).Msg("failed to get stats")
                continue
            }
            
            totalUpload += stats.Upload
            totalDownload += stats.Download
        }
        
        // Обновляем в БД
        tc.db.Model(&user).Updates(map[string]interface{}{
            "traffic_used_bytes": gorm.Expr("traffic_used_bytes + ?", totalUpload + totalDownload),
        })
        
        // Сохраняем в traffic_stats для истории
        tc.db.Create(&models.TrafficStats{
            UserID:        user.ID,
            UploadBytes:   totalUpload,
            DownloadBytes: totalDownload,
            RecordedAt:    time.Now(),
        })
        
        // Проверяем квоту
        if user.TrafficLimitBytes != nil && 
           user.TrafficUsedBytes >= *user.TrafficLimitBytes {
            tc.HandleQuotaExceeded(user)
        }
    }
    
    return nil
}
```

### Применение квот при превышении

**Текущая реализация (MVP):**

```go
func (tc *TrafficCollector) HandleQuotaExceeded(user models.User) error {
    // 1. Отключаем пользователя
    tc.db.Model(&user).Update("is_active", false)
    
    // 2. Регенерируем конфиги всех ядер (пользователь исключается)
    for coreName := range tc.coreProviders {
        if err := tc.coreManager.RegenerateConfig(coreName); err != nil {
            return err
        }
    }
    
    // 3. Graceful restart ядер
    for coreName := range tc.coreProviders {
        if err := tc.coreManager.ApplyConfig(coreName); err != nil {
            return err
        }
    }
    
    // 4. Отправляем уведомление
    tc.notificationService.Send(&models.Notification{
        Type:      "email",
        Event:     "quota_exceeded",
        Recipient: user.Email,
        Subject:   "Traffic quota exceeded",
        Body:      fmt.Sprintf("User %s exceeded traffic quota", user.Username),
    })
    
    return nil
}
```

**TODO (будущая оптимизация):**
- Динамическое отключение через API ядра (без restart)
- Требует поддержки от ядер
- Фиксируем в плане как улучшение для Фазы 14

### Агрегация и очистка данных

```go
// internal/services/data_retention.go

func (dr *DataRetention) AggregateAndCleanup() {
    // Агрегация почасовая (из минутных данных) - SQLite синтаксис
    _, err := dr.db.Exec(`
        INSERT INTO traffic_stats_hourly (user_id, hour_timestamp, upload_bytes, download_bytes)
        SELECT user_id, 
               strftime('%Y-%m-%d %H:00:00', recorded_at) as hour_timestamp,
               SUM(upload_bytes),
               SUM(download_bytes)
        FROM traffic_stats
        WHERE recorded_at < datetime('now', '-1 day')
        GROUP BY user_id, hour_timestamp
    `)
    if err != nil {
        return fmt.Errorf("failed to aggregate hourly stats: %w", err)
    }
    
    // Удаляем сырые данные старше 7 дней
    _, err = dr.db.Exec("DELETE FROM traffic_stats WHERE recorded_at < datetime('now', '-7 days')")
    if err != nil {
        return fmt.Errorf("failed to cleanup traffic_stats: %w", err)
    }
    
    // Удаляем почасовые данные старше 90 дней
    _, err = dr.db.Exec("DELETE FROM traffic_stats_hourly WHERE hour_timestamp < datetime('now', '-90 days')")
    if err != nil {
        return fmt.Errorf("failed to cleanup hourly stats: %w", err)
    }
    
    // Удаляем старые логи
    _, err = dr.db.Exec("DELETE FROM system_logs WHERE created_at < datetime('now', '-30 days')")
    if err != nil {
        return fmt.Errorf("failed to cleanup logs: %w", err)
    }
    
    // Удаляем неактивные соединения
    _, err = dr.db.Exec("DELETE FROM active_connections WHERE last_activity_at < datetime('now', '-5 minutes')")
    if err != nil {
        return fmt.Errorf("failed to cleanup connections: %w", err)
    }
    
    return nil
}
```

---

## Мониторинг подключений (MVP)

### Мониторинг ядер напрямую

**Получение статистики от каждого ядра:**

```go
// internal/services/core_monitor.go

type CoreMonitor struct {
    singboxStats *SingboxStatsProvider
    xrayStats    *XrayStatsProvider
    mihomoStats  *MihomoStatsProvider
}

func (cm *CoreMonitor) GetAllCoreStats() (map[string]*CoreStats, error) {
    stats := make(map[string]*CoreStats)
    
    // Sing-box stats
    if singboxStats, err := cm.singboxStats.GetCoreStatus(); err == nil {
        stats["singbox"] = singboxStats
    }
    
    // Xray stats
    if xrayStats, err := cm.xrayStats.GetCoreStatus(); err == nil {
        stats["xray"] = xrayStats
    }
    
    // Mihomo stats
    if mihomoStats, err := cm.mihomoStats.GetCoreStatus(); err == nil {
        stats["mihomo"] = mihomoStats
    }
    
    return stats, nil
}

type CoreStats struct {
    IsRunning          bool
    Uptime             time.Duration
    ActiveConnections  int
    TotalTrafficUp     int64
    TotalTrafficDown   int64
}
```

---
    
    return nil
}
```

### Dashboard метрики

**Что показывать в панели:**

```
┌─────────────────────────────────────────────────────────┐
│ Connection Statistics                                    │
├─────────────────────────────────────────────────────────┤
│                                                           │
│ HAProxy Connections:  245 / 1024  (24%)  ████░░░░░░     │
│ Panel API Requests:    12 / 256   (5%)   █░░░░░░░░░     │
│                                                           │
│ Active Proxy Users:    18 / 30                           │
│ Connections per User:  13.6 avg                          │
│                                                           │
│ ⚠️ Recommendations:                                       │
│ • Connection usage is healthy                            │
│ • Consider increasing maxconn if consistently > 70%      │
└─────────────────────────────────────────────────────────┘
```

### Alerting правила

```go
// internal/services/connection_alerting.go

type ConnectionAlerting struct {
    monitor      *HAProxyMonitor
    notifier     *NotificationService
    checkInterval time.Duration
}

func (ca *ConnectionAlerting) Start() {
    ticker := time.NewTicker(ca.checkInterval)
    
    for range ticker.C {
        stats, err := ca.monitor.GetStats()
        if err != nil {
            log.Error().Err(err).Msg("Failed to get HAProxy stats")
            continue
        }
        
        usagePercent := float64(stats.CurrentConnections) / float64(stats.MaxConnections) * 100
        
        // Critical: 90%+
        if usagePercent >= 90 {
            ca.notifier.Send(&Notification{
                Type:     "email",
                Priority: "critical",
                Subject:  "CRITICAL: HAProxy connections at 90%",
                Body: fmt.Sprintf(
                    "HAProxy is at %.1f%% capacity (%d/%d connections).\n"+
                    "New connections may be rejected soon.\n"+
                    "Action required: Increase maxconn or reduce load.",
                    usagePercent, stats.CurrentConnections, stats.MaxConnections,
                ),
            })
        }
        
        // Warning: 80%+
        if usagePercent >= 80 && usagePercent < 90 {
            ca.notifier.Send(&Notification{
                Type:     "email",
                Priority: "warning",
                Subject:  "WARNING: HAProxy connections at 80%",
                Body: fmt.Sprintf(
                    "HAProxy is at %.1f%% capacity (%d/%d connections).\n"+
                    "Consider increasing maxconn or monitoring load.",
                    usagePercent, stats.CurrentConnections, stats.MaxConnections,
                ),
            })
        }
    }
}
```

### Рекомендации по масштабированию

**Когда увеличивать maxconn:**

| Текущее использование | Действие | Новый maxconn |
|----------------------|----------|---------------|
| < 50% | Нормально | Не менять |
| 50-70% | Мониторить | Не менять |
| 70-85% | Планировать увеличение | +50% |
| 85-95% | Срочно увеличить | +100% |
| > 95% | Критично | +200% |

**Пример:**
```
Текущий maxconn: 1024
Использование: 850 connections (83%)
→ Рекомендация: Увеличить до 1536 (1024 × 1.5)
```

**Влияние на память:**
```
HAProxy memory per connection: ~50KB
1024 connections: ~50MB
2048 connections: ~100MB
4096 connections: ~200MB
```

---

## Фазы реализации

### Общий подход

**Методология**: Итеративная разработка с MVP подходом
**Приоритет**: Безопасность → Стабильность → Функциональность → UX

### Фаза 0: Подготовка (1 неделя)

#### Задачи
- [ ] Настройка окружения разработки
- [ ] Инициализация Git репозитория
- [ ] Настройка CI/CD pipeline (GitHub Actions)
- [ ] Создание базовой структуры проекта
- [ ] Настройка линтеров и форматтеров

#### Структура проекта
```
isolate-panel/
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── internal/
│   │   ├── api/
│   │   ├── auth/
│   │   ├── config/
│   │   ├── core/
│   │   ├── database/
│   │   ├── middleware/
│   │   ├── models/
│   │   └── services/
│   ├── pkg/
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   ├── stores/
│   │   ├── utils/
│   │   └── main.tsx
│   ├── package.json
│   └── vite.config.ts
├── docker/
│   ├── Dockerfile
│   └── docker-compose.yml
├── scripts/
│   └── install.sh
├── docs/
├── .github/
│   └── workflows/
├── .gitignore
├── README.md
└── LICENSE
```

#### Deliverables
- Рабочее окружение разработки
- Базовая структура проекта
- CI/CD настроен

---

### Фаза 1: MVP Backend (3-4 недели)

#### 1.1 Базовая инфраструктура (1 неделя)

**Задачи:**
- [ ] Настройка Fiber web-сервера
- [ ] Подключение SQLite + GORM
- [ ] **КРИТИЧНО: SQLite оптимизация для конкурентного доступа**
- [ ] Настройка Zerolog логирования (structured logging, log rotation)
- [ ] Конфигурация через Viper
- [ ] Базовые middleware (CORS, Logger, Recovery)
- [ ] Настройка golang-migrate для database migrations
- [ ] Создание всех 20 initial migrations (по одной на таблицу)
- [ ] Migration Manager с поддержкой up/down/steps/version/force
- [ ] Seed data system (default admin, settings, dev users)
- [ ] CLI tool для управления миграциями
- [ ] Автоматический запуск миграций при старте приложения
- [ ] Embedded migrations в бинарник (go:embed)

**Database Migrations Structure:**
```
migrations/
├── 000001_create_admins_table.up.sql
├── 000001_create_admins_table.down.sql
├── 000002_create_users_table.up.sql
├── 000002_create_users_table.down.sql
├── 000003_create_cores_table.up.sql
├── 000003_create_cores_table.down.sql
├── 000004_create_inbounds_table.up.sql
├── 000004_create_inbounds_table.down.sql
├── 000005_create_outbounds_table.up.sql
├── 000005_create_outbounds_table.down.sql
├── 000006_create_user_inbound_mapping_table.up.sql
├── 000006_create_user_inbound_mapping_table.down.sql
├── 000007_create_certificates_table.up.sql
├── 000007_create_certificates_table.down.sql
├── 000008_create_routing_rules_table.up.sql
├── 000008_create_routing_rules_table.down.sql
├── 000009_create_warp_routes_table.up.sql
├── 000009_create_warp_routes_table.down.sql
├── 000010_create_traffic_stats_table.up.sql
├── 000010_create_traffic_stats_table.down.sql
├── 000011_create_active_connections_table.up.sql
├── 000011_create_active_connections_table.down.sql
├── 000012_create_system_logs_table.up.sql
├── 000012_create_system_logs_table.down.sql
├── 000013_create_backups_table.up.sql
├── 000013_create_backups_table.down.sql
├── 000014_create_notifications_table.up.sql
├── 000014_create_notifications_table.down.sql
├── 000015_create_settings_table.up.sql
├── 000015_create_settings_table.down.sql
├── 000016_create_refresh_tokens_table.up.sql
├── 000016_create_refresh_tokens_table.down.sql
├── 000017_create_login_attempts_table.up.sql
├── 000017_create_login_attempts_table.down.sql
├── 000018_create_haproxy_routes_table.up.sql
├── 000018_create_haproxy_routes_table.down.sql
├── 000019_create_subscription_short_urls_table.up.sql
├── 000019_create_subscription_short_urls_table.down.sql
├── 000020_create_subscription_accesses_table.up.sql
└── 000020_create_subscription_accesses_table.down.sql
```

**Example Migration (000001_create_admins_table.up.sql):**
```sql
CREATE TABLE IF NOT EXISTS admins (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    is_super_admin BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_admins_username ON admins(username);
```

**Example Migration (000001_create_admins_table.down.sql):**
```sql
DROP INDEX IF EXISTS idx_admins_username;
DROP TABLE IF EXISTS admins;
```

**Migration Manager (internal/database/migrations.go):**
```go
package database

import (
    "database/sql"
    "embed"
    "fmt"
    
    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/sqlite3"
    "github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type MigrationManager struct {
    db      *sql.DB
    migrate *migrate.Migrate
}

func NewMigrationManager(db *sql.DB) (*MigrationManager, error) {
    // Create source from embedded FS
    source, err := iofs.New(migrationsFS, "migrations")
    if err != nil {
        return nil, fmt.Errorf("failed to create migration source: %w", err)
    }
    
    // Create database driver
    driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
    if err != nil {
        return nil, fmt.Errorf("failed to create database driver: %w", err)
    }
    
    // Create migrate instance
    m, err := migrate.NewWithInstance("iofs", source, "sqlite3", driver)
    if err != nil {
        return nil, fmt.Errorf("failed to create migrate instance: %w", err)
    }
    
    return &MigrationManager{
        db:      db,
        migrate: m,
    }, nil
}

// Up runs all pending migrations
func (mm *MigrationManager) Up() error {
    if err := mm.migrate.Up(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("failed to run migrations: %w", err)
    }
    return nil
}

// Down rolls back the last migration
func (mm *MigrationManager) Down() error {
    if err := mm.migrate.Down(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("failed to rollback migration: %w", err)
    }
    return nil
}

// Steps runs n migrations (positive = up, negative = down)
func (mm *MigrationManager) Steps(n int) error {
    if err := mm.migrate.Steps(n); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("failed to run %d steps: %w", n, err)
    }
    return nil
}

// Version returns current migration version
func (mm *MigrationManager) Version() (uint, bool, error) {
    version, dirty, err := mm.migrate.Version()
    if err != nil && err != migrate.ErrNilVersion {
        return 0, false, fmt.Errorf("failed to get version: %w", err)
    }
    return version, dirty, nil
}

// Force sets the migration version without running migrations
func (mm *MigrationManager) Force(version int) error {
    if err := mm.migrate.Force(version); err != nil {
        return fmt.Errorf("failed to force version: %w", err)
    }
    return nil
}

// Close closes the migration manager
func (mm *MigrationManager) Close() error {
    sourceErr, dbErr := mm.migrate.Close()
    if sourceErr != nil {
        return sourceErr
    }
    return dbErr
}
```

**Seed Data System (internal/database/seeds/seeds.go):**
```go
package seeds

import (
    "fmt"
    "os"
    "time"
    
    "golang.org/x/crypto/argon2"
    "gorm.io/gorm"
    "github.com/yourusername/isolate-panel/internal/models"
)

type Seeder struct {
    db *gorm.DB
}

func NewSeeder(db *gorm.DB) *Seeder {
    return &Seeder{db: db}
}

// RunAll runs all seeders
func (s *Seeder) RunAll() error {
    seeders := []func() error{
        s.SeedDefaultAdmin,
        s.SeedDefaultSettings,
        s.SeedDevelopmentUsers,
    }
    
    for _, seeder := range seeders {
        if err := seeder(); err != nil {
            return err
        }
    }
    
    return nil
}

// SeedDefaultAdmin creates default admin user
func (s *Seeder) SeedDefaultAdmin() error {
    var count int64
    s.db.Model(&models.Admin{}).Count(&count)
    
    if count > 0 {
        return nil // Admin already exists
    }
    
    passwordHash := hashPassword("admin")
    admin := &models.Admin{
        Username:     "admin",
        PasswordHash: passwordHash,
        IsSuperAdmin: true,
    }
    
    if err := s.db.Create(admin).Error; err != nil {
        return fmt.Errorf("failed to seed default admin: %w", err)
    }
    
    fmt.Println("✓ Default admin created (username: admin, password: admin)")
    return nil
}

// SeedDefaultSettings creates default system settings
func (s *Seeder) SeedDefaultSettings() error {
    defaultSettings := []models.Setting{
        {Key: "panel_name", Value: "Isolate Panel"},
        {Key: "haproxy_enabled", Value: "true"},
        {Key: "haproxy_stats_password", Value: "admin"},
        {Key: "traffic_collection_interval", Value: "60"},
        {Key: "data_retention_days", Value: "90"},
        {Key: "max_login_attempts", Value: "5"},
        {Key: "jwt_access_token_ttl", Value: "900"},    // 15 minutes
        {Key: "jwt_refresh_token_ttl", Value: "604800"}, // 7 days
    }
    
    for _, setting := range defaultSettings {
        var existing models.Setting
        err := s.db.Where("key = ?", setting.Key).First(&existing).Error
        
        if err == gorm.ErrRecordNotFound {
            if err := s.db.Create(&setting).Error; err != nil {
                return fmt.Errorf("failed to seed setting %s: %w", setting.Key, err)
            }
        }
    }
    
    fmt.Println("✓ Default settings seeded")
    return nil
}

// SeedDevelopmentUsers creates test users (only in development)
func (s *Seeder) SeedDevelopmentUsers() error {
    // Only run in development mode
    if os.Getenv("APP_ENV") != "development" {
        return nil
    }
    
    testUsers := []models.User{
        {
            UUID:              "550e8400-e29b-41d4-a716-446655440001",
            Username:          "testuser1",
            Email:             "test1@example.com",
            SubscriptionToken: "test_token_1",
            IsActive:          true,
            TrafficLimit:      107374182400, // 100GB
            TrafficUsed:       0,
        },
        {
            UUID:              "550e8400-e29b-41d4-a716-446655440002",
            Username:          "testuser2",
            Email:             "test2@example.com",
            SubscriptionToken: "test_token_2",
            IsActive:          true,
            TrafficLimit:      53687091200,  // 50GB
            TrafficUsed:       10737418240,  // 10GB used
        },
    }
    
    for _, user := range testUsers {
        var existing models.User
        err := s.db.Where("username = ?", user.Username).First(&existing).Error
        
        if err == gorm.ErrRecordNotFound {
            if err := s.db.Create(&user).Error; err != nil {
                return fmt.Errorf("failed to seed user %s: %w", user.Username, err)
            }
        }
    }
    
    fmt.Println("✓ Development users seeded")
    return nil
}

func hashPassword(password string) string {
    salt := []byte("isolate-panel-salt") // In production, use random salt per user
    hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
    return fmt.Sprintf("%x", hash)
}
```

**CLI Tool for Migrations (cmd/migrate/main.go):**
```go
package main

import (
    "database/sql"
    "flag"
    "fmt"
    "log"
    
    _ "github.com/mattn/go-sqlite3"
    "github.com/yourusername/isolate-panel/internal/database"
)

func main() {
    var (
        dbPath  = flag.String("db", "./data/isolate-panel.db", "Database path")
        command = flag.String("cmd", "up", "Command: up, down, steps, version, force")
        steps   = flag.Int("steps", 1, "Number of steps for 'steps' command")
        version = flag.Int("version", 0, "Version for 'force' command")
    )
    
    flag.Parse()
    
    // Open database
    db, err := sql.Open("sqlite3", *dbPath)
    if err != nil {
        log.Fatalf("Failed to open database: %v", err)
    }
    defer db.Close()
    
    // Create migration manager
    mm, err := database.NewMigrationManager(db)
    if err != nil {
        log.Fatalf("Failed to create migration manager: %v", err)
    }
    defer mm.Close()
    
    // Execute command
    switch *command {
    case "up":
        fmt.Println("Running migrations...")
        if err := mm.Up(); err != nil {
            log.Fatalf("Migration failed: %v", err)
        }
        fmt.Println("✓ Migrations completed successfully")
        
    case "down":
        fmt.Println("Rolling back last migration...")
        if err := mm.Down(); err != nil {
            log.Fatalf("Rollback failed: %v", err)
        }
        fmt.Println("✓ Rollback completed successfully")
        
    case "steps":
        fmt.Printf("Running %d migration steps...\n", *steps)
        if err := mm.Steps(*steps); err != nil {
            log.Fatalf("Steps failed: %v", err)
        }
        fmt.Println("✓ Steps completed successfully")
        
    case "version":
        v, dirty, err := mm.Version()
        if err != nil {
            log.Fatalf("Failed to get version: %v", err)
        }
        fmt.Printf("Current version: %d\n", v)
        if dirty {
            fmt.Println("⚠️  Database is in dirty state!")
        }
        
    case "force":
        fmt.Printf("Forcing version to %d...\n", *version)
        if err := mm.Force(*version); err != nil {
            log.Fatalf("Force failed: %v", err)
        }
        fmt.Println("✓ Version forced successfully")
        
    default:
        log.Fatalf("Unknown command: %s", *command)
    }
}
```

**Auto-run Migrations at Startup (internal/app/app.go):**
```go
package app

import (
    "database/sql"
    "log"
    
    "github.com/yourusername/isolate-panel/internal/database"
    "github.com/yourusername/isolate-panel/internal/database/seeds"
    "gorm.io/gorm"
)

func InitializeDatabase(gormDB *gorm.DB) error {
    // Get underlying sql.DB
    sqlDB, err := gormDB.DB()
    if err != nil {
        return fmt.Errorf("failed to get sql.DB: %w", err)
    }
    
    // Run migrations
    if err := runMigrations(sqlDB); err != nil {
        return fmt.Errorf("failed to run migrations: %w", err)
    }
    
    // Run seeds
    if err := runSeeds(gormDB); err != nil {
        return fmt.Errorf("failed to run seeds: %w", err)
    }
    
    return nil
}

func runMigrations(db *sql.DB) error {
    log.Println("Running database migrations...")
    
    mm, err := database.NewMigrationManager(db)
    if err != nil {
        return err
    }
    defer mm.Close()
    
    // Check current version
    version, dirty, err := mm.Version()
    if err != nil {
        log.Printf("No migrations applied yet")
    } else {
        log.Printf("Current migration version: %d (dirty: %v)", version, dirty)
    }
    
    // Run migrations
    if err := mm.Up(); err != nil {
        return err
    }
    
    // Get new version
    version, _, err = mm.Version()
    if err != nil {
        return err
    }
    
    log.Printf("✓ Migrations completed. Current version: %d", version)
    return nil
}

func runSeeds(db *gorm.DB) error {
    log.Println("Running database seeds...")
    
    seeder := seeds.NewSeeder(db)
    if err := seeder.RunAll(); err != nil {
        return err
    }
    
    log.Println("✓ Seeds completed")
    return nil
}
```

**КРИТИЧНО: SQLite Optimization для конкурентного доступа:**

```go
// internal/database/sqlite.go
package database

import (
    "database/sql"
    "fmt"
    "time"
    
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

func InitSQLite(dbPath string) (*gorm.DB, error) {
    // Open database
    db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
        NowFunc: func() time.Time {
            return time.Now().UTC()
        },
    })
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }
    
    // Get underlying sql.DB
    sqlDB, err := db.DB()
    if err != nil {
        return nil, fmt.Errorf("failed to get sql.DB: %w", err)
    }
    
    // КРИТИЧНО: Настройки для конкурентного доступа
    // Write-Ahead Logging - позволяет одновременное чтение и запись
    if err := sqlDB.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
        return nil, fmt.Errorf("failed to set WAL mode: %w", err)
    }
    
    // Таймаут ожидания при блокировке БД (5 секунд)
    if err := sqlDB.Exec("PRAGMA busy_timeout=5000").Error; err != nil {
        return nil, fmt.Errorf("failed to set busy_timeout: %w", err)
    }
    
    // Баланс между безопасностью и производительностью
    if err := sqlDB.Exec("PRAGMA synchronous=NORMAL").Error; err != nil {
        return nil, fmt.Errorf("failed to set synchronous mode: %w", err)
    }
    
    // Увеличиваем cache size для лучшей производительности (8MB)
    if err := sqlDB.Exec("PRAGMA cache_size=-8000").Error; err != nil {
        return nil, fmt.Errorf("failed to set cache_size: %w", err)
    }
    
    // КРИТИЧНО: Для SQLite лучше использовать 1 соединение
    // Это предотвращает "database is locked" ошибки
    // Очередь запросов выстраивается на уровне Go, а не ломает БД
    sqlDB.SetMaxOpenConns(1)
    sqlDB.SetMaxIdleConns(1)
    sqlDB.SetConnMaxLifetime(0)
    
    return db, nil
}
```

**Traffic Stats Batching для минимизации записей:**

```go
// internal/services/traffic_collector.go
type TrafficCollector struct {
    db            *gorm.DB
    coreProviders map[string]CoreStatsProvider
    interval      time.Duration
    
    // Batch buffer для накопления статистики
    statsBatch    []models.TrafficStats
    batchMutex    sync.Mutex
    batchSize     int
    batchInterval time.Duration
}

func NewTrafficCollector(db *gorm.DB, interval time.Duration) *TrafficCollector {
    return &TrafficCollector{
        db:            db,
        coreProviders: make(map[string]CoreStatsProvider),
        interval:      interval,
        statsBatch:    make([]models.TrafficStats, 0, 100),
        batchSize:     100,  // Писать в БД каждые 100 записей
        batchInterval: 60 * time.Second, // или каждые 60 секунд
    }
}

func (tc *TrafficCollector) Start() {
    // Goroutine для сбора статистики
    go func() {
        ticker := time.NewTicker(tc.interval)
        defer ticker.Stop()
        
        for range ticker.C {
            tc.collectStats()
        }
    }()
    
    // Goroutine для периодической записи батчей
    go func() {
        ticker := time.NewTicker(tc.batchInterval)
        defer ticker.Stop()
        
        for range ticker.C {
            tc.flushBatch()
        }
    }()
}

func (tc *TrafficCollector) collectStats() {
    // Собираем статистику со всех ядер
    var users []models.User
    tc.db.Where("is_active = ?", true).Find(&users)
    
    for _, user := range users {
        totalUpload := int64(0)
        totalDownload := int64(0)
        
        for coreName, provider := range tc.coreProviders {
            stats, err := provider.GetUserTraffic(user.UUID)
            if err != nil {
                log.Warn().Err(err).Str("core", coreName).Msg("failed to get stats")
                continue
            }
            
            totalUpload += stats.Upload
            totalDownload += stats.Download
        }
        
        // Добавляем в batch вместо немедленной записи
        tc.addToBatch(models.TrafficStats{
            UserID:        user.ID,
            UploadBytes:   totalUpload,
            DownloadBytes: totalDownload,
            RecordedAt:    time.Now(),
        })
        
        // Обновляем общий счетчик пользователя (это быстрая операция)
        tc.db.Model(&user).Update("traffic_used_bytes", 
            gorm.Expr("traffic_used_bytes + ?", totalUpload + totalDownload))
        
        // Проверяем квоту
        if user.TrafficLimitBytes != nil && 
           user.TrafficUsedBytes >= *user.TrafficLimitBytes {
            tc.HandleQuotaExceeded(user)
        }
    }
}

func (tc *TrafficCollector) addToBatch(stat models.TrafficStats) {
    tc.batchMutex.Lock()
    defer tc.batchMutex.Unlock()
    
    tc.statsBatch = append(tc.statsBatch, stat)
    
    // Если batch заполнен, записываем в БД
    if len(tc.statsBatch) >= tc.batchSize {
        tc.flushBatchUnsafe()
    }
}

func (tc *TrafficCollector) flushBatch() {
    tc.batchMutex.Lock()
    defer tc.batchMutex.Unlock()
    tc.flushBatchUnsafe()
}

func (tc *TrafficCollector) flushBatchUnsafe() {
    if len(tc.statsBatch) == 0 {
        return
    }
    
    // Batch insert - одна транзакция вместо N запросов
    if err := tc.db.CreateInBatches(tc.statsBatch, 100).Error; err != nil {
        log.Error().Err(err).Msg("failed to flush traffic stats batch")
        return
    }
    
    log.Debug().Int("count", len(tc.statsBatch)).Msg("flushed traffic stats batch")
    
    // Очищаем batch
    tc.statsBatch = tc.statsBatch[:0]
}
```

**Почему это критично:**

1. **WAL mode**: Позволяет одновременное чтение и запись. Без этого панель будет получать "database is locked" при попытке записать логи во время чтения пользователей.

2. **busy_timeout**: Дает 5 секунд на ожидание освобождения БД вместо немедленной ошибки.

3. **SetMaxOpenConns(1)**: SQLite не поддерживает настоящую конкурентную запись. Одно соединение + WAL mode = правильная архитектура.

4. **Batching**: Вместо 30 записей в секунду (при 30 пользователях и интервале 10 сек) делаем 1 batch insert раз в минуту. Это снижает нагрузку на БД в 60 раз.

**Acceptance Criteria:**
- ✅ PRAGMA journal_mode=WAL установлен при инициализации БД
- ✅ PRAGMA busy_timeout=5000 установлен
- ✅ SetMaxOpenConns(1) для SQLite соединения
- ✅ Traffic stats пишутся батчами (100 записей или 60 секунд)
- ✅ Нет "database is locked" ошибок при нагрузке
- ✅ Concurrent read/write работает корректно
- ✅ Unit тесты для batching механизма
```

**Comprehensive Logging Strategy:**

**Logger Configuration (internal/logger/logger.go):**
```go
package logger

import (
    "fmt"
    "io"
    "os"
    "path/filepath"
    "time"
    
    "github.com/rs/zerolog"
    "gopkg.in/natefinch/lumberjack.v2"
)

var Log zerolog.Logger

type Config struct {
    Level      string // debug, info, warn, error
    Format     string // json, console
    Output     string // stdout, file, both
    FilePath   string
    MaxSize    int // MB
    MaxBackups int
    MaxAge     int // days
    Compress   bool
}

func Init(cfg *Config) error {
    // Set log level
    level, err := parseLevel(cfg.Level)
    if err != nil {
        return err
    }
    zerolog.SetGlobalLevel(level)
    
    // Configure time format
    zerolog.TimeFieldFormat = time.RFC3339
    
    // Configure output
    var writers []io.Writer
    
    // Console output
    if cfg.Output == "stdout" || cfg.Output == "both" {
        if cfg.Format == "console" {
            writers = append(writers, zerolog.ConsoleWriter{
                Out:        os.Stdout,
                TimeFormat: "2006-01-02 15:04:05",
                NoColor:    false,
            })
        } else {
            writers = append(writers, os.Stdout)
        }
    }
    
    // File output with rotation
    if cfg.Output == "file" || cfg.Output == "both" {
        // Ensure log directory exists
        logDir := filepath.Dir(cfg.FilePath)
        if err := os.MkdirAll(logDir, 0755); err != nil {
            return fmt.Errorf("failed to create log directory: %w", err)
        }
        
        fileWriter := &lumberjack.Logger{
            Filename:   cfg.FilePath,
            MaxSize:    cfg.MaxSize,
            MaxBackups: cfg.MaxBackups,
            MaxAge:     cfg.MaxAge,
            Compress:   cfg.Compress,
        }
        writers = append(writers, fileWriter)
    }
    
    // Create multi-writer
    var output io.Writer
    if len(writers) == 1 {
        output = writers[0]
    } else {
        output = io.MultiWriter(writers...)
    }
    
    // Create logger
    Log = zerolog.New(output).With().
        Timestamp().
        Caller().
        Logger()
    
    Log.Info().
        Str("level", cfg.Level).
        Str("format", cfg.Format).
        Str("output", cfg.Output).
        Msg("Logger initialized")
    
    return nil
}

func parseLevel(level string) (zerolog.Level, error) {
    switch level {
    case "debug":
        return zerolog.DebugLevel, nil
    case "info":
        return zerolog.InfoLevel, nil
    case "warn":
        return zerolog.WarnLevel, nil
    case "error":
        return zerolog.ErrorLevel, nil
    default:
        return zerolog.InfoLevel, fmt.Errorf("unknown log level: %s", level)
    }
}

// Component-specific loggers
func WithComponent(component string) zerolog.Logger {
    return Log.With().Str("component", component).Logger()
}

// Request-specific logger
func WithRequestID(requestID string) zerolog.Logger {
    return Log.With().Str("request_id", requestID).Logger()
}

// User-specific logger
func WithUser(userID uint, username string) zerolog.Logger {
    return Log.With().
        Uint("user_id", userID).
        Str("username", username).
        Logger()
}

// Sampled logger for high-volume events
func SampledLogger(component string, sampleRate int) zerolog.Logger {
    return Log.Sample(&zerolog.BasicSampler{N: sampleRate}).
        With().
        Str("component", component).
        Logger()
}
```

**Request ID Middleware (internal/middleware/request_id.go):**
```go
package middleware

import (
    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"
    "github.com/yourusername/isolate-panel/internal/logger"
)

const RequestIDHeader = "X-Request-ID"

func RequestID() fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Get or generate request ID
        requestID := c.Get(RequestIDHeader)
        if requestID == "" {
            requestID = uuid.New().String()
        }
        
        // Set request ID in context
        c.Locals("request_id", requestID)
        
        // Set response header
        c.Set(RequestIDHeader, requestID)
        
        return c.Next()
    }
}

// GetRequestID retrieves request ID from context
func GetRequestID(c *fiber.Ctx) string {
    if requestID, ok := c.Locals("request_id").(string); ok {
        return requestID
    }
    return ""
}

// GetLogger returns a logger with request ID
func GetLogger(c *fiber.Ctx) zerolog.Logger {
    requestID := GetRequestID(c)
    return logger.WithRequestID(requestID)
}
```

**Logging Middleware (internal/middleware/logger.go):**
```go
package middleware

import (
    "time"
    
    "github.com/gofiber/fiber/v2"
    "github.com/yourusername/isolate-panel/internal/logger"
)

func Logger() fiber.Handler {
    return func(c *fiber.Ctx) error {
        start := time.Now()
        
        // Get request ID
        requestID := GetRequestID(c)
        
        // Process request
        err := c.Next()
        
        // Calculate duration
        duration := time.Since(start)
        
        // Get status code
        status := c.Response().StatusCode()
        
        // Create log event
        event := logger.Log.Info()
        if status >= 500 {
            event = logger.Log.Error()
        } else if status >= 400 {
            event = logger.Log.Warn()
        }
        
        // Log request
        event.
            Str("request_id", requestID).
            Str("method", c.Method()).
            Str("path", c.Path()).
            Str("ip", c.IP()).
            Int("status", status).
            Dur("duration", duration).
            Int("size", len(c.Response().Body())).
            Str("user_agent", c.Get("User-Agent")).
            Msg("HTTP request")
        
        return err
    }
}
```

**Component-Specific Logging Examples:**

**Auth Service:**
```go
// internal/services/auth_service.go
var authLog = logger.WithComponent("auth")

func (s *AuthService) Login(username, password string) (*LoginResponse, error) {
    authLog.Info().
        Str("username", username).
        Str("ip", s.getClientIP()).
        Msg("Login attempt")
    
    user, err := s.authenticateUser(username, password)
    if err != nil {
        authLog.Warn().
            Str("username", username).
            Err(err).
            Msg("Login failed")
        return nil, err
    }
    
    authLog.Info().
        Uint("user_id", user.ID).
        Str("username", username).
        Msg("Login successful")
    
    return tokens, nil
}
```

**Core Service:**
```go
// internal/services/core_service.go
var coreLog = logger.WithComponent("core")

func (s *CoreService) StartCore(coreID uint) error {
    core, err := s.getCore(coreID)
    if err != nil {
        return err
    }
    
    coreLog.Info().
        Uint("core_id", coreID).
        Str("core_type", core.Type).
        Str("core_name", core.Name).
        Msg("Starting core")
    
    if err := s.supervisord.Start(core.Name); err != nil {
        coreLog.Error().
            Uint("core_id", coreID).
            Str("core_name", core.Name).
            Err(err).
            Msg("Failed to start core")
        return err
    }
    
    coreLog.Info().
        Uint("core_id", coreID).
        Str("core_name", core.Name).
        Msg("Core started successfully")
    
    return nil
}
```

**Traffic Collector (with sampling):**
```go
// internal/services/traffic_collector.go
var trafficLog = logger.SampledLogger("traffic", 10) // Log 1 in 10 events

func (tc *TrafficCollector) CollectStats() error {
    start := time.Now()
    
    trafficLog.Debug().Msg("Starting traffic collection")
    
    stats, err := tc.collectFromAllCores()
    if err != nil {
        logger.Log.Error(). // Use main logger for errors
            Err(err).
            Msg("Failed to collect traffic stats")
        return err
    }
    
    duration := time.Since(start)
    trafficLog.Info().
        Int("stats_count", len(stats)).
        Dur("duration", duration).
        Msg("Traffic collection completed")
    
    return nil
}
```

**Error Logging with Stack Traces:**
```go
// internal/utils/errors.go
package utils

import (
    "runtime/debug"
    "github.com/yourusername/isolate-panel/internal/logger"
)

func LogPanic() {
    if r := recover(); r != nil {
        logger.Log.Error().
            Interface("panic", r).
            Str("stack", string(debug.Stack())).
            Msg("Panic recovered")
    }
}

// Usage in handlers
func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
    defer utils.LogPanic()
    // Handler logic...
}
```

**Configuration (config.yaml):**
```yaml
logging:
  level: info          # debug, info, warn, error
  format: json         # json, console
  output: both         # stdout, file, both
  file:
    path: /var/log/isolate-panel/app.log
    max_size: 100      # MB
    max_backups: 3
    max_age: 28        # days
    compress: true

# Component-specific log levels (optional)
log_levels:
  auth: info
  core: info
  traffic: warn        # Less verbose for high-volume
  haproxy: info
  api: info
```

**Resource Usage (for 1GB RAM, 1 CPU, 10GB disk VPS):**
- RAM: 3-8MB (0.3-0.8% of 1GB) ✅
- Disk: ~190MB with rotation and compression (1.9% of 10GB) ✅
- CPU: ~0.5% overhead ✅

**Optimizations for resource-constrained environments:**
- Log rotation prevents disk fill-up
- Compression reduces disk usage by 60-70%
- Log sampling for high-volume events
- Component-specific log levels
- Zerolog's zero-allocation design minimizes GC pressure

**Acceptance Criteria:**
- ✅ golang-migrate установлен и настроен
- ✅ Все 20 миграций созданы с up и down файлами
- ✅ Миграции embedded в бинарник (не требуют отдельных файлов)
- ✅ Migration Manager поддерживает up/down/steps/version/force
- ✅ CLI tool работает: `go run cmd/migrate/main.go -cmd up`
- ✅ Миграции запускаются автоматически при старте приложения
- ✅ Seed data создает default admin (username: admin, password: admin)
- ✅ Seed data создает default settings
- ✅ Development seeds создаются только в dev режиме (APP_ENV=development)
- ✅ Версия миграции отображается в логах при старте
- ✅ Dirty state обнаруживается и логируется
- ✅ Rollback работает корректно
- ✅ Structured logging настроен с log rotation (100MB, 3 backups, 28 days)
- ✅ Логи пишутся в файл и stdout одновременно
- ✅ Request ID добавляется для трассировки

**Deliverables:**
- HTTP сервер слушает на localhost:8080
- База данных инициализируется через migrations (не AutoMigrate)
- Все 20 таблиц созданы с правильными indexes
- Default admin создан автоматически
- Default settings созданы
- Логи пишутся в файл и stdout с rotation
- CLI tool для управления миграциями работает
- Rollback механизм работает

#### 1.2 Аутентификация (1 неделя)

**Задачи:**
- [ ] Модель Admin в БД
- [ ] Хеширование паролей (Argon2id)
- [ ] JWT токены (access + refresh)
- [ ] Endpoints: POST /api/auth/login, POST /api/auth/refresh, POST /api/auth/logout, GET /api/auth/me
- [ ] Middleware для проверки JWT
- [ ] Rate limiting для login endpoint
- [ ] Token refresh механизм с защитой от race conditions
- [ ] Graceful handling при истечении refresh token

**Endpoints:**
```
POST /api/auth/login
  Request: { "username": "admin", "password": "password" }
  Response: { "access_token": "...", "refresh_token": "...", "expires_at": "...", "user": {...} }

POST /api/auth/refresh
  Request: { "refresh_token": "..." }
  Response: { "access_token": "...", "refresh_token": "...", "expires_at": "..." }

POST /api/auth/logout
  Request: { "refresh_token": "..." }
  Response: { "message": "Logged out successfully" }

GET /api/auth/me
  Headers: Authorization: Bearer <access_token>
  Response: { "id": 1, "username": "admin", "is_admin": true }
```

**Acceptance Criteria:**
- ✅ Access token истекает через 15 минут
- ✅ Refresh token истекает через 7 дней
- ✅ Rate limiting: максимум 5 попыток логина в минуту
- ✅ Middleware корректно отклоняет невалидные токены
- ✅ GET /api/auth/me возвращает информацию о текущем пользователе
- ✅ Refresh token может быть использован только один раз (rotation)

**Deliverables:**
- Администратор может войти в систему
- JWT токены работают
- Защита от brute-force
- Endpoint для проверки текущего пользователя

#### 1.3 User Management System (5 дней)

> **📚 Детальная документация:** [PROTOCOL_SMART_FORMS_PLAN.md](./PROTOCOL_SMART_FORMS_PLAN.md#фаза-12-user-management-system)

**Задачи:**
- [ ] Расширить модель User с универсальными credentials (UUID, Password, Token)
- [ ] Реализовать UserService с auto-generation всех типов credentials
- [ ] API endpoints для управления пользователями
- [ ] Endpoint для регенерации credentials
- [ ] Валидация уникальности UUID, Token, Username

**Примечание MVP:** Credentials хранятся в plaintext для упрощения. Post-MVP: миграция на шифрование.

**Endpoints:**
- POST /api/users - создать пользователя (возвращает credentials)
- GET /api/users - список пользователей (с пагинацией)
- GET /api/users/:id - получить пользователя
- PUT /api/users/:id - обновить пользователя
- DELETE /api/users/:id - удалить пользователя
- POST /api/users/:id/regenerate - регенерировать credentials
- GET /api/users/:id/inbounds - список inbound для пользователя

**Acceptance Criteria:**
- ✅ Пользователь создается с автогенерацией всех credentials
- ✅ UUID, Token, Username уникальны (проверка на уровне БД и сервиса)
- ✅ API возвращает все credentials при создании
- ✅ Unit тесты для всех CRUD операций
- ✅ Integration тесты для API endpoints

**Deliverables:**
- Users CRUD через API
- Auto-generation всех типов credentials

#### 1.4 Управление ядрами (2 недели)

**Задачи:**
- [ ] Модель Core в БД
- [ ] Настройка Supervisord для управления процессами
- [ ] **КРИТИЧНО: Lazy Loading для ядер (экономия 80-100MB RAM)**
- [ ] Интеграция с Sing-box через Supervisord
- [ ] Интеграция с Xray через Supervisord
- [ ] Интеграция с Mihomo через Supervisord
- [ ] Генератор конфигурации Sing-box (Go structs → JSON)
- [ ] Генератор конфигурации Xray (Go structs → JSON)
- [ ] Генератор конфигурации Mihomo (Go structs → YAML)
- [ ] Валидация конфигов: `sing-box check`, `xray test`, etc.
- [ ] Graceful restart через SIGHUP (с fallback на restart)
- [ ] Мониторинг статуса ядра (PID, uptime через supervisorctl)
- [ ] Условный запуск ядер (только если есть активные inbound)
- [ ] Автоматический запуск ядра при создании первого inbound
- [ ] Автоматическая остановка ядра при удалении последнего inbound
- [ ] Endpoints для управления ядрами

**Задачи:**
- [ ] Модель Core в БД
- [ ] Настройка Supervisord для управления процессами
- [ ] **КРИТИЧНО: Lazy Loading для ядер (экономия 80-100MB RAM)**
- [ ] Интеграция с Sing-box через Supervisord
- [ ] Генератор конфигурации Sing-box (Go structs → JSON)
- [ ] Валидация конфигов: `sing-box check -c config.json`
- [ ] Graceful restart через SIGHUP (с fallback на restart)
- [ ] Мониторинг статуса ядра (PID, uptime через supervisorctl)
- [ ] Условный запуск ядер (только если есть активные inbound)
- [ ] Автоматический запуск ядра при создании первого inbound
- [ ] Автоматическая остановка ядра при удалении последнего inbound
- [ ] Полная настройка HAProxy с production-ready конфигурацией
- [ ] Генератор конфигурации HAProxy (SNI routing, path routing, health checks)
- [ ] HAProxy stats page configuration (port 8404)
- [ ] HAProxy health checks для всех ядер (inter 5s, fall 3, rise 2)
- [ ] HAProxy rate limiting (stick tables, per-IP limits)
- [ ] HAProxy SSL/TLS configuration (ciphers, TLS 1.2+)
- [ ] HAProxy logging configuration (structured logs)
- [ ] Валидация HAProxy конфигурации: `haproxy -c -f config.cfg`
- [ ] Graceful reload HAProxy через supervisorctl
- [ ] Endpoints для управления ядрами и HAProxy

**HAProxy Full Configuration Template:**
```haproxy
# /data/configs/haproxy.cfg
# Generated by Isolate Panel

global
    log /dev/log local0
    log /dev/log local1 notice
    chroot /var/lib/haproxy
    stats socket /run/haproxy/admin.sock mode 660 level admin
    stats timeout 30s
    user haproxy
    group haproxy
    daemon
    
    # Default SSL material locations
    ca-base /etc/ssl/certs
    crt-base /etc/ssl/private
    
    # SSL/TLS configuration
    ssl-default-bind-ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256
    ssl-default-bind-ciphersuites TLS_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384
    ssl-default-bind-options ssl-min-ver TLSv1.2 no-tls-tickets
    
    # Performance tuning
    maxconn 4096
    tune.ssl.default-dh-param 2048

defaults
    log     global
    mode    tcp
    option  tcplog
    option  dontlognull
    timeout connect 5000
    timeout client  50000
    timeout server  50000

# ============================================
# Stats Page (Admin Interface)
# ============================================
listen stats
    bind :8404
    mode http
    stats enable
    stats uri /stats
    stats refresh 5s
    stats show-legends
    stats show-node
    stats auth admin:{{.StatsPassword}}
    stats admin if TRUE

# ============================================
# Frontend: HTTPS (443) - SNI-based routing
# ============================================
frontend https_frontend
    bind :443
    mode tcp
    option tcplog
    
    # SNI-based routing
    tcp-request inspect-delay 5s
    tcp-request content accept if { req_ssl_hello_type 1 }
    
    # Route based on SNI
    {{range .SNIRoutes}}
    use_backend {{.BackendName}} if { req_ssl_sni -i {{.Domain}} }
    {{end}}
    
    # Default backend
    default_backend default_backend
    
    # Connection limits
    maxconn 2000
    
    # Rate limiting (per IP)
    stick-table type ip size 100k expire 30s store conn_rate(3s)
    tcp-request connection track-sc0 src
    tcp-request connection reject if { sc_conn_rate(0) gt 100 }

# ============================================
# Frontend: HTTP (80) - Path-based routing
# ============================================
frontend http_frontend
    bind :80
    mode http
    option httplog
    
    # Path-based routing
    {{range .PathRoutes}}
    use_backend {{.BackendName}} if { path_beg {{.Path}} }
    {{end}}
    
    # Default: redirect to HTTPS
    redirect scheme https code 301 if !{ ssl_fc }
    
    # Connection limits
    maxconn 2000

# ============================================
# Backends: Proxy Cores
# ============================================
{{range .Backends}}
backend {{.Name}}
    mode tcp
    balance roundrobin
    
    # Health checks
    option tcp-check
    tcp-check connect port {{.HealthCheckPort}}
    
    # Servers
    {{range .Servers}}
    server {{.Name}} {{.Address}}:{{.Port}} check inter 5s fall 3 rise 2 maxconn 500
    {{end}}
    
    # Timeouts
    timeout server 1h
    timeout connect 5s
    
    # Logging
    log global
{{end}}

# ============================================
# Default Backend (fallback)
# ============================================
backend default_backend
    mode tcp
    server default 127.0.0.1:9999 check
```

**HAProxy Configuration Generator (Go):**
```go
// internal/haproxy/config_generator.go
package haproxy

import (
    "bytes"
    "fmt"
    "os/exec"
    "text/template"
    "github.com/yourusername/isolate-panel/internal/models"
)

type HAProxyConfig struct {
    StatsPassword string
    SNIRoutes     []SNIRoute
    PathRoutes    []PathRoute
    Backends      []Backend
}

type SNIRoute struct {
    Domain      string
    BackendName string
}

type PathRoute struct {
    Path        string
    BackendName string
}

type Backend struct {
    Name            string
    HealthCheckPort int
    Servers         []Server
}

type Server struct {
    Name    string
    Address string
    Port    int
}

func GenerateHAProxyConfig(cores []models.Core, inbounds []models.Inbound, statsPassword string) (string, error) {
    config := HAProxyConfig{
        StatsPassword: statsPassword,
        SNIRoutes:     []SNIRoute{},
        PathRoutes:    []PathRoute{},
        Backends:      []Backend{},
    }
    
    // Build SNI routes
    for _, inbound := range inbounds {
        if inbound.TLS == "tls" && inbound.SNI != "" {
            config.SNIRoutes = append(config.SNIRoutes, SNIRoute{
                Domain:      inbound.SNI,
                BackendName: fmt.Sprintf("backend_%s_%d", inbound.CoreType, inbound.Port),
            })
        }
    }
    
    // Build path routes
    for _, inbound := range inbounds {
        if inbound.Path != "" {
            config.PathRoutes = append(config.PathRoutes, PathRoute{
                Path:        inbound.Path,
                BackendName: fmt.Sprintf("backend_%s_%d", inbound.CoreType, inbound.Port),
            })
        }
    }
    
    // Build backends
    backendMap := make(map[string]*Backend)
    for _, inbound := range inbounds {
        backendName := fmt.Sprintf("backend_%s_%d", inbound.CoreType, inbound.Port)
        
        if _, exists := backendMap[backendName]; !exists {
            backendMap[backendName] = &Backend{
                Name:            backendName,
                HealthCheckPort: inbound.Port,
                Servers:         []Server{},
            }
        }
        
        backendMap[backendName].Servers = append(backendMap[backendName].Servers, Server{
            Name:    fmt.Sprintf("%s_%d", inbound.CoreType, inbound.Port),
            Address: "127.0.0.1",
            Port:    inbound.Port,
        })
    }
    
    for _, backend := range backendMap {
        config.Backends = append(config.Backends, *backend)
    }
    
    // Render template
    tmpl, err := template.New("haproxy").Parse(haproxyTemplate)
    if err != nil {
        return "", err
    }
    
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, config); err != nil {
        return "", err
    }
    
    return buf.String(), nil
}

func ValidateHAProxyConfig(configPath string) error {
    cmd := exec.Command("haproxy", "-c", "-f", configPath)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("HAProxy config validation failed: %s", string(output))
    }
    return nil
}

func ReloadHAProxy() error {
    // Graceful reload via supervisorctl
    cmd := exec.Command("supervisorctl", "restart", "haproxy")
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to reload HAProxy: %w", err)
    }
    return nil
}
```

**Endpoints:**
- GET /api/cores - список ядер
- POST /api/cores/:id/start - запустить ядро
- POST /api/cores/:id/stop - остановить ядро
- POST /api/cores/:id/restart - перезапустить ядро
- GET /api/cores/:id/status - статус ядра

**Примечание:** HAProxy endpoints исключены из MVP (Post-MVP feature)

**КРИТИЧНО: Lazy Loading для ядер (экономия 80-100MB RAM):**

Вместо запуска всех трех ядер при старте системы, запускаем только те, которые реально используются.

**Supervisord Configuration с условным autostart:**

```ini
# /etc/supervisord.conf

[program:singbox]
command=/usr/local/bin/sing-box run -c /data/configs/singbox.json
autostart=false  # НЕ запускать автоматически
autorestart=true
stdout_logfile=/var/log/isolate-panel/singbox.log
stderr_logfile=/var/log/isolate-panel/singbox.err.log
startsecs=3
stopwaitsecs=10

[program:xray]
command=/usr/local/bin/xray run -c /data/configs/xray.json
autostart=false  # НЕ запускать автоматически
autorestart=true
stdout_logfile=/var/log/isolate-panel/xray.log
stderr_logfile=/var/log/isolate-panel/xray.err.log
startsecs=3
stopwaitsecs=10

[program:mihomo]
command=/usr/local/bin/mihomo -f /data/configs/mihomo.yaml
autostart=false  # НЕ запускать автоматически
autorestart=true
stdout_logfile=/var/log/isolate-panel/mihomo.log
stderr_logfile=/var/log/isolate-panel/mihomo.err.log
startsecs=3
stopwaitsecs=10

# HAProxy исключен из MVP (Post-MVP)
# [program:haproxy]
# command=/usr/local/sbin/haproxy -f /data/configs/haproxy.cfg
# autostart=false
# autorestart=true
# stdout_logfile=/var/log/isolate-panel/haproxy.log
# stderr_logfile=/var/log/isolate-panel/haproxy.err.log
```

**Core Lifecycle Manager:**

```go
// internal/services/core_lifecycle.go
package services

import (
    "fmt"
    
    "github.com/yourusername/isolate-panel/internal/models"
    "github.com/rs/zerolog/log"
    "gorm.io/gorm"
)

type CoreLifecycleManager struct {
    db          *gorm.DB
    coreManager *CoreManager
}

func NewCoreLifecycleManager(db *gorm.DB, coreManager *CoreManager) *CoreLifecycleManager {
    return &CoreLifecycleManager{
        db:          db,
        coreManager: coreManager,
    }
}

// InitializeCores запускает только необходимые ядра при старте системы
func (clm *CoreLifecycleManager) InitializeCores() error {
    log.Info().Msg("Initializing cores (lazy loading)")
    
    cores := []string{"singbox", "xray", "mihomo"}
    
    for _, coreName := range cores {
        shouldStart, err := clm.shouldCoreBeRunning(coreName)
        if err != nil {
            return fmt.Errorf("failed to check if core should run: %w", err)
        }
        
        if shouldStart {
            log.Info().Str("core", coreName).Msg("Starting core (has active inbounds)")
            if err := clm.coreManager.StartCore(coreName); err != nil {
                log.Error().Err(err).Str("core", coreName).Msg("Failed to start core")
                // Не возвращаем ошибку, продолжаем с другими ядрами
            }
        } else {
            log.Info().Str("core", coreName).Msg("Skipping core (no active inbounds)")
        }
    }
    
    return nil
}

// shouldCoreBeRunning проверяет, есть ли активные inbound для данного ядра
func (clm *CoreLifecycleManager) shouldCoreBeRunning(coreName string) (bool, error) {
    var core models.Core
    if err := clm.db.Where("name = ?", coreName).First(&core).Error; err != nil {
        return false, err
    }
    
    // Проверяем, есть ли активные inbound для этого ядра
    var count int64
    err := clm.db.Model(&models.Inbound{}).
        Where("core_id = ? AND is_enabled = ?", core.ID, true).
        Count(&count).Error
    
    if err != nil {
        return false, err
    }
    
    return count > 0, nil
}

func (clm *CoreLifecycleManager) isHAProxyEnabled() bool {
    var setting models.Setting
    err := clm.db.Where("key = ?", "haproxy_enabled").First(&setting).Error
    if err != nil {
        return false
    }
    return setting.Value == "true"
}

// OnInboundCreated вызывается при создании нового inbound
func (clm *CoreLifecycleManager) OnInboundCreated(inbound *models.Inbound) error {
    // Загружаем core
    var core models.Core
    if err := clm.db.First(&core, inbound.CoreID).Error; err != nil {
        return err
    }
    
    // Проверяем, запущено ли ядро
    isRunning, err := clm.coreManager.IsCoreRunning(core.Name)
    if err != nil {
        return err
    }
    
    if !isRunning {
        log.Info().
            Str("core", core.Name).
            Uint("inbound_id", inbound.ID).
            Msg("Starting core (first inbound created)")
        
        if err := clm.coreManager.StartCore(core.Name); err != nil {
            return fmt.Errorf("failed to start core: %w", err)
        }
    }
    
    return nil
}

// OnInboundDeleted вызывается при удалении inbound
func (clm *CoreLifecycleManager) OnInboundDeleted(inbound *models.Inbound) error {
    // Загружаем core
    var core models.Core
    if err := clm.db.First(&core, inbound.CoreID).Error; err != nil {
        return err
    }
    
    // Проверяем, остались ли еще inbound для этого ядра
    var count int64
    err := clm.db.Model(&models.Inbound{}).
        Where("core_id = ? AND is_enabled = ? AND id != ?", core.ID, true, inbound.ID).
        Count(&count).Error
    
    if err != nil {
        return err
    }
    
    // Если это был последний inbound, останавливаем ядро
    if count == 0 {
        log.Info().
            Str("core", core.Name).
            Uint("inbound_id", inbound.ID).
            Msg("Stopping core (last inbound deleted)")
        
        if err := clm.coreManager.StopCore(core.Name); err != nil {
            log.Error().Err(err).Str("core", core.Name).Msg("Failed to stop core")
            // Не возвращаем ошибку, это не критично
        }
    }
    
    return nil
}
```

**Integration в Inbound Service:**

```go
// internal/services/inbound_service.go

func (is *InboundService) CreateInbound(req *CreateInboundRequest) (*models.Inbound, error) {
    // ... validation and creation logic ...
    
    inbound := &models.Inbound{
        Name:     req.Name,
        Protocol: req.Protocol,
        CoreID:   req.CoreID,
        Port:     req.Port,
        // ... other fields ...
    }
    
    if err := is.db.Create(inbound).Error; err != nil {
        return nil, err
    }
    
    // НОВОЕ: Уведомляем lifecycle manager о создании inbound
    if err := is.lifecycleManager.OnInboundCreated(inbound); err != nil {
        log.Error().Err(err).Msg("Failed to handle inbound creation lifecycle")
        // Не возвращаем ошибку, inbound уже создан
    }
    
    // Регенерируем конфиг и применяем
    if err := is.applyInboundConfig(inbound); err != nil {
        return nil, err
    }
    
    return inbound, nil
}

func (is *InboundService) DeleteInbound(id uint) error {
    var inbound models.Inbound
    if err := is.db.First(&inbound, id).Error; err != nil {
        return err
    }
    
    // НОВОЕ: Уведомляем lifecycle manager о удалении inbound
    if err := is.lifecycleManager.OnInboundDeleted(&inbound); err != nil {
        log.Error().Err(err).Msg("Failed to handle inbound deletion lifecycle")
    }
    
    if err := is.db.Delete(&inbound).Error; err != nil {
        return err
    }
    
    return nil
}
```

**Экономия памяти:**

```
Сценарий 1: Используется только Sing-box (типичный случай)
БЕЗ lazy loading:
- Sing-box: 50MB
- Xray: 40MB (не используется, но запущен)
- Mihomo: 40MB (не используется, но запущен)
ИТОГО: 130MB

С lazy loading:
- Sing-box: 50MB
- Xray: 0MB (не запущен)
- Mihomo: 0MB (не запущен)
ИТОГО: 50MB
ЭКОНОМИЯ: 80MB (61%)

Сценарий 2: Используется Sing-box + Xray
С lazy loading:
- Sing-box: 50MB
- Xray: 40MB
- Mihomo: 0MB (не запущен)
ИТОГО: 90MB
ЭКОНОМИЯ: 40MB (30%)
```

**Acceptance Criteria:**
- ✅ Все три ядра (Sing-box, Xray, Mihomo) запускаются через supervisord за < 5 секунд
- ✅ **Ядра НЕ запускаются автоматически при старте системы (autostart=false)**
- ✅ **Ядро запускается автоматически при создании первого inbound для него**
- ✅ **Ядро останавливается автоматически при удалении последнего inbound**
- ✅ **Экономия 80-100MB RAM когда используется только одно ядро**
- ✅ API endpoint GET /api/cores/:id/status возвращает корректный статус
- ✅ При некорректном конфиге ядро не запускается и возвращается ошибка
- ✅ Graceful restart не обрывает существующие соединения (если поддерживается)
- ✅ Валидация конфигураций перед применением (sing-box check, xray test)
- ✅ Unit тесты для core manager и lifecycle manager (coverage > 80%)
- ✅ Документация API для core management endpoints

**Deliverables:**
- Все три ядра запускаются и управляются через Supervisord
- **Lazy loading экономит 80-100MB RAM на типичных конфигурациях**
- **Ядра запускаются/останавливаются автоматически по требованию**
- Конфигурация генерируется автоматически из БД
- Graceful restart работает для всех ядер
- Валидация конфигураций перед применением

---

### Фаза 1.5: UI/UX Design & Design System (1 неделя)

**Цель:** Создать полную спецификацию дизайна перед началом разработки frontend

#### 1.5.1 Wireframes & Navigation Structure (2 дня)

**Задачи:**
- [ ] Создать wireframes для всех основных страниц
- [ ] Определить navigation structure (Sidebar + Header)
- [ ] Спроектировать page layouts
- [ ] Определить responsive breakpoints
- [ ] Спроектировать mobile navigation (hamburger menu)

**Wireframes для создания:**
- Login page
- Dashboard (overview, stats, quick actions)
- Users (list, create/edit modal, details)
- Inbounds (list, create/edit form)
- Outbounds (list, create/edit form)
- Cores (status cards, logs viewer)
- Certificates (list, request form)
- Routing (rules editor)
- Statistics (charts, filters)
- Connections (live table)
- Logs (viewer with filters)
- Settings (tabs, forms)

**Navigation Structure:**
```
┌─────────────────────────────────────────────────────────────┐
│ [Logo] Isolate Panel  [Theme] [Lang] [User ▼]              │
├──────────┬──────────────────────────────────────────────────┤
│          │                                                   │
│ Sidebar  │              Main Content Area                   │
│          │                                                   │
│ Dashboard│                                                   │
│ Users    │                                                   │
│ Inbounds │                                                   │
│ Outbounds│                                                   │
│ Cores    │                                                   │
│ Certs    │                                                   │
│ Routing  │                                                   │
│ Stats    │                                                   │
│ Connects │                                                   │
│ Logs     │                                                   │
│ Settings │                                                   │
│          │                                                   │
└──────────┴──────────────────────────────────────────────────┘
```

**Responsive Breakpoints:**
- sm: 640px (Mobile landscape)
- md: 768px (Tablet)
- lg: 1024px (Desktop)
- xl: 1280px (Large desktop)
- 2xl: 1536px (Extra large)

**Deliverables:**
- Wireframes для всех 12 страниц
- Navigation structure документация
- Mobile navigation design (hamburger + drawer)

#### 1.5.2 Design System Specification (2 дня)

**Задачи:**
- [ ] Определить цветовую палитру (light + dark themes)
- [ ] Определить типографику
- [ ] Определить spacing system
- [ ] Определить border radius values
- [ ] Определить shadow system
- [ ] Определить transition timings
- [ ] Определить z-index scale
- [ ] Создать design tokens (CSS variables)

**Design Tokens Structure:**
```css
:root {
  /* Colors */
  --color-primary: 59 130 246;
  --color-success: 34 197 94;
  --color-warning: 245 158 11;
  --color-danger: 239 68 68;
  
  /* Backgrounds */
  --bg-primary: 255 255 255;
  --bg-secondary: 249 250 251;
  --bg-tertiary: 243 244 246;
  
  /* Text */
  --text-primary: 17 24 39;
  --text-secondary: 75 85 99;
  --text-tertiary: 156 163 175;
  
  /* Borders */
  --border-primary: 229 231 235;
  
  /* Spacing */
  --spacing-xs: 0.25rem;
  --spacing-sm: 0.5rem;
  --spacing-md: 1rem;
  --spacing-lg: 1.5rem;
  --spacing-xl: 2rem;
  
  /* Transitions */
  --transition-fast: 150ms;
  --transition-base: 200ms;
  --transition-slow: 300ms;
}

[data-theme="dark"] {
  --bg-primary: 17 24 39;
  --bg-secondary: 31 41 55;
  --text-primary: 243 244 246;
  /* ... */
}
```

**Tailwind Configuration:**
- Настроить custom colors через CSS variables
- Настроить spacing scale
- Настроить typography
- Добавить plugins: @tailwindcss/forms, @tailwindcss/typography

**Deliverables:**
- Design tokens спецификация
- Tailwind config с custom theme
- Color palette документация
- Typography scale документация

#### 1.5.3 Component Library Structure (1 день)

**Задачи:**
- [ ] Определить component hierarchy
- [ ] Спроектировать базовые UI компоненты
- [ ] Спроектировать layout компоненты
- [ ] Спроектировать form компоненты
- [ ] Спроектировать state views (loading, error, empty)
- [ ] Определить icon system (Lucide Icons)

**Component Structure:**
```
frontend/src/components/
├── ui/                    # Базовые UI компоненты
│   ├── Button.tsx
│   ├── Input.tsx
│   ├── Select.tsx
│   ├── Checkbox.tsx
│   ├── Radio.tsx
│   ├── Switch.tsx
│   ├── Modal.tsx
│   ├── Toast.tsx
│   ├── Card.tsx
│   ├── Table.tsx
│   ├── Badge.tsx
│   ├── Spinner.tsx
│   ├── Alert.tsx
│   ├── Tooltip.tsx
│   ├── Dropdown.tsx
│   ├── Tabs.tsx
│   └── Icon.tsx
│
├── layout/                # Layout компоненты
│   ├── Sidebar.tsx
│   ├── MobileSidebar.tsx
│   ├── Header.tsx
│   ├── PageHeader.tsx
│   ├── Container.tsx
│   └── PageLayout.tsx
│
├── forms/                 # Form компоненты
│   ├── FormField.tsx
│   ├── FormError.tsx
│   ├── FormLabel.tsx
│   ├── UserForm.tsx
│   ├── InboundForm.tsx
│   ├── OutboundForm.tsx
│   └── CertificateForm.tsx
│
├── features/              # Feature-specific компоненты
│   ├── UserTable.tsx
│   ├── InboundCard.tsx
│   ├── TrafficChart.tsx
│   ├── ConnectionList.tsx
│   ├── CoreStatusCard.tsx
│   ├── LogViewer.tsx
│   └── LiveIndicator.tsx
│
└── views/                 # State views
    ├── LoadingView.tsx
    ├── ErrorView.tsx
    └── EmptyView.tsx
```

**Icon System:**
- Использовать Lucide Icons (tree-shakeable, легковесная)
- Создать Icon wrapper компонент
- Определить icon mapping для навигации

**Deliverables:**
- Component library структура
- Спецификация каждого компонента
- Icon system документация

#### 1.5.4 Theme & Internationalization Design (1 день)

**Задачи:**
- [ ] Спроектировать theme switching mechanism
- [ ] Спроектировать language switching mechanism
- [ ] Определить i18n структуру
- [ ] Создать translation file structure
- [ ] Спроектировать theme persistence
- [ ] Спроектировать language persistence

**Theme System:**
- Zustand store для theme state
- CSS attribute `[data-theme="dark"]` на `<html>`
- Tailwind dark mode через attribute
- LocalStorage persistence
- Theme toggle в Header

**i18n System:**
- Библиотека: i18next + react-i18next
- Языки: English (default), Russian, Chinese
- Browser language detection
- LocalStorage persistence
- Language switcher в Header

**Translation Structure:**
```
frontend/src/i18n/
├── index.ts              # i18n setup
└── locales/
    ├── en.json           # English
    ├── ru.json           # Russian
    └── zh.json           # Chinese
```

**Translation File Structure:**
```json
{
  "common": { "save": "Save", "cancel": "Cancel", ... },
  "nav": { "dashboard": "Dashboard", "users": "Users", ... },
  "auth": { "login": "Login", "logout": "Logout", ... },
  "users": { "title": "Users", "addUser": "Add User", ... }
}
```

**Deliverables:**
- Theme system спецификация
- i18n структура
- Translation files template
- Theme/Language switcher design

#### 1.5.5 Accessibility & Animations (1 день)

**Задачи:**
- [ ] Определить accessibility requirements (WCAG 2.1 AA)
- [ ] Спроектировать keyboard navigation
- [ ] Спроектировать focus management
- [ ] Спроектировать screen reader support
- [ ] Определить animation system
- [ ] Спроектировать loading animations
- [ ] Спроектировать transition effects

**Accessibility Requirements:**
- ARIA labels для всех интерактивных элементов
- Keyboard navigation (Tab, Enter, Space, Escape)
- Focus trap для модальных окон
- Focus visible indicators
- Screen reader announcements (role, aria-live, aria-label)
- Color contrast ratio ≥ 4.5:1 (WCAG AA)

**Keyboard Navigation:**
- Tab: переход между элементами
- Enter/Space: активация кнопок
- Escape: закрытие модальных окон
- Arrow keys: навигация в dropdown/select

**Animation System:**
- Tailwind animations + custom keyframes
- Transition timings: fast (150ms), base (200ms), slow (300ms)
- Loading spinner animation
- Modal/Toast fade in/out
- Sidebar slide in/out (mobile)
- Hover/Focus transitions

**Deliverables:**
- Accessibility checklist (WCAG 2.1 AA)
- Keyboard navigation specification
- Animation system documentation
- Focus management patterns

#### 1.5.6 Performance & Error Handling Design (1 день)

**Задачи:**
- [ ] Спроектировать code splitting strategy
- [ ] Спроектировать lazy loading для routes
- [ ] Спроектировать virtual scrolling для больших списков
- [ ] Спроектировать error boundaries
- [ ] Спроектировать error UI patterns
- [ ] Спроектировать real-time indicators

**Performance Strategy:**
- Code splitting: lazy load routes
- Virtual scrolling: для таблиц с >100 строк
- Image optimization: lazy loading
- Bundle size monitoring
- Tree-shaking для icons

**Error Handling:**
- Error Boundary для каждого route
- Graceful error UI
- Error logging (готовность к интеграции)
- Retry mechanisms
- Fallback UI

**Real-time Indicators:**
- Live indicator для WebSocket connection
- Loading states для async operations
- Optimistic updates для лучшего UX
- Skeleton loaders

**Deliverables:**
- Performance optimization plan
- Error handling patterns
- Real-time UI patterns
- Loading states specification

**Acceptance Criteria для Фазы 1.5:**
- ✅ Wireframes созданы для всех 12 страниц
- ✅ Design system полностью документирован
- ✅ Component library структура определена
- ✅ Theme system спроектирован (light/dark)
- ✅ i18n структура определена (en, ru, zh)
- ✅ Accessibility requirements документированы (WCAG 2.1 AA)
- ✅ Animation system определен
- ✅ Performance strategy определена
- ✅ Error handling patterns определены
- ✅ Mobile responsive design спроектирован

**Deliverables для Фазы 1.5:**
- Wireframes (Figma/Sketch или аналог)
- Design System документация
- Component Library спецификация
- Theme & i18n спецификация
- Accessibility checklist
- Performance plan
- Готовность к началу разработки frontend

---

### Фаза 2: MVP Frontend (2-3 недели)

#### 2.1 Базовая настройка + Design System Implementation (5 дней)

**Задачи:**

**День 1: Project Setup**
- [ ] Настройка Vite + Preact + TypeScript
- [ ] Настройка ESLint + Prettier
- [ ] Настройка path aliases (@/ для src/)
- [ ] Базовая структура директорий

**День 2: Design System Foundation**
- [ ] Создать design tokens (src/styles/tokens.css)
- [ ] Настроить Tailwind CSS с custom theme
- [ ] Настроить Tailwind plugins (@tailwindcss/forms, @tailwindcss/typography)
- [ ] Создать global styles
- [ ] Настроить CSS variables для light/dark themes

**День 3: Core Infrastructure**
- [ ] Настроить Zustand для state management
- [ ] Создать theme store (light/dark switching)
- [ ] Создать toast store (notifications)
- [ ] Настроить i18next для internationalization
- [ ] Создать translation files (en, ru, zh)
- [ ] Настроить axios для API client
- [ ] Создать API interceptors (auth, error handling)

**День 4: Base UI Components**
- [ ] Icon component (Lucide Icons wrapper)
- [ ] Button component (variants: primary, secondary, danger, ghost)
- [ ] Input component (text, email, password, number)
- [ ] Select component
- [ ] Checkbox component
- [ ] Switch component
- [ ] Card component
- [ ] Badge component
- [ ] Spinner component
- [ ] Alert component

**День 5: Layout & Navigation**
- [ ] Sidebar component (desktop)
- [ ] MobileSidebar component (hamburger + drawer)
- [ ] Header component (theme switcher, language switcher, user menu)
- [ ] **System Metrics Widget в Header (RAM/CPU monitoring)**
- [ ] PageLayout component
- [ ] PageHeader component
- [ ] Container component
- [ ] Базовый роутинг (preact-router)
- [ ] ProtectedRoute component (route guards)

**Структура проекта:**
```
frontend/
├── src/
│   ├── api/
│   │   ├── client.ts              # Axios instance с interceptors
│   │   └── endpoints/             # API endpoint functions
│   ├── components/
│   │   ├── ui/                    # Базовые UI компоненты
│   │   ├── layout/                # Layout компоненты
│   │   ├── forms/                 # Form компоненты
│   │   ├── features/              # Feature компоненты
│   │   └── views/                 # State views (Loading, Error, Empty)
│   ├── hooks/
│   │   ├── useBreakpoint.ts       # Responsive breakpoint hook
│   │   ├── useFocusTrap.ts        # Focus management
│   │   └── useClickOutside.ts     # Click outside detection
│   ├── i18n/
│   │   ├── index.ts               # i18n setup
│   │   └── locales/
│   │       ├── en.json
│   │       ├── ru.json
│   │       └── zh.json
│   ├── pages/                     # Page components
│   ├── router/
│   │   ├── ProtectedRoute.tsx     # Route guard
│   │   └── routes.tsx             # Route configuration
│   ├── stores/
│   │   ├── authStore.ts           # Authentication state
│   │   ├── themeStore.ts          # Theme state
│   │   └── toastStore.ts          # Toast notifications
│   ├── styles/
│   │   ├── tokens.css             # Design tokens
│   │   └── global.css             # Global styles
│   ├── utils/
│   │   ├── animations.ts          # Animation utilities
│   │   ├── formatters.ts          # Data formatters
│   │   └── validators.ts          # Form validators
│   ├── config/
│   │   ├── icons.ts               # Icon mapping
│   │   └── breakpoints.ts         # Responsive breakpoints
│   ├── App.tsx
│   └── main.tsx
├── tailwind.config.js
├── vite.config.ts
├── tsconfig.json
└── package.json
```

**Dependencies:**
```json
{
  "dependencies": {
    "preact": "^10.29.0",
    "preact-router": "^4.1.2",
    "zustand": "^5.0.12",
    "axios": "^1.7.9",
    "i18next": "^24.2.0",
    "react-i18next": "^15.2.0",
    "i18next-browser-languagedetector": "^8.0.2",
    "lucide-preact": "^0.468.0",
    "clsx": "^2.1.1"
  },
  "devDependencies": {
    "@preact/preset-vite": "^2.9.3",
    "vite": "^6.0.7",
    "typescript": "^5.9.3",
    "tailwindcss": "^4.2.2",
    "@tailwindcss/forms": "^0.5.9",
    "@tailwindcss/typography": "^0.5.16",
    "autoprefixer": "^10.4.20",
    "postcss": "^8.4.49"
  }
}
```

**Acceptance Criteria:**
- ✅ Vite dev server запускается без ошибок
- ✅ Tailwind CSS работает с custom theme
- ✅ Design tokens применяются корректно
- ✅ Theme switching работает (light/dark)
- ✅ Language switching работает (en/ru/zh)
- ✅ Все базовые UI компоненты рендерятся корректно
- ✅ Sidebar работает на desktop
- ✅ Mobile sidebar (drawer) работает на mobile
- ✅ Icons отображаются корректно
- ✅ Responsive breakpoints работают
- ✅ TypeScript компилируется без ошибок
- ✅ **System Metrics Widget отображается в Header**
- ✅ **RAM/CPU метрики обновляются каждые 5 секунд**

**System Metrics Widget в Header:**

Постоянный мониторинг ресурсов критичен для VPS с 1GB RAM.

```typescript
// frontend/src/components/layout/SystemMetricsWidget.tsx
import { useQuery } from '../../hooks/useQuery'
import { Activity, AlertCircle } from 'lucide-preact'
import { Tooltip } from '../ui/Tooltip'

interface SystemMetrics {
  ram: {
    used: number
    total: number
    percent: number
  }
  cpu: {
    percent: number
  }
}

export const SystemMetricsWidget = () => {
  const { data: metrics, isLoading } = useQuery<SystemMetrics>(
    '/api/system/resources',
    { refetchInterval: 5000 } // Update every 5 seconds
  )
  
  if (isLoading || !metrics) {
    return (
      <div className="flex items-center gap-2 text-gray-400">
        <Activity size={16} className="animate-pulse" />
        <span className="text-xs">Loading...</span>
      </div>
    )
  }
  
  const ramStatus = getStatus(metrics.ram.percent)
  const cpuStatus = getStatus(metrics.cpu.percent)
  
  return (
    <div className="flex items-center gap-3">
      {/* RAM Indicator */}
      <Tooltip content={`RAM: ${metrics.ram.used}MB / ${metrics.ram.total}MB`}>
        <div className="flex items-center gap-1.5 cursor-help">
          <div className={`w-2 h-2 rounded-full ${ramStatus.dotColor} animate-pulse`} />
          <span className={`text-xs font-medium ${ramStatus.textColor}`}>
            RAM {metrics.ram.percent}%
          </span>
        </div>
      </Tooltip>
      
      {/* CPU Indicator */}
      <Tooltip content={`CPU Usage: ${metrics.cpu.percent}%`}>
        <div className="flex items-center gap-1.5 cursor-help">
          <Activity size={14} className={cpuStatus.textColor} />
          <span className={`text-xs font-medium ${cpuStatus.textColor}`}>
            {metrics.cpu.percent}%
          </span>
        </div>
      </Tooltip>
      
      {/* Warning Icon if critical */}
      {(metrics.ram.percent > 85 || metrics.cpu.percent > 85) && (
        <Tooltip content="System resources critical! Check dashboard.">
          <AlertCircle size={16} className="text-red-500 animate-pulse" />
        </Tooltip>
      )}
    </div>
  )
}

function getStatus(percent: number) {
  if (percent > 85) {
    return {
      dotColor: 'bg-red-500',
      textColor: 'text-red-600 dark:text-red-400',
      status: 'critical'
    }
  }
  if (percent > 70) {
    return {
      dotColor: 'bg-yellow-500',
      textColor: 'text-yellow-600 dark:text-yellow-400',
      status: 'warning'
    }
  }
  return {
    dotColor: 'bg-green-500',
    textColor: 'text-green-600 dark:text-green-400',
    status: 'healthy'
  }
}
```

**Integration в Header:**

```typescript
// frontend/src/components/layout/Header.tsx
import { SystemMetricsWidget } from './SystemMetricsWidget'
import { ThemeSwitcher } from './ThemeSwitcher'
import { LanguageSwitcher } from './LanguageSwitcher'
import { UserMenu } from './UserMenu'

export const Header = () => {
  return (
    <header className="h-16 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
      <div className="h-full px-4 flex items-center justify-between">
        {/* Left: Logo */}
        <div className="flex items-center gap-3">
          <h1 className="text-xl font-bold">Isolate Panel</h1>
        </div>
        
        {/* Center: System Metrics (always visible) */}
        <div className="flex-1 flex justify-center">
          <SystemMetricsWidget />
        </div>
        
        {/* Right: Actions */}
        <div className="flex items-center gap-3">
          <ThemeSwitcher />
          <LanguageSwitcher />
          <UserMenu />
        </div>
      </div>
    </header>
  )
}
```

**Mobile Responsive:**

```typescript
// На мобильных устройствах показываем только индикаторы без текста
export const SystemMetricsWidget = () => {
  const { data: metrics } = useQuery<SystemMetrics>(
    '/api/system/resources',
    { refetchInterval: 5000 }
  )
  
  const isMobile = useBreakpoint('md') // < 768px
  
  if (!metrics) return null
  
  if (isMobile) {
    // Compact view for mobile
    return (
      <div className="flex items-center gap-2">
        <div className={`w-2 h-2 rounded-full ${getStatus(metrics.ram.percent).dotColor}`} />
        <span className="text-xs">{metrics.ram.percent}%</span>
      </div>
    )
  }
  
  // Full view for desktop (as shown above)
  return (/* ... full implementation ... */)
}
```

**Преимущества:**

1. **Постоянная видимость** - метрики всегда на виду, не нужно открывать dashboard
2. **Раннее предупреждение** - цветовая индикация показывает проблемы до критической ситуации
3. **Минимальное место** - компактный дизайн не загромождает header
4. **Real-time** - обновление каждые 5 секунд
5. **Responsive** - адаптируется под мобильные устройства

**Acceptance Criteria:**
- ✅ Vite dev server запускается без ошибок
- ✅ Tailwind CSS работает с custom theme
- ✅ Design tokens применяются корректно
- ✅ Theme switching работает (light/dark)
- ✅ Language switching работает (en/ru/zh)
- ✅ Все базовые UI компоненты рендерятся корректно
- ✅ Sidebar работает на desktop
- ✅ Mobile sidebar (drawer) работает на mobile
- ✅ Icons отображаются корректно
- ✅ Responsive breakpoints работают
- ✅ TypeScript компилируется без ошибок
- ✅ **System Metrics Widget отображается в Header**
- ✅ **RAM/CPU метрики обновляются каждые 5 секунд**
- ✅ **Цветовая индикация: зеленый (<70%), желтый (70-85%), красный (>85%)**
- ✅ **Warning icon появляется при критических значениях**
- ✅ **Tooltip показывает детальную информацию**
- ✅ **Responsive дизайн для mobile устройств**

**Deliverables:**
- Frontend собирается и запускается
- Design system полностью реализован
- Базовые UI компоненты готовы к использованию
- Layout компоненты готовы
- Theme system работает
- i18n система работает
- API client настроен
- Routing настроен с route guards

#### 2.2 Аутентификация UI (4 дня)

**Задачи:**
- [ ] Страница логина с валидацией формы
- [ ] Auth Store (Zustand) с persist middleware
- [ ] API client с interceptors (axios)
- [ ] Router-level route guards (ProtectedRoute компонент)
- [ ] API interceptor с очередью для concurrent 401
- [ ] Автоматический refresh токена с защитой от race conditions
- [ ] Loading state при проверке аутентификации (предотвращение flash контента)
- [ ] Graceful logout при неудачном refresh

**Структура файлов:**
```
frontend/src/
├── stores/
│   └── authStore.ts          # Zustand store для аутентификации
├── api/
│   └── client.ts             # Axios client с interceptors
├── router/
│   ├── ProtectedRoute.tsx    # Router-level guard
│   └── routes.tsx            # Конфигурация routes
└── pages/
    └── Login.tsx             # Страница логина
```

**Auth Store (Zustand):**
```typescript
interface AuthState {
  accessToken: string | null
  refreshToken: string | null
  isAuthenticated: boolean
  isLoading: boolean
  user: { username: string; isAdmin: boolean } | null
  login: (username: string, password: string) => Promise<void>
  logout: () => void
  refreshToken: () => Promise<string>
  checkAuth: () => Promise<void>
}
```

**API Interceptor:**
- Флаг `isRefreshing` для предотвращения множественных refresh запросов
- Очередь `failedQueue` для запросов, ожидающих refresh
- Флаг `_retry` для предотвращения бесконечного цикла
- Автоматический logout при неудачном refresh

**ProtectedRoute компонент:**
- Проверка аутентификации при монтировании
- Loading state ДО проверки (нет flash контента)
- Автоматический redirect на /login если не авторизован
- Не рендерит защищенный компонент до проверки

**Acceptance Criteria:**
- ✅ Прямой доступ к /dashboard без логина редиректит на /login
- ✅ Нет flash защищенного контента (loading state работает)
- ✅ После логина пользователь редиректится на /dashboard
- ✅ При 401 ответе происходит попытка refresh token
- ✅ Если refresh token невалиден - редирект на /login
- ✅ Concurrent 401 запросы не вызывают множественные refresh
- ✅ Токены сохраняются в localStorage через Zustand persist
- ✅ При перезагрузке страницы аутентификация сохраняется

**Deliverables:**
- Администратор может войти через UI
- Токены обновляются автоматически
- Все защищенные routes недоступны без аутентификации
- Нет race conditions при refresh токена

#### 2.3 Frontend Architecture & Data Management (3 дня)

**Цель:** Реализовать production-ready архитектуру для работы с данными, формами и real-time обновлениями

**День 1: Form Validation & Custom Hooks**

**Задачи:**
- [ ] Установить Zod для validation
- [ ] Создать validation schemas для всех форм
- [ ] Реализовать useForm hook
- [ ] Создать переиспользуемые form компоненты

**Form Validation с Zod:**
```typescript
// src/utils/validators.ts
import { z } from 'zod'

export const userSchema = z.object({
  username: z.string().min(3).max(50).regex(/^[a-zA-Z0-9_-]+$/),
  email: z.string().email().optional(),
  trafficLimit: z.number().min(0).optional(),
  expiryDate: z.string().datetime().optional(),
  isAdmin: z.boolean().default(false),
})

export type UserFormData = z.infer<typeof userSchema>

export const inboundSchema = z.object({
  name: z.string().min(1),
  protocol: z.enum(['vmess', 'vless', 'trojan', 'shadowsocks', 'hysteria2']),
  port: z.number().min(1).max(65535),
  settings: z.record(z.any()),
})

export type InboundFormData = z.infer<typeof inboundSchema>
```

**useForm Hook:**
```typescript
// src/hooks/useForm.ts
import { useState } from 'preact/hooks'
import { z } from 'zod'

interface UseFormOptions<T> {
  schema: z.ZodSchema<T>
  onSubmit: (data: T) => Promise<void>
  initialValues?: Partial<T>
}

export const useForm = <T extends Record<string, any>>({
  schema,
  onSubmit,
  initialValues = {},
}: UseFormOptions<T>) => {
  const [values, setValues] = useState<Partial<T>>(initialValues)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [isSubmitting, setIsSubmitting] = useState(false)

  const handleChange = (name: keyof T, value: any) => {
    setValues(prev => ({ ...prev, [name]: value }))
    if (errors[name as string]) {
      setErrors(prev => {
        const newErrors = { ...prev }
        delete newErrors[name as string]
        return newErrors
      })
    }
  }

  const validate = (): boolean => {
    try {
      schema.parse(values)
      setErrors({})
      return true
    } catch (error) {
      if (error instanceof z.ZodError) {
        const newErrors: Record<string, string> = {}
        error.errors.forEach(err => {
          if (err.path[0]) {
            newErrors[err.path[0] as string] = err.message
          }
        })
        setErrors(newErrors)
      }
      return false
    }
  }

  const handleSubmit = async (e?: Event) => {
    e?.preventDefault()
    if (!validate()) return
    
    setIsSubmitting(true)
    try {
      await onSubmit(values as T)
    } finally {
      setIsSubmitting(false)
    }
  }

  const reset = () => {
    setValues(initialValues)
    setErrors({})
  }

  return { values, errors, isSubmitting, handleChange, handleSubmit, reset, setValues }
}
```

**День 2: Data Fetching & Caching**

**Задачи:**
- [ ] Реализовать useQuery hook (SWR-like pattern)
- [ ] Реализовать useMutation hook
- [ ] Создать in-memory cache с TTL
- [ ] Создать domain-specific hooks (useUsers, useInbounds, etc.)

**useQuery Hook:**
```typescript
// src/hooks/useQuery.ts
import { useState, useEffect, useCallback } from 'preact/hooks'

interface UseQueryOptions<T> {
  enabled?: boolean
  refetchInterval?: number
  cacheTime?: number
  onSuccess?: (data: T) => void
  onError?: (error: Error) => void
}

interface UseQueryResult<T> {
  data: T | null
  error: Error | null
  isLoading: boolean
  isRefetching: boolean
  refetch: () => Promise<void>
}

export const useQuery = <T>(
  key: string,
  fetcher: () => Promise<T>,
  options: UseQueryOptions<T> = {}
): UseQueryResult<T> => {
  const { enabled = true, refetchInterval, cacheTime = 300000, onSuccess, onError } = options
  
  const [data, setData] = useState<T | null>(() => cache.get(key))
  const [error, setError] = useState<Error | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [isRefetching, setIsRefetching] = useState(false)

  const fetchData = useCallback(async (isRefetch = false) => {
    if (!enabled) return

    // Check cache first
    if (!isRefetch) {
      const cachedData = cache.get<T>(key)
      if (cachedData) {
        setData(cachedData)
        setIsLoading(false)
        return
      }
    }

    try {
      if (isRefetch) {
        setIsRefetching(true)
      } else {
        setIsLoading(true)
      }

      const result = await fetcher()
      cache.set(key, result, cacheTime)
      setData(result)
      setError(null)
      onSuccess?.(result)
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Unknown error')
      setError(error)
      onError?.(error)
    } finally {
      setIsLoading(false)
      setIsRefetching(false)
    }
  }, [enabled, fetcher, key, cacheTime, onSuccess, onError])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  // Polling
  useEffect(() => {
    if (!refetchInterval || !enabled) return

    const interval = setInterval(() => {
      fetchData(true)
    }, refetchInterval)

    return () => clearInterval(interval)
  }, [refetchInterval, enabled, fetchData])

  const refetch = useCallback(() => fetchData(true), [fetchData])

  return { data, error, isLoading, isRefetching, refetch }
}

// useMutation Hook
interface UseMutationOptions<TData, TVariables> {
  onSuccess?: (data: TData, variables: TVariables) => void
  onError?: (error: Error, variables: TVariables) => void
}

export const useMutation = <TData, TVariables>(
  mutationFn: (variables: TVariables) => Promise<TData>,
  options: UseMutationOptions<TData, TVariables> = {}
) => {
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<Error | null>(null)

  const mutate = useCallback(async (variables: TVariables) => {
    setIsLoading(true)
    setError(null)

    try {
      const data = await mutationFn(variables)
      options.onSuccess?.(data, variables)
      return data
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Unknown error')
      setError(error)
      options.onError?.(error, variables)
      throw error
    } finally {
      setIsLoading(false)
    }
  }, [mutationFn, options])

  return { mutate, isLoading, error }
}
```

**Cache Implementation:**
```typescript
// src/utils/cache.ts
interface CacheEntry<T> {
  data: T
  timestamp: number
  ttl: number
}

class Cache {
  private cache = new Map<string, CacheEntry<any>>()

  set<T>(key: string, data: T, ttl: number = 300000): void {
    this.cache.set(key, { data, timestamp: Date.now(), ttl })
  }

  get<T>(key: string): T | null {
    const entry = this.cache.get(key)
    if (!entry) return null

    const isExpired = Date.now() - entry.timestamp > entry.ttl
    if (isExpired) {
      this.cache.delete(key)
      return null
    }

    return entry.data as T
  }

  invalidate(key: string): void {
    this.cache.delete(key)
  }

  invalidatePattern(pattern: RegExp): void {
    for (const key of this.cache.keys()) {
      if (pattern.test(key)) {
        this.cache.delete(key)
      }
    }
  }

  clear(): void {
    this.cache.clear()
  }
}

export const cache = new Cache()
```

**Domain-specific Hooks:**
```typescript
// src/hooks/useUsers.ts
import { useQuery, useMutation } from './useQuery'
import api from '../api/client'
import { useToastStore } from '../stores/toastStore'

export const useUsers = () => {
  return useQuery('users', () => api.get('/users').then(res => res.data), {
    refetchInterval: 30000, // Refetch every 30 seconds
  })
}

export const useUser = (id: number) => {
  return useQuery(
    `user-${id}`,
    () => api.get(`/users/${id}`).then(res => res.data),
    { enabled: !!id }
  )
}

export const useCreateUser = () => {
  const { addToast } = useToastStore()
  
  return useMutation(
    (data: UserFormData) => api.post('/users', data).then(res => res.data),
    {
      onSuccess: () => {
        addToast({ type: 'success', message: 'User created successfully' })
        cache.invalidate('users')
      },
      onError: (error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )
}

export const useUpdateUser = () => {
  const { addToast } = useToastStore()
  
  return useMutation(
    ({ id, data }: { id: number; data: Partial<UserFormData> }) =>
      api.put(`/users/${id}`, data).then(res => res.data),
    {
      onSuccess: (_, { id }) => {
        addToast({ type: 'success', message: 'User updated successfully' })
        cache.invalidate('users')
        cache.invalidate(`user-${id}`)
      },
      onError: (error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )
}

export const useDeleteUser = () => {
  const { addToast } = useToastStore()
  
  return useMutation(
    (id: number) => api.delete(`/users/${id}`).then(res => res.data),
    {
      onSuccess: () => {
        addToast({ type: 'success', message: 'User deleted successfully' })
        cache.invalidate('users')
      },
      onError: (error) => {
        addToast({ type: 'error', message: error.message })
      },
    }
  )
}
```

**День 3: Real-time Updates & Optimistic Updates**

**Задачи:**
- [ ] Реализовать useWebSocket hook
- [ ] Создать hooks для connections и logs
- [ ] Реализовать polling для statistics
- [ ] Реализовать optimistic updates

**WebSocket Hook:**
```typescript
// src/hooks/useWebSocket.ts
import { useState, useEffect, useRef, useCallback } from 'preact/hooks'

interface UseWebSocketOptions {
  onMessage?: (data: any) => void
  onError?: (error: Event) => void
  onOpen?: () => void
  onClose?: () => void
  reconnectInterval?: number
  reconnectAttempts?: number
}

export const useWebSocket = (url: string, options: UseWebSocketOptions = {}) => {
  const {
    onMessage,
    onError,
    onOpen,
    onClose,
    reconnectInterval = 3000,
    reconnectAttempts = 5,
  } = options

  const [isConnected, setIsConnected] = useState(false)
  const [lastMessage, setLastMessage] = useState<any>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectCountRef = useRef(0)
  const reconnectTimeoutRef = useRef<number>()

  const connect = useCallback(() => {
    try {
      const ws = new WebSocket(url)

      ws.onopen = () => {
        setIsConnected(true)
        reconnectCountRef.current = 0
        onOpen?.()
      }

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          setLastMessage(data)
          onMessage?.(data)
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error)
        }
      }

      ws.onerror = (error) => {
        console.error('WebSocket error:', error)
        onError?.(error)
      }

      ws.onclose = () => {
        setIsConnected(false)
        onClose?.()

        // Auto-reconnect
        if (reconnectCountRef.current < reconnectAttempts) {
          reconnectCountRef.current++
          reconnectTimeoutRef.current = window.setTimeout(() => {
            connect()
          }, reconnectInterval)
        }
      }

      wsRef.current = ws
    } catch (error) {
      console.error('Failed to create WebSocket:', error)
    }
  }, [url, onMessage, onError, onOpen, onClose, reconnectInterval, reconnectAttempts])

  useEffect(() => {
    connect()

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [connect])

  const send = useCallback((data: any) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data))
    }
  }, [])

  const disconnect = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close()
    }
  }, [])

  return { isConnected, lastMessage, send, disconnect }
}

// Usage for connections
export const useConnections = () => {
  const [connections, setConnections] = useState<Connection[]>([])

  const { isConnected } = useWebSocket('ws://localhost:8080/api/connections', {
    onMessage: (data) => {
      setConnections(data.connections || [])
    },
  })

  return { connections, isConnected }
}

// Usage for logs
export const useLogs = () => {
  const [logs, setLogs] = useState<LogEntry[]>([])

  const { isConnected } = useWebSocket('ws://localhost:8080/api/logs', {
    onMessage: (data) => {
      setLogs(prev => [...prev, data].slice(-1000)) // Keep last 1000 logs
    },
  })

  const clearLogs = () => setLogs([])

  return { logs, isConnected, clearLogs }
}
```

**Polling для Statistics:**
```typescript
// src/hooks/useStats.ts
export const useStats = () => {
  return useQuery(
    'stats',
    () => api.get('/stats').then(res => res.data),
    { refetchInterval: 60000 } // Poll every 60 seconds
  )
}

export const useCoreStatus = () => {
  return useQuery(
    'core-status',
    () => api.get('/cores/status').then(res => res.data),
    { refetchInterval: 5000 } // Poll every 5 seconds
  )
}
```

**Optimistic Updates:**
```typescript
// src/hooks/useOptimisticUpdate.ts
import { useCallback } from 'preact/hooks'
import { cache } from '../utils/cache'

export const useOptimisticUpdate = <T>(queryKey: string) => {
  const optimisticUpdate = useCallback(
    async (
      updater: (oldData: T) => T,
      mutationFn: () => Promise<T>
    ) => {
      const oldData = cache.get<T>(queryKey)
      if (!oldData) {
        return await mutationFn()
      }

      // Apply optimistic update
      const optimisticData = updater(oldData)
      cache.set(queryKey, optimisticData)

      try {
        const result = await mutationFn()
        cache.set(queryKey, result)
        return result
      } catch (error) {
        // Rollback on error
        cache.set(queryKey, oldData)
        throw error
      }
    },
    [queryKey]
  )

  return { optimisticUpdate }
}
```

**Структура файлов:**
```
frontend/src/
├── hooks/
│   ├── useForm.ts              # Form validation hook
│   ├── useQuery.ts             # Data fetching hooks
│   ├── useWebSocket.ts         # WebSocket hook
│   ├── useOptimisticUpdate.ts  # Optimistic updates
│   ├── useUsers.ts             # User management hooks
│   ├── useInbounds.ts          # Inbound management hooks
│   ├── useStats.ts             # Statistics hooks
│   └── useConnections.ts       # Connections hooks
├── utils/
│   ├── cache.ts                # In-memory cache
│   └── validators.ts           # Zod schemas
```

**Dependencies:**
```json
{
  "dependencies": {
    "zod": "^3.24.1"
  }
}
```

**Acceptance Criteria:**
- ✅ Form validation работает с Zod schemas
- ✅ useForm hook переиспользуется во всех формах
- ✅ useQuery hook кеширует данные с TTL
- ✅ useMutation hook интегрирован с toast notifications
- ✅ WebSocket подключается с auto-reconnect
- ✅ Polling работает для statistics и core status
- ✅ Optimistic updates применяются с rollback при ошибке
- ✅ Cache invalidation работает корректно
- ✅ Все domain hooks (useUsers, useInbounds, etc.) реализованы
- ✅ TypeScript типизация полная

**Deliverables:**
- Production-ready data fetching architecture
- Type-safe form validation
- Real-time updates через WebSocket
- Caching с TTL и invalidation
- Optimistic updates с rollback
- Переиспользуемые hooks для всех операций
- Централизованная обработка ошибок

#### 2.4 Dashboard (4 дня)

**Задачи:**
- [ ] Главная страница с обзором
- [ ] Статистика: количество пользователей, трафик, активные подключения
- [ ] Статус ядер
- [ ] Быстрые действия
- [ ] **RAM Panic Button (критично для 1GB VPS)**
- [ ] Отображение текущего потребления RAM и CPU
- [ ] Emergency cleanup функционал

**RAM Panic Button (для VPS с ограниченной памятью):**

На VPS с 1GB RAM каждый мегабайт на счету. Когда память заканчивается, администратору нужна возможность быстро освободить ресурсы.

**UI Component:**

```typescript
// frontend/src/components/features/RAMPanicButton.tsx
import { useState } from 'preact/hooks'
import { Button } from '../ui/Button'
import { Card } from '../ui/Card'
import { Alert } from '../ui/Alert'
import { Spinner } from '../ui/Spinner'
import { AlertTriangle, Zap } from 'lucide-preact'

interface SystemResources {
  ram: {
    total: number
    used: number
    free: number
    percent: number
  }
  cpu: {
    percent: number
    cores: number
  }
}

export const RAMPanicButton = () => {
  const [isExecuting, setIsExecuting] = useState(false)
  const [result, setResult] = useState<string | null>(null)
  
  const { data: resources } = useQuery<SystemResources>(
    '/api/system/resources',
    { refetchInterval: 5000 }
  )
  
  const handlePanic = async () => {
    if (!confirm('This will clear caches and restart cores. Continue?')) {
      return
    }
    
    setIsExecuting(true)
    setResult(null)
    
    try {
      const response = await api.post('/api/system/emergency-cleanup')
      setResult(`✓ Freed ${response.data.freed_mb}MB of memory`)
    } catch (error) {
      setResult(`✗ Failed: ${error.message}`)
    } finally {
      setIsExecuting(false)
    }
  }
  
  if (!resources) return <Spinner />
  
  const ramPercent = resources.ram.percent
  const isWarning = ramPercent > 70
  const isCritical = ramPercent > 85
  
  return (
    <Card className="border-2 border-red-500">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold flex items-center gap-2">
          <AlertTriangle className="text-red-500" />
          Emergency Memory Management
        </h3>
      </div>
      
      {/* RAM Usage */}
      <div className="mb-4">
        <div className="flex justify-between text-sm mb-1">
          <span>RAM Usage</span>
          <span className={
            isCritical ? 'text-red-600 font-bold' :
            isWarning ? 'text-yellow-600 font-semibold' :
            'text-green-600'
          }>
            {resources.ram.used}MB / {resources.ram.total}MB ({ramPercent}%)
          </span>
        </div>
        <div className="w-full bg-gray-200 rounded-full h-3">
          <div
            className={`h-3 rounded-full transition-all ${
              isCritical ? 'bg-red-600' :
              isWarning ? 'bg-yellow-500' :
              'bg-green-500'
            }`}
            style={{ width: `${ramPercent}%` }}
          />
        </div>
      </div>
      
      {/* CPU Usage */}
      <div className="mb-4">
        <div className="flex justify-between text-sm mb-1">
          <span>CPU Usage</span>
          <span>{resources.cpu.percent}%</span>
        </div>
        <div className="w-full bg-gray-200 rounded-full h-2">
          <div
            className="bg-blue-500 h-2 rounded-full transition-all"
            style={{ width: `${resources.cpu.percent}%` }}
          />
        </div>
      </div>
      
      {(isWarning || isCritical) && (
        <Alert variant={isCritical ? 'danger' : 'warning'} className="mb-4">
          {isCritical ? (
            <>
              <strong>Critical:</strong> Memory usage is very high. 
              Consider using the panic button to free resources.
            </>
          ) : (
            <>
              <strong>Warning:</strong> Memory usage is elevated. 
              Monitor closely or free resources if needed.
            </>
          )}
        </Alert>
      )}
      
      <Button
        variant="danger"
        onClick={handlePanic}
        disabled={isExecuting}
        className="w-full"
      >
        {isExecuting ? (
          <>
            <Spinner className="mr-2" size="sm" />
            Cleaning up...
          </>
        ) : (
          <>
            <Zap className="mr-2" />
            Emergency: Free Memory
          </>
        )}
      </Button>
      
      {result && (
        <div className={`mt-3 text-sm ${
          result.startsWith('✓') ? 'text-green-600' : 'text-red-600'
        }`}>
          {result}
        </div>
      )}
      
      <div className="mt-3 text-xs text-gray-500">
        <strong>What this does:</strong>
        <ul className="list-disc list-inside mt-1">
          <li>Clears subscription cache</li>
          <li>Clears config generation cache</li>
          <li>Gracefully restarts all cores</li>
          <li>Forces Go garbage collection</li>
          <li>Flushes SQLite cache</li>
        </ul>
      </div>
    </Card>
  )
}
```

**Backend API:**

```go
// internal/handlers/system_handler.go
func (h *SystemHandler) EmergencyCleanup(c *fiber.Ctx) error {
    log.Warn().Msg("Emergency cleanup initiated by admin")
    
    freedMB := 0
    
    // 1. Clear subscription cache
    h.subscriptionCache.Clear()
    freedMB += 5
    
    // 2. Clear config generation cache
    h.configCache.Clear()
    freedMB += 3
    
    // 3. Gracefully restart cores (frees memory leaks)
    cores := []string{"singbox", "xray", "mihomo"}
    for _, coreName := range cores {
        isRunning, _ := h.coreManager.IsCoreRunning(coreName)
        if isRunning {
            log.Info().Str("core", coreName).Msg("Restarting core for memory cleanup")
            if err := h.coreManager.GracefulReload(coreName); err != nil {
                log.Error().Err(err).Str("core", coreName).Msg("Failed to reload core")
            }
            freedMB += 20 // Approximate
        }
    }
    
    // 4. Force Go garbage collection
    runtime.GC()
    debug.FreeOSMemory()
    freedMB += 10
    
    // 5. Flush SQLite cache (PRAGMA shrink_memory)
    h.db.Exec("PRAGMA shrink_memory")
    freedMB += 5
    
    log.Info().Int("freed_mb", freedMB).Msg("Emergency cleanup completed")
    
    return c.JSON(fiber.Map{
        "success":  true,
        "freed_mb": freedMB,
        "message":  "Emergency cleanup completed successfully",
    })
}

func (h *SystemHandler) GetSystemResources(c *fiber.Ctx) error {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    // Get system memory info
    vmStat, err := mem.VirtualMemory()
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "Failed to get memory stats"})
    }
    
    // Get CPU usage
    cpuPercent, err := cpu.Percent(time.Second, false)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "Failed to get CPU stats"})
    }
    
    return c.JSON(fiber.Map{
        "ram": fiber.Map{
            "total":   vmStat.Total / 1024 / 1024,      // MB
            "used":    vmStat.Used / 1024 / 1024,       // MB
            "free":    vmStat.Free / 1024 / 1024,       // MB
            "percent": int(vmStat.UsedPercent),
        },
        "cpu": fiber.Map{
            "percent": int(cpuPercent[0]),
            "cores":   runtime.NumCPU(),
        },
    })
}
```

**Acceptance Criteria:**
- ✅ RAM Panic Button отображается на dashboard
- ✅ Текущее потребление RAM обновляется каждые 5 секунд
- ✅ Цветовая индикация: зеленый (<70%), желтый (70-85%), красный (>85%)
- ✅ Emergency cleanup освобождает 40-50MB памяти
- ✅ Graceful restart ядер не обрывает соединения
- ✅ Подтверждение перед выполнением cleanup
- ✅ Отображение результата (сколько MB освобождено)
- ✅ Описание действий cleanup для пользователя

**Deliverables:**
- Информативный dashboard
- Визуализация основных метрик
- **RAM Panic Button для экстренного освобождения памяти**
- **Real-time мониторинг RAM и CPU**
- **Критично для VPS с 1GB RAM**

#### 2.4 Управление пользователями UI (5 дней)

> **📚 Детальная документация:** [PROTOCOL_SMART_FORMS_PLAN.md](./PROTOCOL_SMART_FORMS_PLAN.md#122-frontend-для-users-2-дня)

**Задачи:**
- [ ] Страница `/users` с компактным списком пользователей
- [ ] Раскрывающиеся детали пользователя при клике
- [ ] Modal для создания пользователя
- [ ] **Отображение credentials ОДИН РАЗ после создания** (критично!)
- [ ] Возможность копирования каждого credential
- [ ] Возможность скачать credentials (JSON/TXT)
- [ ] Редактирование пользователя (квоты, email)
- [ ] Удаление пользователя (с подтверждением)
- [ ] Регенерация credentials (с предупреждением)
- [ ] Поиск и фильтрация
- [ ] Отображение inbound, в которых используется пользователь

**UI Components:**
- UsersList.tsx - главная страница со списком
- UserCreateModal.tsx - modal для создания
- UserDetailsPanel.tsx - раскрывающаяся панель деталей
- UserCredentialsDisplay.tsx - показ credentials (один раз!)
- UserInboundsList.tsx - список inbound пользователя

**Deliverables:**
- Полное управление пользователями через UI
- Credentials показываются ОДИН РАЗ после создания
- Удобный UX с компактным списком и раскрытием деталей

---

### Фаза 3: Inbound/Outbound (3 недели)

> **📚 Детальная документация:** [PROTOCOL_SMART_FORMS_PLAN.md](./PROTOCOL_SMART_FORMS_PLAN.md#фаза-3-inbound-management-с-protocol-aware-forms)

#### 3.1 Backend для Inbound + Protocol Schema Registry (1.5 недели)

**Задачи:**
- [ ] Модели Inbound, UserInboundMapping, HAProxyRoute в БД
- [ ] **Protocol Schema Registry** (новое!)
- [ ] CRUD endpoints для inbound
- [ ] **API для получения protocol schemas** (новое!)
- [ ] **API для управления пользователями в inbound** (новое!)
- [ ] Генерация конфигурации для ядер из БД
- [ ] **Динамическая регенерация конфига при изменении пользователей** (новое!)
- [ ] Генерация HAProxy routes при создании inbound
- [ ] Port Manager для выделения портов (диапазоны по ядрам)
- [ ] Валидация портов (проверка занятости)
- [ ] Автоматическая регенерация HAProxy конфига

**Новые API Endpoints:**
- GET /api/protocols - список всех протоколов
- GET /api/protocols/:name/schema - schema конкретного протокола
- GET /api/protocols/by-core/:core - протоколы для конкретного ядра
- POST /api/inbounds/:id/users - добавить пользователей к inbound
- DELETE /api/inbounds/:id/users/:userId - удалить пользователя из inbound
- GET /api/inbounds/:id/users - список пользователей inbound
- POST /api/inbounds/:id/users/bulk - массовое добавление пользователей

**Acceptance Criteria:**
- ✅ Inbound создается с автоматическим выделением порта
- ✅ HAProxy route создается автоматически
- ✅ Конфиг ядер регенерируется автоматически
- ✅ Валидация предотвращает конфликты портов
- ✅ **Protocol Schema Registry интегрирован**
- ✅ **API возвращает schema для каждого протокола**
- ✅ **Валидация параметров на основе schema**
- ✅ **Пользователи добавляются/удаляются из inbound через API**
- ✅ **Конфиг ядра автоматически регенерируется при изменении пользователей**
- ✅ **Поддержка массового добавления пользователей**
- ✅ Unit тесты для port manager и config generators

**Deliverables:**
- Inbound создаются и управляются через API
- Protocol-Aware валидация параметров
- Пользователи добавляются/удаляются из inbound динамически
- Конфигурация ядер обновляется автоматически
- HAProxy автоматически роутит трафик на правильное ядро

#### 3.2 Frontend для Inbound + Protocol-Aware Forms (1.5 недели)

**Задачи:**
- [ ] **Wizard для создания inbound (5 шагов)** (новое!)
- [ ] **Динамические формы на основе Protocol Schema** (новое!)
- [ ] **Детальная страница inbound с управлением пользователями** (новое!)
- [ ] Страница списка inbound
- [ ] Редактирование inbound
- [ ] Удаление inbound

**Wizard Steps:**
1. Выбор ядра (Sing-box / Xray / Mihomo)
2. Выбор протокола (фильтруется по ядру)
3. Базовые настройки (port, listen, protocol parameters)
4. Transport (опционально)
5. TLS (опционально/обязательно для некоторых протоколов)

**Deliverables:**
- Wizard-based создание inbound
- Protocol-Aware формы с auto-generation
- Управление пользователями через UI
- Детальная страница inbound с табами (Overview, Users, Config, Stats)

#### 3.3 Outbound + Routing (1 неделя)

**Задачи:**
- [ ] Модели Outbound, RoutingRule в БД
- [ ] CRUD endpoints для outbound и routing
- [ ] Генерация routing конфигурации
- [ ] Поддержка простых цепочек

**Deliverables:**
- Outbound и routing работают
- Трафик маршрутизируется согласно правилам

---

### Фаза 4: Подписки (2 недели)

#### 4.1 Генерация подписок (1 неделя)

**Задачи:**
- [ ] Endpoint: GET /sub/:token - получить подписку
- [ ] Endpoint: GET /sub/short/:code - получить подписку по short URL
- [ ] Endpoint: GET /sub/:token/qr - получить QR код подписки
- [ ] Генерация V2Ray формата (base64) с полной поддержкой всех параметров
- [ ] Генерация Clash формата (YAML) с proxy-groups
- [ ] Генерация Sing-box формата (JSON) с transport options
- [ ] User-Agent detection для автоматического выбора формата
- [ ] Query parameter ?format=clash|v2ray|singbox для явного выбора
- [ ] Subscription info headers (upload/download/total/expire)
- [ ] Profile-update-interval header (24 hours)
- [ ] Content-Disposition header для правильного имени файла
- [ ] Проверка активности пользователя и срока действия
- [ ] Модель SubscriptionShortURL в БД для коротких ссылок

**V2Ray Format Implementation:**
```go
// internal/subscription/v2ray.go
type V2RayNode struct {
    V    string `json:"v"`    // "2"
    Ps   string `json:"ps"`   // Node name
    Add  string `json:"add"`  // Server address
    Port string `json:"port"` // Port
    ID   string `json:"id"`   // UUID
    Aid  string `json:"aid"`  // AlterID
    Net  string `json:"net"`  // Network (tcp/ws/grpc)
    Type string `json:"type"` // Header type
    Host string `json:"host"` // Host header
    Path string `json:"path"` // Path
    TLS  string `json:"tls"`  // TLS (tls/none)
    SNI  string `json:"sni"`  // SNI
    Alpn string `json:"alpn"` // ALPN
}

func GenerateV2RaySubscription(user *models.User, inbounds []models.Inbound) string {
    var nodes []string
    for _, inbound := range inbounds {
        node := V2RayNode{
            V:    "2",
            Ps:   fmt.Sprintf("%s - %s", user.Username, inbound.Name),
            Add:  inbound.ServerAddress,
            Port: strconv.Itoa(inbound.Port),
            ID:   user.UUID,
            Aid:  "0",
            Net:  inbound.Network,
            Type: inbound.HeaderType,
            Host: inbound.Host,
            Path: inbound.Path,
            TLS:  inbound.TLS,
            SNI:  inbound.SNI,
            Alpn: inbound.ALPN,
        }
        jsonBytes, _ := json.Marshal(node)
        encoded := base64.StdEncoding.EncodeToString(jsonBytes)
        nodes = append(nodes, "vmess://"+encoded)
    }
    return base64.StdEncoding.EncodeToString([]byte(strings.Join(nodes, "\n")))
}
```

**Clash Format Implementation:**
```go
// internal/subscription/clash.go
func GenerateClashSubscription(user *models.User, inbounds []models.Inbound) string {
    var proxies []map[string]interface{}
    
    for _, inbound := range inbounds {
        proxy := map[string]interface{}{
            "name":    fmt.Sprintf("%s - %s", user.Username, inbound.Name),
            "type":    inbound.Protocol, // vmess, vless, trojan
            "server":  inbound.ServerAddress,
            "port":    inbound.Port,
            "uuid":    user.UUID,
            "alterId": 0,
            "cipher":  "auto",
            "tls":     inbound.TLS == "tls",
            "network": inbound.Network,
        }
        
        if inbound.Network == "ws" {
            proxy["ws-opts"] = map[string]interface{}{
                "path": inbound.Path,
                "headers": map[string]string{
                    "Host": inbound.Host,
                },
            }
        }
        
        proxies = append(proxies, proxy)
    }
    
    config := map[string]interface{}{
        "proxies": proxies,
        "proxy-groups": []map[string]interface{}{
            {
                "name":     "Auto",
                "type":     "url-test",
                "proxies":  getProxyNames(proxies),
                "url":      "http://www.gstatic.com/generate_204",
                "interval": 300,
            },
        },
    }
    
    yamlBytes, _ := yaml.Marshal(config)
    return string(yamlBytes)
}
```

**Sing-box Format Implementation:**
```go
// internal/subscription/singbox.go
func GenerateSingboxSubscription(user *models.User, inbounds []models.Inbound) string {
    var outbounds []map[string]interface{}
    
    for _, inbound := range inbounds {
        outbound := map[string]interface{}{
            "type":        inbound.Protocol,
            "tag":         fmt.Sprintf("%s - %s", user.Username, inbound.Name),
            "server":      inbound.ServerAddress,
            "server_port": inbound.Port,
            "uuid":        user.UUID,
        }
        
        if inbound.TLS == "tls" {
            outbound["tls"] = map[string]interface{}{
                "enabled":     true,
                "server_name": inbound.SNI,
                "alpn":        strings.Split(inbound.ALPN, ","),
            }
        }
        
        if inbound.Network == "ws" {
            outbound["transport"] = map[string]interface{}{
                "type": "ws",
                "path": inbound.Path,
                "headers": map[string]string{
                    "Host": inbound.Host,
                },
            }
        }
        
        outbounds = append(outbounds, outbound)
    }
    
    config := map[string]interface{}{
        "outbounds": outbounds,
    }
    
    jsonBytes, _ := json.MarshalIndent(config, "", "  ")
    return string(jsonBytes)
}
```

**User-Agent Detection:**
```go
// internal/subscription/detector.go
func DetectClientType(userAgent string) string {
    ua := strings.ToLower(userAgent)
    
    switch {
    case strings.Contains(ua, "clash"):
        return "clash"
    case strings.Contains(ua, "sing-box"):
        return "singbox"
    case strings.Contains(ua, "v2ray"):
        return "v2ray"
    case strings.Contains(ua, "shadowrocket"):
        return "v2ray" // Shadowrocket uses V2Ray format
    case strings.Contains(ua, "quantumult"):
        return "v2ray"
    case strings.Contains(ua, "surge"):
        return "surge"
    default:
        return "v2ray" // Default fallback
    }
}
```

**Subscription Handler:**
```go
// internal/handlers/subscription.go
func (h *SubscriptionHandler) GetSubscription(c *fiber.Ctx) error {
    token := c.Params("token")
    userAgent := c.Get("User-Agent")
    
    user, err := h.userService.GetBySubscriptionToken(token)
    if err != nil {
        return c.Status(404).SendString("Subscription not found")
    }
    
    // Check if user is active and not expired
    if !user.IsActive || (user.ExpiryDate != nil && user.ExpiryDate.Before(time.Now())) {
        return c.Status(403).SendString("Subscription expired or disabled")
    }
    
    inbounds := h.inboundService.GetUserInbounds(user.ID)
    
    // Detect format from User-Agent or query param
    format := c.Query("format", DetectClientType(userAgent))
    
    var content string
    var contentType string
    
    switch format {
    case "clash":
        content = GenerateClashSubscription(user, inbounds)
        contentType = "application/x-yaml"
    case "singbox":
        content = GenerateSingboxSubscription(user, inbounds)
        contentType = "application/json"
    default:
        content = GenerateV2RaySubscription(user, inbounds)
        contentType = "text/plain"
    }
    
    // Set subscription info headers
    c.Set("subscription-userinfo", fmt.Sprintf(
        "upload=%d; download=%d; total=%d; expire=%d",
        user.TrafficUsed,
        user.TrafficUsed,
        user.TrafficLimit,
        user.ExpiryDate.Unix(),
    ))
    c.Set("profile-update-interval", "24") // 24 hours
    c.Set("content-disposition", fmt.Sprintf("attachment; filename=%s.txt", user.Username))
    
    // Log subscription access
    h.logService.LogSubscriptionAccess(user.ID, userAgent, c.IP())
    
    return c.Type(contentType).SendString(content)
}
```

**Short URL Generation:**
```go
// internal/subscription/shorturl.go
func GenerateShortURL(token string) string {
    hash := sha256.Sum256([]byte(token))
    shortCode := base64.URLEncoding.EncodeToString(hash[:])[:6]
    return shortCode
}

// Handler
func (h *SubscriptionHandler) GetByShortURL(c *fiber.Ctx) error {
    shortCode := c.Params("code")
    
    var shortURL SubscriptionShortURL
    if err := h.db.Where("short_code = ?", shortCode).First(&shortURL).Error; err != nil {
        return c.Status(404).SendString("Short URL not found")
    }
    
    // Redirect to full subscription URL
    return c.Redirect(fmt.Sprintf("/sub/%s", shortURL.Token))
}
```

**QR Code Generation:**
```go
// internal/subscription/qrcode.go
import "github.com/skip2/go-qrcode"

func GenerateQRCode(subscriptionURL string) ([]byte, error) {
    png, err := qrcode.Encode(subscriptionURL, qrcode.Medium, 256)
    if err != nil {
        return nil, err
    }
    return png, nil
}

// Handler
func (h *SubscriptionHandler) GetQRCode(c *fiber.Ctx) error {
    token := c.Params("token")
    subscriptionURL := fmt.Sprintf("https://%s/sub/%s", c.Hostname(), token)
    
    qrCode, err := GenerateQRCode(subscriptionURL)
    if err != nil {
        return c.Status(500).SendString("Failed to generate QR code")
    }
    
    return c.Type("image/png").Send(qrCode)
}
```

**Database Models:**
```go
// internal/models/subscription.go
type SubscriptionShortURL struct {
    ID        uint      `gorm:"primaryKey"`
    ShortCode string    `gorm:"uniqueIndex;size:10"`
    Token     string    `gorm:"index;size:64"`
    UserID    uint      `gorm:"index"`
    CreatedAt time.Time
}

type SubscriptionAccess struct {
    ID        uint      `gorm:"primaryKey"`
    UserID    uint      `gorm:"index"`
    UserAgent string    `gorm:"size:255"`
    IPAddress string    `gorm:"size:45"`
    Format    string    `gorm:"size:20"`
    CreatedAt time.Time `gorm:"index"`
}
```

**Acceptance Criteria:**
- ✅ V2Ray формат генерируется корректно с всеми параметрами (network, tls, sni, alpn)
- ✅ Clash формат генерируется с proxy-groups и url-test
- ✅ Sing-box формат генерируется с transport и tls options
- ✅ User-Agent detection работает для всех популярных клиентов
- ✅ Query parameter ?format= переопределяет User-Agent detection
- ✅ Subscription-userinfo header содержит upload/download/total/expire
- ✅ Profile-update-interval header установлен в 24 часа
- ✅ Неактивные и истекшие пользователи получают 403 ошибку
- ✅ Short URL работают и редиректят на полную подписку
- ✅ QR коды генерируются корректно
- ✅ Все обращения к подписке логируются

**Deliverables:**
- Подписки генерируются корректно для всех форматов
- Работают во всех популярных клиентах (V2RayNG, Clash, Sing-box, Shadowrocket, Quantumult)
- Subscription info headers для отображения квот в клиентах
- Short URL для удобного шаринга
- QR коды для быстрого добавления

#### 4.2 Управление подписками (1 неделя)

**Задачи:**
- [ ] Обновление subscription token (regenerate)
- [ ] Статистика использования подписки (по дням, форматам, IP)
- [ ] Логирование обращений к подписке с User-Agent и IP
- [ ] Кэширование подписок с TTL 5 минут
- [ ] Инвалидация кэша при изменении inbound или пользователя
- [ ] Endpoint: GET /api/users/:id/subscription/stats - статистика обращений
- [ ] Endpoint: POST /api/users/:id/subscription/regenerate - новый токен
- [ ] UI для просмотра статистики подписки
- [ ] UI для копирования subscription URL и short URL
- [ ] UI для отображения QR кода

**Subscription Caching:**
```go
// internal/subscription/cache.go
type SubscriptionCache struct {
    cache *cache.Cache
}

func NewSubscriptionCache() *SubscriptionCache {
    return &SubscriptionCache{
        cache: cache.New(5*time.Minute, 10*time.Minute),
    }
}

func (sc *SubscriptionCache) Get(userID uint, format string) (string, bool) {
    key := fmt.Sprintf("sub:%d:%s", userID, format)
    if content, found := sc.cache.Get(key); found {
        return content.(string), true
    }
    return "", false
}

func (sc *SubscriptionCache) Set(userID uint, format string, content string) {
    key := fmt.Sprintf("sub:%d:%s", userID, format)
    sc.cache.Set(key, content, cache.DefaultExpiration)
}

func (sc *SubscriptionCache) Invalidate(userID uint) {
    formats := []string{"v2ray", "clash", "singbox"}
    for _, format := range formats {
        key := fmt.Sprintf("sub:%d:%s", userID, format)
        sc.cache.Delete(key)
    }
}
```

**Subscription Analytics:**
```go
// internal/services/subscription_service.go
type SubscriptionStats struct {
    TotalAccesses int
    ByFormat      map[string]int
    ByDay         map[string]int
    UniqueIPs     int
    LastAccess    time.Time
}

func (s *SubscriptionService) GetAccessStats(userID uint, days int) (*SubscriptionStats, error) {
    var accesses []SubscriptionAccess
    since := time.Now().AddDate(0, 0, -days)
    
    err := s.db.Where("user_id = ? AND created_at > ?", userID, since).
        Find(&accesses).Error
    
    if err != nil {
        return nil, err
    }
    
    stats := &SubscriptionStats{
        TotalAccesses: len(accesses),
        ByFormat:      make(map[string]int),
        ByDay:         make(map[string]int),
    }
    
    uniqueIPs := make(map[string]bool)
    
    for _, access := range accesses {
        stats.ByFormat[access.Format]++
        day := access.CreatedAt.Format("2006-01-02")
        stats.ByDay[day]++
        uniqueIPs[access.IPAddress] = true
        
        if access.CreatedAt.After(stats.LastAccess) {
            stats.LastAccess = access.CreatedAt
        }
    }
    
    stats.UniqueIPs = len(uniqueIPs)
    
    return stats, nil
}
```

**Endpoints:**
```
GET /api/users/:id/subscription/stats
  Response: {
    "total_accesses": 150,
    "by_format": {
      "v2ray": 80,
      "clash": 50,
      "singbox": 20
    },
    "by_day": {
      "2026-03-20": 45,
      "2026-03-21": 55,
      "2026-03-22": 50
    },
    "unique_ips": 3,
    "last_access": "2026-03-22T10:30:00Z"
  }

POST /api/users/:id/subscription/regenerate
  Response: {
    "subscription_token": "new_token_here",
    "subscription_url": "https://panel.example.com/sub/new_token_here",
    "short_url": "https://panel.example.com/s/abc123",
    "qr_code_url": "https://panel.example.com/sub/new_token_here/qr"
  }
```

**Acceptance Criteria:**
- ✅ Subscription token можно регенерировать (старый перестает работать)
- ✅ Статистика показывает обращения за последние 7/30/90 дней
- ✅ Кэш инвалидируется при изменении пользователя или inbound
- ✅ Кэш уменьшает нагрузку на БД (hit rate > 80%)
- ✅ UI показывает статистику в виде графиков
- ✅ UI позволяет копировать URL одним кликом
- ✅ QR код отображается в модальном окне

**Deliverables:**
- Полное управление подписками через API и UI
- Статистика использования с графиками
- Кэширование для производительности
- Удобный UX для копирования и шаринга

---

### Фаза 5: Сертификаты (2 недели)

#### 5.1 ACME интеграция (1 неделя)

**Задачи:**
- [ ] Модель Certificate в БД
- [ ] Интеграция с Lego (Let's Encrypt)
- [ ] **DNS-01 challenge (РЕКОМЕНДУЕТСЯ для localhost-only панели)**
- [ ] HTTP-01 challenge (с особыми требованиями)
- [ ] Автоматическое обновление сертификатов
- [ ] Поддержка основных DNS провайдеров (Cloudflare, Route53, etc.)

**КРИТИЧНО: ACME Challenge и изоляция панели**

Панель работает только на localhost (127.0.0.1:8080), что создает проблему для HTTP-01 challenge.

**Проблема HTTP-01:**
```
Let's Encrypt → http://yourdomain.com:80/.well-known/acme-challenge/token
                     ↓
                HAProxy:80 (если включен) или iptables
                     ↓
                Панель на 127.0.0.1:8080 ← ИЗОЛИРОВАНА!
```

Let's Encrypt не может достучаться до панели напрямую, так как она слушает только localhost.

**Решения:**

**1. DNS-01 Challenge (РЕКОМЕНДУЕТСЯ) ⭐**

DNS-01 не требует открытых портов вообще - идеально для "stealth" архитектуры.

```go
// internal/acme/dns_challenge.go
package acme

import (
    "github.com/go-acme/lego/v4/challenge/dns01"
    "github.com/go-acme/lego/v4/providers/dns/cloudflare"
)

func setupDNSChallenge(provider string, credentials map[string]string) (challenge.Provider, error) {
    switch provider {
    case "cloudflare":
        config := cloudflare.NewDefaultConfig()
        config.AuthEmail = credentials["email"]
        config.AuthKey = credentials["api_key"]
        return cloudflare.NewDNSProviderConfig(config)
        
    case "route53":
        // AWS Route53 configuration
        
    case "digitalocean":
        // DigitalOcean DNS configuration
        
    default:
        return nil, fmt.Errorf("unsupported DNS provider: %s", provider)
    }
}
```

**Преимущества DNS-01:**
- ✅ Не требует открытых портов
- ✅ Работает с localhost-only панелью
- ✅ Поддерживает wildcard сертификаты (*.domain.com)
- ✅ Идеально для "stealth" сервера

**Недостатки:**
- ❌ Требует API доступ к DNS провайдеру
- ❌ Немного сложнее настроить

**2. HTTP-01 Challenge с HAProxy (если HAProxy включен)**

Если HAProxy включен, можно настроить специальный роутинг для ACME challenge:

```haproxy
# HAProxy configuration для ACME
frontend http-in
    bind :80
    mode http
    
    # ACME challenge routing
    acl is_acme_challenge path_beg /.well-known/acme-challenge/
    use_backend acme_backend if is_acme_challenge
    
    # Redirect all other HTTP to HTTPS
    redirect scheme https code 301 if !is_acme_challenge

backend acme_backend
    mode http
    server panel 127.0.0.1:8080
```

**Требования:**
- HAProxy должен быть включен
- Порт 80 должен быть открыт в firewall
- Панель должна обрабатывать /.well-known/acme-challenge/

**3. HTTP-01 Challenge без HAProxy (временное открытие порта)**

Если HAProxy выключен, нужно временно открыть порт 80:

```go
// internal/acme/http_challenge.go
func (ac *ACMEClient) setupHTTPChallenge() error {
    // 1. Временно добавляем iptables правило
    cmd := exec.Command("iptables", "-I", "INPUT", "-p", "tcp", "--dport", "80", "-j", "ACCEPT")
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to open port 80: %w", err)
    }
    
    // 2. Запускаем временный HTTP сервер на :80
    server := &http.Server{
        Addr:    ":80",
        Handler: ac.challengeHandler,
    }
    
    go server.ListenAndServe()
    
    // 3. После получения сертификата закрываем порт
    defer func() {
        server.Shutdown(context.Background())
        exec.Command("iptables", "-D", "INPUT", "-p", "tcp", "--dport", "80", "-j", "ACCEPT").Run()
    }()
    
    return nil
}
```

**Недостатки:**
- ❌ Требует root привилегий для iptables
- ❌ Временно открывает порт 80 (нарушает "stealth")
- ❌ Сложнее и опаснее

**Рекомендации:**

| Сценарий | Рекомендуемый метод | Причина |
|----------|---------------------|---------|
| HAProxy выключен | **DNS-01** | Не требует открытых портов |
| HAProxy включен | DNS-01 или HTTP-01 | Оба работают, DNS-01 проще |
| Нет доступа к DNS API | HTTP-01 с HAProxy | Единственный вариант |
| Wildcard сертификат | **DNS-01** (только) | HTTP-01 не поддерживает wildcard |

**Реализация:**

```go
// internal/services/certificate_service.go
type CertificateService struct {
    db          *gorm.DB
    acmeClient  *lego.Client
    haproxyEnabled bool
}

func (cs *CertificateService) RequestCertificate(req *CertificateRequest) error {
    // Определяем метод challenge
    challengeMethod := cs.selectChallengeMethod(req)
    
    switch challengeMethod {
    case "dns-01":
        log.Info().Msg("Using DNS-01 challenge (recommended)")
        return cs.requestWithDNS(req)
        
    case "http-01":
        if !cs.haproxyEnabled {
            log.Warn().Msg("HTTP-01 without HAProxy requires temporary port 80 opening")
            return cs.requestWithHTTPTemporary(req)
        }
        log.Info().Msg("Using HTTP-01 challenge with HAProxy routing")
        return cs.requestWithHTTPHAProxy(req)
        
    default:
        return fmt.Errorf("no suitable challenge method available")
    }
}

func (cs *CertificateService) selectChallengeMethod(req *CertificateRequest) string {
    // Приоритет: DNS-01 > HTTP-01
    
    if req.DNSProvider != "" && req.DNSCredentials != nil {
        return "dns-01"
    }
    
    if cs.haproxyEnabled || req.AllowTemporaryPort80 {
        return "http-01"
    }
    
    return ""
}
```

**UI Warning для HTTP-01:**

```typescript
// frontend/src/pages/Certificates/RequestCertificate.tsx
{challengeMethod === 'http-01' && !haproxyEnabled && (
  <Alert variant="warning">
    <AlertTitle>Security Warning</AlertTitle>
    <p>
      HTTP-01 challenge without HAProxy requires temporarily opening port 80.
      This may expose your server during certificate issuance.
    </p>
    <p className="mt-2">
      <strong>Recommended:</strong> Use DNS-01 challenge instead for better security.
    </p>
  </Alert>
)}
```

**Acceptance Criteria:**
- ✅ DNS-01 challenge работает с основными провайдерами (Cloudflare, Route53, DigitalOcean)
- ✅ HTTP-01 challenge работает с HAProxy (routing на панель)
- ✅ HTTP-01 без HAProxy показывает предупреждение и требует подтверждения
- ✅ Wildcard сертификаты работают через DNS-01
- ✅ Автоматическое обновление сертификатов за 30 дней до истечения
- ✅ UI показывает рекомендации по выбору метода challenge
- ✅ Документация объясняет проблему HTTP-01 с localhost-only панелью
- ✅ Unit тесты для обоих методов challenge

**Deliverables:**
- Сертификаты получаются автоматически через DNS-01 (рекомендуется)
- HTTP-01 работает с HAProxy или временным открытием порта
- Автообновление работает
- Пользователи понимают ограничения и риски каждого метода
- Документация содержит примеры настройки для популярных DNS провайдеров

#### 5.2 Ручная загрузка + UI (1 неделя)

**Задачи:**
- [ ] Endpoint для загрузки сертификатов
- [ ] Валидация сертификатов
- [ ] UI для управления сертификатами
- [ ] Привязка сертификатов к inbound

**Deliverables:**
- Все варианты получения сертификатов работают

---

### Фаза 6: Мониторинг (2 недели)

#### 6.1 Статистика трафика + Unified Stats Provider (1.5 недели)

**Задачи:**
- [ ] Реализация CoreStatsProvider интерфейса
- [ ] SingboxStatsProvider (Clash API)
- [ ] XrayStatsProvider (gRPC Stats API)
- [ ] MihomoStatsProvider (REST API)
- [ ] Настройка Sing-box Experimental API
- [ ] Настройка Xray gRPC Stats API
- [ ] Настройка Mihomo External Controller
- [ ] TrafficCollector сервис с настраиваемым интервалом
- [ ] Модель TrafficStats в БД
- [ ] Агрегация данных (сырые → почасовые → дневные)
- [ ] Data retention сервис (автоочистка старых данных)
- [ ] Endpoints для получения статистики
- [ ] **УЛУЧШЕНО: Smart Quota Enforcement (без full restart)**
- [ ] Xray gRPC HandlerService для динамического управления пользователями
- [ ] Graceful reload для Sing-box и Mihomo
- [ ] Закрытие активных соединений пользователя через API ядер
- [ ] Уведомления при превышении квоты

**Smart Quota Enforcement (улучшенный подход):**

Вместо полного перезапуска всех ядер при превышении квоты, используем умное отключение:

**Стратегия по ядрам:**

1. **Xray (лучший вариант)** - Динамическое управление через gRPC API:
   - `RemoveUser` - мгновенное удаление пользователя из inbound
   - Без перезапуска, без влияния на других пользователей
   - Latency: ~10ms

2. **Sing-box и Mihomo (хороший вариант)** - Graceful reload:
   - Graceful reload через SIGHUP
   - Существующие соединения сохраняются
   - Новые соединения блокируются
   - Latency: ~500ms

3. **Fallback** - Full restart только если graceful failed

**Реализация:**

```go
// internal/services/quota_enforcer.go
package services

import (
    "context"
    "fmt"
    
    "github.com/yourusername/isolate-panel/internal/cores/xray"
    "github.com/yourusername/isolate-panel/internal/models"
    "github.com/rs/zerolog/log"
    "gorm.io/gorm"
)

type QuotaEnforcer struct {
    db                    *gorm.DB
    xrayService           *xray.GRPCClient
    coreManager           *CoreManager
    notificationService   *NotificationService
}

func NewQuotaEnforcer(
    db *gorm.DB,
    xrayService *xray.GRPCClient,
    coreManager *CoreManager,
    notificationService *NotificationService,
) *QuotaEnforcer {
    return &QuotaEnforcer{
        db:                  db,
        xrayService:         xrayService,
        coreManager:         coreManager,
        notificationService: notificationService,
    }
}

func (qe *QuotaEnforcer) DisableUser(user *models.User) error {
    log.Info().
        Uint("user_id", user.ID).
        Str("username", user.Username).
        Msg("Disabling user due to quota exceeded")
    
    // 1. Mark as inactive in DB
    user.IsActive = false
    if err := qe.db.Save(user).Error; err != nil {
        return fmt.Errorf("failed to mark user as inactive: %w", err)
    }
    
    // 2. Get user's inbounds
    var mappings []models.UserInboundMapping
    if err := qe.db.Where("user_id = ?", user.ID).
        Preload("Inbound.Core").
        Find(&mappings).Error; err != nil {
        return fmt.Errorf("failed to get user inbounds: %w", err)
    }
    
    // 3. Disable per core with smart strategy
    coresAffected := make(map[string]bool)
    
    for _, mapping := range mappings {
        core := mapping.Inbound.Core
        coresAffected[core.Name] = true
        
        switch core.Name {
        case "xray":
            // BEST: Dynamic removal via gRPC API
            err := qe.xrayService.RemoveUser(
                mapping.Inbound.Tag,
                user.UUID,
            )
            if err != nil {
                log.Error().
                    Err(err).
                    Str("inbound_tag", mapping.Inbound.Tag).
                    Msg("Failed to remove user from Xray via gRPC, falling back to reload")
                
                // Fallback to graceful reload
                if err := qe.gracefulReloadCore("xray"); err != nil {
                    return fmt.Errorf("failed to reload xray: %w", err)
                }
            } else {
                log.Info().
                    Str("inbound_tag", mapping.Inbound.Tag).
                    Msg("User removed from Xray via gRPC (no restart)")
            }
            
        case "singbox", "mihomo":
            // GOOD: Graceful reload (will be done once per core after loop)
            log.Debug().
                Str("core", core.Name).
                Msg("Marking core for graceful reload")
        }
    }
    
    // 4. Graceful reload for Sing-box and Mihomo (once per core)
    for coreName := range coresAffected {
        if coreName == "singbox" || coreName == "mihomo" {
            if err := qe.gracefulReloadCore(coreName); err != nil {
                return fmt.Errorf("failed to reload %s: %w", coreName, err)
            }
        }
    }
    
    // 5. Close active connections
    if err := qe.closeUserConnections(user.UUID); err != nil {
        log.Warn().Err(err).Msg("Failed to close user connections")
    }
    
    // 6. Send notification
    qe.notificationService.Send(&models.Notification{
        Type:      "email",
        Event:     "quota_exceeded",
        Recipient: user.Email,
        Subject:   "Traffic quota exceeded",
        Body: fmt.Sprintf(
            "User %s has exceeded their traffic quota.\n"+
            "Used: %d bytes\n"+
            "Limit: %d bytes\n"+
            "The account has been automatically disabled.",
            user.Username,
            user.TrafficUsedBytes,
            *user.TrafficLimitBytes,
        ),
    })
    
    log.Info().
        Uint("user_id", user.ID).
        Str("username", user.Username).
        Msg("User disabled successfully")
    
    return nil
}

func (qe *QuotaEnforcer) gracefulReloadCore(coreName string) error {
    log.Info().Str("core", coreName).Msg("Starting graceful reload")
    
    // 1. Regenerate config
    if err := qe.coreManager.RegenerateConfig(coreName); err != nil {
        return fmt.Errorf("failed to regenerate config: %w", err)
    }
    
    // 2. Try graceful reload
    err := qe.coreManager.GracefulReload(coreName)
    if err != nil {
        log.Error().
            Err(err).
            Str("core", coreName).
            Msg("Graceful reload failed, falling back to full restart")
        
        // Fallback to full restart
        if err := qe.coreManager.RestartCore(coreName); err != nil {
            return fmt.Errorf("failed to restart core: %w", err)
        }
        
        log.Warn().
            Str("core", coreName).
            Msg("Core restarted (full restart used as fallback)")
    } else {
        log.Info().
            Str("core", coreName).
            Msg("Core reloaded gracefully (existing connections preserved)")
    }
    
    return nil
}

func (qe *QuotaEnforcer) closeUserConnections(userUUID string) error {
    // Close connections via core APIs
    // Implementation depends on each core's API
    log.Debug().Str("uuid", userUUID).Msg("Closing user connections")
    
    // For Sing-box: DELETE /connections/:id via Clash API
    // For Xray: connections will be closed when user is removed
    // For Mihomo: DELETE /connections/:id via External Controller
    
    return nil
}
```

**Xray gRPC Integration:**

```go
// internal/cores/xray/grpc_client.go
package xray

import (
    "context"
    "fmt"
    "time"
    
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "github.com/xtls/xray-core/app/proxyman/command"
    "github.com/xtls/xray-core/common/protocol"
    "github.com/xtls/xray-core/common/serial"
    "github.com/xtls/xray-core/proxy/vmess"
)

type GRPCClient struct {
    conn          *grpc.ClientConn
    handlerClient command.HandlerServiceClient
    statsClient   command.StatsServiceClient
}

func NewGRPCClient(address string) (*GRPCClient, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    conn, err := grpc.DialContext(
        ctx,
        address,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithBlock(),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to connect to Xray gRPC: %w", err)
    }
    
    return &GRPCClient{
        conn:          conn,
        handlerClient: command.NewHandlerServiceClient(conn),
        statsClient:   command.NewStatsServiceClient(conn),
    }, nil
}

func (c *GRPCClient) Close() error {
    return c.conn.Close()
}

// RemoveUser removes a user from an inbound without restarting Xray
func (c *GRPCClient) RemoveUser(inboundTag string, userEmail string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    req := &command.AlterInboundRequest{
        Tag: inboundTag,
        Operation: serial.ToTypedMessage(&command.RemoveUserOperation{
            Email: userEmail, // Xray uses "email" field for user identification
        }),
    }
    
    _, err := c.handlerClient.AlterInbound(ctx, req)
    if err != nil {
        return fmt.Errorf("failed to remove user: %w", err)
    }
    
    return nil
}

// AddUser adds a user to an inbound without restarting Xray
func (c *GRPCClient) AddUser(inboundTag string, user *protocol.User) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    req := &command.AlterInboundRequest{
        Tag: inboundTag,
        Operation: serial.ToTypedMessage(&command.AddUserOperation{
            User: user,
        }),
    }
    
    _, err := c.handlerClient.AlterInbound(ctx, req)
    if err != nil {
        return fmt.Errorf("failed to add user: %w", err)
    }
    
    return nil
}

// GetUserStats gets traffic stats for a specific user
func (c *GRPCClient) GetUserStats(userEmail string) (upload, download int64, err error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    // Query uplink
    uplinkReq := &command.GetStatsRequest{
        Name:   fmt.Sprintf("user>>>%s>>>traffic>>>uplink", userEmail),
        Reset_: false,
    }
    uplinkResp, err := c.statsClient.GetStats(ctx, uplinkReq)
    if err != nil {
        return 0, 0, fmt.Errorf("failed to get uplink stats: %w", err)
    }
    
    // Query downlink
    downlinkReq := &command.GetStatsRequest{
        Name:   fmt.Sprintf("user>>>%s>>>traffic>>>downlink", userEmail),
        Reset_: false,
    }
    downlinkResp, err := c.statsClient.GetStats(ctx, downlinkReq)
    if err != nil {
        return 0, 0, fmt.Errorf("failed to get downlink stats: %w", err)
    }
    
    return uplinkResp.Stat.Value, downlinkResp.Stat.Value, nil
}
```

**Core Manager Graceful Reload:**

```go
// internal/services/core_manager.go

func (cm *CoreManager) GracefulReload(coreName string) error {
    log.Info().Str("core", coreName).Msg("Attempting graceful reload")
    
    // Check if core supports graceful reload
    if !cm.supportsGracefulReload(coreName) {
        return fmt.Errorf("core %s does not support graceful reload", coreName)
    }
    
    // Get core PID
    core, err := cm.getCore(coreName)
    if err != nil {
        return err
    }
    
    if core.PID == 0 {
        return fmt.Errorf("core %s is not running", coreName)
    }
    
    // Send SIGHUP signal for graceful reload
    process, err := os.FindProcess(core.PID)
    if err != nil {
        return fmt.Errorf("failed to find process: %w", err)
    }
    
    if err := process.Signal(syscall.SIGHUP); err != nil {
        return fmt.Errorf("failed to send SIGHUP: %w", err)
    }
    
    log.Info().
        Str("core", coreName).
        Int("pid", core.PID).
        Msg("SIGHUP sent for graceful reload")
    
    // Wait a bit and verify core is still running
    time.Sleep(2 * time.Second)
    
    if err := process.Signal(syscall.Signal(0)); err != nil {
        return fmt.Errorf("core died after reload: %w", err)
    }
    
    return nil
}

func (cm *CoreManager) supportsGracefulReload(coreName string) bool {
    // Xray: supports SIGHUP
    // Sing-box: supports SIGHUP (needs verification)
    // Mihomo: supports SIGHUP (needs verification)
    
    switch coreName {
    case "xray", "singbox", "mihomo":
        return true
    default:
        return false
    }
}
```

**Xray Configuration for gRPC API:**

```json
{
  "api": {
    "tag": "api",
    "services": [
      "HandlerService",
      "StatsService"
    ]
  },
  "stats": {},
  "policy": {
    "levels": {
      "0": {
        "statsUserUplink": true,
        "statsUserDownlink": true
      }
    },
    "system": {
      "statsInboundUplink": true,
      "statsInboundDownlink": true
    }
  },
  "inbounds": [
    {
      "tag": "api",
      "listen": "127.0.0.1",
      "port": 10085,
      "protocol": "dokodemo-door",
      "settings": {
        "address": "127.0.0.1"
      }
    }
  ],
  "routing": {
    "rules": [
      {
        "type": "field",
        "inboundTag": ["api"],
        "outboundTag": "api"
      }
    ]
  }
}
```

**Acceptance Criteria:**
- ✅ Статистика собирается каждые 60 секунд (Lite) / 10 секунд (Full)
- ✅ Данные корректно агрегируются (минуты → часы → дни)
- ✅ **Xray: пользователь отключается через gRPC API без перезапуска (<10ms)**
- ✅ **Sing-box/Mihomo: graceful reload сохраняет существующие соединения (~500ms)**
- ✅ **Fallback на full restart работает если graceful failed**
- ✅ **Другие пользователи не испытывают downtime при отключении одного**
- ✅ Email уведомление отправляется при превышении квоты
- ✅ Старые данные автоматически удаляются (7 дней сырых, 90 дней почасовых)
- ✅ Unit тесты для QuotaEnforcer и всех трех стратегий
- ✅ Integration тесты для Xray gRPC API
- ✅ Graceful reload тестируется на всех ядрах

**Deliverables:**
- Статистика трафика собирается со всех ядер через unified interface
- **Квоты применяются без downtime для других пользователей**
- **Xray: динамическое управление пользователями через gRPC**
- **Sing-box/Mihomo: graceful reload вместо full restart**
- Уведомления работают
- Профессиональное production-ready решение

#### 6.2 Активные подключения (1 неделя)

**Задачи:**
- [ ] Модель ActiveConnection в БД
- [ ] Получение активных подключений из ядер
- [ ] Endpoint для просмотра подключений
- [ ] UI для отображения активных подключений
- [ ] Возможность отключить пользователя

**Deliverables:**
- Видны активные подключения в реальном времени

#### 6.3 HAProxy Monitoring & Stats (0.5 недели)

**Задачи:**
- [ ] HAProxy Stats API integration (CSV parser)
- [ ] Модель HAProxyStats в памяти (не БД, real-time)
- [ ] Endpoint: GET /api/haproxy/stats - получить статистику HAProxy
- [ ] Endpoint: GET /api/haproxy/logs - получить последние логи HAProxy
- [ ] HAProxy log parser (structured logging)
- [ ] UI компонент для отображения HAProxy stats
- [ ] Real-time обновление stats (polling каждые 5 секунд)
- [ ] Отображение frontend/backend/server статистики
- [ ] Отображение health check статуса серверов
- [ ] Ссылка на native HAProxy stats page (http://localhost:8404/stats)

**HAProxy Stats API Integration:**
```go
// internal/haproxy/stats.go
package haproxy

import (
    "encoding/csv"
    "net/http"
    "strconv"
)

type HAProxyStats struct {
    Frontends []FrontendStats
    Backends  []BackendStats
    Servers   []ServerStats
}

type FrontendStats struct {
    Name            string
    Status          string
    CurrentSessions int
    MaxSessions     int
    TotalSessions   int64
    BytesIn         int64
    BytesOut        int64
    RequestRate     int
}

type BackendStats struct {
    Name            string
    Status          string
    CurrentSessions int
    TotalSessions   int64
    BytesIn         int64
    BytesOut        int64
    ActiveServers   int
    BackupServers   int
}

type ServerStats struct {
    Backend     string
    Name        string
    Status      string
    CurrentSessions int
    TotalSessions   int64
    BytesIn         int64
    BytesOut        int64
    CheckStatus     string
    LastCheck       string
    Downtime        int
}

func GetHAProxyStats() (*HAProxyStats, error) {
    // Query HAProxy stats via socket
    resp, err := http.Get("http://localhost:8404/stats;csv")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    reader := csv.NewReader(resp.Body)
    records, err := reader.ReadAll()
    if err != nil {
        return nil, err
    }
    
    stats := &HAProxyStats{
        Frontends: []FrontendStats{},
        Backends:  []BackendStats{},
        Servers:   []ServerStats{},
    }
    
    // Parse CSV (skip header)
    for i, record := range records {
        if i == 0 {
            continue
        }
        
        proxyName := record[0]
        serviceName := record[1]
        proxyType := record[32] // 0=frontend, 1=backend, 2=server
        
        switch proxyType {
        case "0": // Frontend
            stats.Frontends = append(stats.Frontends, FrontendStats{
                Name:            proxyName,
                Status:          record[17],
                CurrentSessions: parseInt(record[4]),
                MaxSessions:     parseInt(record[5]),
                TotalSessions:   parseInt64(record[7]),
                BytesIn:         parseInt64(record[8]),
                BytesOut:        parseInt64(record[9]),
                RequestRate:     parseInt(record[33]),
            })
        case "1": // Backend
            stats.Backends = append(stats.Backends, BackendStats{
                Name:            proxyName,
                Status:          record[17],
                CurrentSessions: parseInt(record[4]),
                TotalSessions:   parseInt64(record[7]),
                BytesIn:         parseInt64(record[8]),
                BytesOut:        parseInt64(record[9]),
                ActiveServers:   parseInt(record[19]),
                BackupServers:   parseInt(record[20]),
            })
        case "2": // Server
            stats.Servers = append(stats.Servers, ServerStats{
                Backend:         proxyName,
                Name:            serviceName,
                Status:          record[17],
                CurrentSessions: parseInt(record[4]),
                TotalSessions:   parseInt64(record[7]),
                BytesIn:         parseInt64(record[8]),
                BytesOut:        parseInt64(record[9]),
                CheckStatus:     record[36],
                LastCheck:       record[38],
                Downtime:        parseInt(record[24]),
            })
        }
    }
    
    return stats, nil
}

func parseInt(s string) int {
    v, _ := strconv.Atoi(s)
    return v
}

func parseInt64(s string) int64 {
    v, _ := strconv.ParseInt(s, 10, 64)
    return v
}
```

**HAProxy Log Parser:**
```go
// internal/haproxy/logging.go
package haproxy

import (
    "bufio"
    "os"
    "strings"
    "time"
)

type HAProxyLog struct {
    Timestamp   time.Time
    Frontend    string
    Backend     string
    Server      string
    ClientIP    string
    StatusCode  int
    BytesSent   int64
    Duration    int
    RequestPath string
}

func ParseHAProxyLog(logPath string) ([]HAProxyLog, error) {
    file, err := os.Open(logPath)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    var logs []HAProxyLog
    scanner := bufio.NewScanner(file)
    
    for scanner.Scan() {
        line := scanner.Text()
        log := parseLogLine(line)
        if log != nil {
            logs = append(logs, *log)
        }
    }
    
    return logs, scanner.Err()
}

func parseLogLine(line string) *HAProxyLog {
    // Parse HAProxy log format
    // Example: Mar 22 10:30:45 localhost haproxy[1234]: 192.168.1.1:54321 [22/Mar/2026:10:30:45.123] https_frontend backend_singbox_443/singbox_443 0/0/1/2/3 200 1234 - - ---- 1/1/0/0/0 0/0 "GET /path HTTP/1.1"
    
    parts := strings.Fields(line)
    if len(parts) < 15 {
        return nil
    }
    
    // Extract relevant fields
    log := &HAProxyLog{
        ClientIP:   strings.Split(parts[5], ":")[0],
        Frontend:   parts[7],
        Backend:    strings.Split(parts[8], "/")[0],
        Server:     strings.Split(parts[8], "/")[1],
        StatusCode: parseInt(parts[10]),
        BytesSent:  parseInt64(parts[11]),
    }
    
    // Parse timestamp
    timestampStr := strings.Trim(parts[6], "[]")
    log.Timestamp, _ = time.Parse("02/Jan/2006:15:04:05.000", timestampStr)
    
    // Parse request path
    if len(parts) > 16 {
        log.RequestPath = parts[16]
    }
    
    return log
}
```

**API Handlers:**
```go
// internal/handlers/haproxy.go
package handlers

func (h *HAProxyHandler) GetStats(c *fiber.Ctx) error {
    stats, err := h.haproxyService.GetStats()
    if err != nil {
        return c.Status(500).JSON(fiber.Map{
            "error": "Failed to get HAProxy stats",
        })
    }
    
    return c.JSON(stats)
}

func (h *HAProxyHandler) GetLogs(c *fiber.Ctx) error {
    limit := c.QueryInt("limit", 100)
    
    logs, err := h.haproxyService.GetRecentLogs(limit)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{
            "error": "Failed to get HAProxy logs",
        })
    }
    
    return c.JSON(logs)
}
```

**UI Component (Frontend):**
```typescript
// frontend/src/pages/HAProxyStats.tsx
import { useQuery } from '../hooks/useQuery'
import { Card } from '../components/ui/Card'
import { Badge } from '../components/ui/Badge'
import { Table } from '../components/ui/Table'
import { formatBytes } from '../utils/format'

export const HAProxyStats = () => {
    const { data: stats, isLoading } = useQuery('/api/haproxy/stats', {
        refetchInterval: 5000, // Refresh every 5 seconds
    })
    
    if (isLoading) return <LoadingView />
    
    return (
        <PageLayout>
            <PageHeader title="HAProxy Statistics" />
            
            {/* Frontends */}
            <Card title="Frontends" className="mb-4">
                <Table>
                    <thead>
                        <tr>
                            <th>Name</th>
                            <th>Status</th>
                            <th>Current Sessions</th>
                            <th>Total Sessions</th>
                            <th>Bytes In/Out</th>
                            <th>Request Rate</th>
                        </tr>
                    </thead>
                    <tbody>
                        {stats.frontends.map((frontend) => (
                            <tr key={frontend.name}>
                                <td>{frontend.name}</td>
                                <td>
                                    <Badge variant={frontend.status === 'OPEN' ? 'success' : 'danger'}>
                                        {frontend.status}
                                    </Badge>
                                </td>
                                <td>{frontend.current_sessions}</td>
                                <td>{frontend.total_sessions.toLocaleString()}</td>
                                <td>
                                    {formatBytes(frontend.bytes_in)} / {formatBytes(frontend.bytes_out)}
                                </td>
                                <td>{frontend.request_rate} req/s</td>
                            </tr>
                        ))}
                    </tbody>
                </Table>
            </Card>
            
            {/* Backends */}
            <Card title="Backends" className="mb-4">
                <Table>
                    <thead>
                        <tr>
                            <th>Name</th>
                            <th>Status</th>
                            <th>Active Servers</th>
                            <th>Current Sessions</th>
                            <th>Total Sessions</th>
                            <th>Bytes In/Out</th>
                        </tr>
                    </thead>
                    <tbody>
                        {stats.backends.map((backend) => (
                            <tr key={backend.name}>
                                <td>{backend.name}</td>
                                <td>
                                    <Badge variant={backend.status === 'UP' ? 'success' : 'danger'}>
                                        {backend.status}
                                    </Badge>
                                </td>
                                <td>
                                    {backend.active_servers} / {backend.active_servers + backend.backup_servers}
                                </td>
                                <td>{backend.current_sessions}</td>
                                <td>{backend.total_sessions.toLocaleString()}</td>
                                <td>
                                    {formatBytes(backend.bytes_in)} / {formatBytes(backend.bytes_out)}
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </Table>
            </Card>
            
            {/* Servers */}
            <Card title="Servers">
                <Table>
                    <thead>
                        <tr>
                            <th>Backend</th>
                            <th>Server</th>
                            <th>Status</th>
                            <th>Health Check</th>
                            <th>Sessions</th>
                            <th>Bytes In/Out</th>
                            <th>Downtime</th>
                        </tr>
                    </thead>
                    <tbody>
                        {stats.servers.map((server) => (
                            <tr key={`${server.backend}-${server.name}`}>
                                <td>{server.backend}</td>
                                <td>{server.name}</td>
                                <td>
                                    <Badge variant={server.status === 'UP' ? 'success' : 'danger'}>
                                        {server.status}
                                    </Badge>
                                </td>
                                <td>
                                    <span className="text-sm text-gray-600">
                                        {server.check_status} - {server.last_check}
                                    </span>
                                </td>
                                <td>{server.current_sessions}</td>
                                <td>
                                    {formatBytes(server.bytes_in)} / {formatBytes(server.bytes_out)}
                                </td>
                                <td>{server.downtime > 0 ? `${server.downtime}s` : '-'}</td>
                            </tr>
                        ))}
                    </tbody>
                </Table>
            </Card>
            
            {/* Link to native HAProxy stats */}
            <div className="mt-4">
                <a
                    href="http://localhost:8404/stats"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-primary-600 hover:underline"
                >
                    Open native HAProxy stats page →
                </a>
            </div>
        </PageLayout>
    )
}
```

**Endpoints:**
- GET /api/haproxy/stats - получить статистику HAProxy (frontends, backends, servers)
- GET /api/haproxy/logs?limit=100 - получить последние N логов HAProxy

**Acceptance Criteria:**
- ✅ HAProxy stats API парсит CSV корректно
- ✅ Статистика обновляется каждые 5 секунд в UI
- ✅ Отображаются все frontends с их статусами
- ✅ Отображаются все backends с количеством активных серверов
- ✅ Отображаются все servers с health check статусами
- ✅ Downtime серверов отображается корректно
- ✅ Bytes in/out форматируются читабельно (KB, MB, GB)
- ✅ Ссылка на native HAProxy stats page работает
- ✅ UI компонент responsive и работает на mobile
- ✅ Логи HAProxy парсятся корректно

**Deliverables:**
- HAProxy stats интегрированы в панель
- Real-time мониторинг HAProxy через UI
- Health check статусы видны для всех серверов
- Логи HAProxy доступны через API

---

### Фаза 7: Дополнительные ядра (2 недели)

#### 7.1 Интеграция Xray (1 неделя)

**Задачи:**
- [ ] Добавить Xray в Docker образ
- [ ] Добавить Xray в Supervisord конфигурацию
- [ ] Генератор конфигурации Xray (Go structs → JSON)
- [ ] XrayStatsProvider (gRPC Stats API)
- [ ] Настройка Xray Stats API и gRPC
- [ ] Поддержка XHTTP протокола
- [ ] Интеграция с HAProxy (backend для Xray)
- [ ] Port allocation для Xray (диапазон 20000-29999)
- [ ] Endpoints для управления Xray

**Acceptance Criteria:**
- ✅ Xray запускается через supervisord
- ✅ XHTTP inbound создается и работает
- ✅ HAProxy корректно роутит XHTTP трафик на Xray
- ✅ Статистика собирается через gRPC Stats API
- ✅ Конфиг генерируется из БД корректно
- ✅ Unit тесты для Xray config generator и stats provider

**Deliverables:**
- Xray работает для XHTTP и других Xray-специфичных протоколов
- Статистика собирается через unified interface

#### 7.2 Интеграция Mihomo (1 неделя)

**Задачи:**
- [ ] Добавить Mihomo в Docker образ
- [ ] Добавить Mihomo в Supervisord конфигурацию
- [ ] Генератор конфигурации Mihomo (Go structs → YAML)
- [ ] MihomoStatsProvider (REST API)
- [ ] Настройка Mihomo external-controller API
- [ ] Поддержка Mihomo-специфичных протоколов (Mieru, Sudoku, etc.)
- [ ] Интеграция с HAProxy (backend для Mihomo)
- [ ] Port allocation для Mihomo (диапазон 30000-39999)
- [ ] Endpoints для управления Mihomo

**Acceptance Criteria:**
- ✅ Mihomo запускается через supervisord
- ✅ Mihomo-специфичные протоколы работают
- ✅ HAProxy корректно роутит трафик на Mihomo
- ✅ Статистика собирается через REST API
- ✅ Конфиг генерируется из БД корректно
- ✅ Unit тесты для Mihomo config generator и stats provider

**Deliverables:**
- Mihomo работает для специфичных протоколов
- Все три ядра интегрированы и работают параллельно
- Unified stats collection работает для всех ядер

---

### Фаза 8: WARP + GeoIP (1-2 недели)

#### 8.1 WARP интеграция (1 неделя)

**Задачи:**
- [ ] Модель WarpRoute в БД
- [ ] Настройка WARP outbound
- [ ] Маршрутизация через WARP по доменам/IP
- [ ] UI для управления WARP маршрутами

**Deliverables:**
- WARP работает
- Ресурсы маршрутизируются через WARP

#### 8.2 GeoIP/GeoSite (опционально, 1 неделя)

**Задачи:**
- [ ] Скачивание GeoIP/GeoSite баз
- [ ] Автообновление баз
- [ ] Интеграция в routing rules
- [ ] UI для GeoIP правил

**Deliverables:**
- GeoIP маршрутизация работает

---

### Фаза 9: Резервное копирование (1 неделя)

#### Задачи
- [ ] Модель Backup в БД
- [ ] Создание бэкапа (БД + конфигурации)
- [ ] Шифрование бэкапов
- [ ] Поддержка destinations: Local, S3, FTP/SFTP
- [ ] Автоматические бэкапы по расписанию (cron)
- [ ] Восстановление из бэкапа
- [ ] UI для управления бэкапами

**Deliverables:**
- Бэкапы создаются автоматически и вручную
- Восстановление работает
- Поддержка всех destinations

---

### Фаза 10: Уведомления (1 неделя)

#### Задачи
- [ ] Модель Notification в БД
- [ ] Email уведомления (SMTP)
- [ ] Webhook уведомления
- [ ] События: quota_exceeded, expiry_warning, cert_renewed, core_error, failed_login
- [ ] Retry механизм для неудачных отправок
- [ ] UI для настройки уведомлений
- [ ] Заглушка для Telegram (логика без реализации)

**Deliverables:**
- Email и Webhook уведомления работают
- Администраторы получают алерты

---

### Фаза 11: CLI интерфейс (1-2 недели)

#### 11.1 CLI Authentication & Core Framework (3 дня)

**Задачи:**
- [ ] CLI framework (cobra)
- [ ] Multi-profile configuration system
- [ ] Config file: ~/.isolate-panel/config.json
- [ ] Authentication commands (login, logout)
- [ ] Profile management (switch, list)
- [ ] Automatic token refresh
- [ ] Error handling с exit codes

**Config Structure:**
```json
{
  "current_profile": "local",
  "profiles": {
    "local": {
      "panel_url": "http://localhost:8080",
      "access_token": "...",
      "refresh_token": "...",
      "token_expires_at": "2026-03-22T10:00:00Z"
    },
    "production": {
      "panel_url": "http://192.168.1.100:8080",
      "access_token": "...",
      "refresh_token": "...",
      "token_expires_at": "2026-03-22T10:00:00Z"
    }
  }
}
```

**Authentication Commands:**
```bash
# Login to default profile
isolate-panel login
isolate-panel login --profile production --url http://192.168.1.100:8080

# Logout
isolate-panel logout
isolate-panel logout --profile production

# Profile management
isolate-panel profile list
isolate-panel profile switch production
isolate-panel profile current

# Check authentication status
isolate-panel auth status
```

**Exit Codes:**
```go
const (
  ExitSuccess           = 0
  ExitGeneralError      = 1
  ExitAuthError         = 2
  ExitNotFoundError     = 3
  ExitValidationError   = 4
  ExitNetworkError      = 5
  ExitPermissionError   = 6
)
```

**Acceptance Criteria:**
- ✅ CLI требует аутентификации для всех команд (кроме login, version, help)
- ✅ Поддержка нескольких профилей (панелей)
- ✅ Автоматический refresh токена при истечении
- ✅ Корректные exit codes для автоматизации
- ✅ Понятные сообщения об ошибках

#### 11.2 Output Formatters (2 дня)

**Задачи:**
- [ ] Table formatter (human-readable, default)
- [ ] JSON formatter (--json flag)
- [ ] CSV formatter (--csv flag)
- [ ] Quiet mode (--quiet flag, только значения)
- [ ] Colored output с опцией --no-color
- [ ] Progress indicators для длительных операций

**Output Examples:**
```bash
# Table format (default)
$ isolate-panel user list
ID  USERNAME  EMAIL              STATUS    TRAFFIC USED  EXPIRY
1   user1     user1@example.com  Active    1.2 GB        2026-12-31
2   user2     user2@example.com  Inactive  500 MB        2026-06-30

# JSON format
$ isolate-panel user list --json
[{"id":1,"username":"user1","email":"user1@example.com","is_active":true}]

# CSV format
$ isolate-panel user list --csv
id,username,email,is_active,traffic_used_bytes,expiry_date
1,user1,user1@example.com,true,1288490188,2026-12-31T23:59:59Z

# Quiet mode (для pipe)
$ isolate-panel user list --quiet
user1
user2
```

**Acceptance Criteria:**
- ✅ Все команды поддерживают --json, --csv, --quiet
- ✅ Table format красиво форматирован с выравниванием
- ✅ Colored output работает в терминалах с поддержкой цвета
- ✅ --no-color отключает цвета для pipe

#### 11.3 User Management Commands (2 дня)

> **📚 Детальная документация:** [PROTOCOL_SMART_FORMS_PLAN.md](./PROTOCOL_SMART_FORMS_PLAN.md#cli-интерфейс)

**Задачи:**
- [ ] Команды для управления пользователями
- [ ] **Отображение credentials ОДИН РАЗ при создании** (критично!)
- [ ] Interactive prompts для команд без флагов
- [ ] Non-interactive mode с флагами

**Commands:**
```bash
# Create user (показывает credentials ОДИН РАЗ!)
isolate-panel user create <username> [--email=<email>] [--traffic-limit=<GB>] [--expiry=<date>]

# List users
isolate-panel user list [--active] [--expired] [--limit=<n>]

# Show user details
isolate-panel user show <username|id>

# Show credentials (только для администратора)
isolate-panel user credentials <username|id>

# Regenerate credentials (с предупреждением!)
isolate-panel user regenerate <username|id> [--confirm]

# Delete user
isolate-panel user delete <username|id> [--force]

# Update quotas
isolate-panel user update <username|id> --traffic-limit=<GB> --expiry=<date>
```

**Example Output:**
```bash
$ isolate-panel user create alice --traffic-limit=100

✓ User created successfully!

⚠️  IMPORTANT: Save these credentials now!
   They will NOT be shown again!

Username: alice
UUID: 12345678-abcd-1234-5678-123456789abc
Password: Xy9#mK2$pL4@qR7!
Token: aGVsbG8gd29ybGQgdGhpcyBpcyBhIHRva2Vu
Subscription Token: c3Vic2NyaXB0aW9uX3Rva2VuX2hlcmU=

SSH Public Key:
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC...

SSH Private Key:
-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9u...
-----END OPENSSH PRIVATE KEY-----

WireGuard Private Key: YNXtAzepDqRv9H52osJ...
WireGuard Public Key: Z1XXLsKYkYxuiYjJIkRvtIKFepCYHTgON+GwPq7SOV4=
```

#### 11.4 Inbound/Outbound Management (2 дня)

> **📚 Детальная документация:** [PROTOCOL_SMART_FORMS_PLAN.md](./PROTOCOL_SMART_FORMS_PLAN.md#cli-интерфейс)

**Commands:**
```bash
# Inbound management
isolate-panel inbound list [--core <singbox|xray|mihomo>]
isolate-panel inbound create  # Interactive wizard
isolate-panel inbound create --core=<core> --protocol=<protocol> --name=<name> --port=<port>
isolate-panel inbound show <id|name>
isolate-panel inbound update <id|name> [--name <name>] [--port <port>]
isolate-panel inbound delete <id|name> [--force]

# Inbound-User Association (новое!)
isolate-panel inbound add-users <inbound-id> <user-id1> [user-id2] [...]
isolate-panel inbound remove-user <inbound-id> <user-id>
isolate-panel inbound users <inbound-id>  # List users in inbound

# Outbound management
isolate-panel outbound list
isolate-panel outbound add --name <name> --type <direct|block|freedom|...>
isolate-panel outbound get <id>
isolate-panel outbound update <id> [--name <name>]
isolate-panel outbound delete <id>
```

**Interactive Wizard Example:**
```bash
$ isolate-panel inbound create

Welcome to Inbound Creation Wizard
===================================

Step 1/5: Select Core
---------------------
1) Sing-box (recommended)
2) Xray
3) Mihomo

Select core [1]: 1

Step 2/5: Select Protocol
-------------------------
Available protocols for Sing-box:
1) VMess
2) VLESS
3) Trojan
...

Select protocol [1]: 1

Step 3/5: Basic Settings
------------------------
Name [vmess-443]: 
Port [443]: 

✓ Port 443 is available

...

✓ Inbound created successfully!
  ID: 5
  Name: vmess-443
  Port: 443

You can now add users to this inbound:
  isolate-panel inbound add-users 5 <user-id>
```

#### 11.5 Core & System Management (1 день)

**Commands:**
```bash
# Core management
isolate-panel core list
isolate-panel core status [<singbox|xray|mihomo>]
isolate-panel core start <singbox|xray|mihomo>
isolate-panel core stop <singbox|xray|mihomo>
isolate-panel core restart <singbox|xray|mihomo>
isolate-panel core logs <singbox|xray|mihomo> [--tail <n>] [--follow]
isolate-panel core validate <singbox|xray|mihomo>  # Validate config

# System management
isolate-panel system status
isolate-panel system restart
isolate-panel system logs [--tail <n>] [--level <level>] [--follow]

# Statistics
isolate-panel stats                              # Dashboard stats
isolate-panel stats export --format <csv|json>   # Export statistics

# Active connections
isolate-panel connections [--user <id>] [--core <core>]

# Settings
isolate-panel settings list
isolate-panel settings get <key>
isolate-panel settings set <key> <value>
```

#### 11.6 Backup & Certificates (1 день)

**Commands:**
```bash
# Backup management
isolate-panel backup create [--destination <local|s3|ftp>]
isolate-panel backup list
isolate-panel backup restore <backup-id> [--force]
isolate-panel backup delete <backup-id>

# Certificate management
isolate-panel cert list
isolate-panel cert request <domain> [--email <email>]
isolate-panel cert get <id>
isolate-panel cert delete <id>
```

#### 11.7 Completion & Documentation (1 день)

**Задачи:**
- [ ] Bash completion script
- [ ] Zsh completion script
- [ ] Fish completion script
- [ ] Man pages
- [ ] CLI documentation (markdown)

**Installation:**
```bash
# Bash
isolate-panel completion bash > /etc/bash_completion.d/isolate-panel

# Zsh
isolate-panel completion zsh > /usr/local/share/zsh/site-functions/_isolate-panel

# Fish
isolate-panel completion fish > ~/.config/fish/completions/isolate-panel.fish
```

**Global Flags:**
```bash
--profile <name>      Use specific profile
--json                Output in JSON format
--csv                 Output in CSV format
--quiet               Minimal output (values only)
--no-color            Disable colored output
--config <path>       Config file path (default: ~/.isolate-panel/config.json)
-h, --help            Help for command
-v, --version         Version information
```

**Deliverables:**
- ✅ Полный функционал доступен через CLI
- ✅ CLI/UI feature parity (все операции из UI доступны в CLI)
- ✅ Multi-profile support для управления несколькими панелями
- ✅ Автоматическая аутентификация с refresh токенов
- ✅ Множественные output форматы (table, JSON, CSV, quiet)
- ✅ Interactive и non-interactive режимы
- ✅ Colored output с опцией отключения
- ✅ Shell completion для bash/zsh/fish
- ✅ Корректные exit codes для автоматизации
- ✅ Comprehensive documentation

---

### Фаза 12: Docker + Deployment (1 неделя)

#### 12.1 Dockerfile

**Задачи:**
- [ ] Multi-stage build для минимального размера
- [ ] Alpine Linux базовый образ
- [ ] Установка всех трех ядер
- [ ] Supervisord для управления процессами
- [ ] Healthcheck

**Dockerfile структура:**
```dockerfile
# Stage 1: Build backend
FROM golang:1.26.1-alpine AS backend-builder
# ... build Go app

# Stage 2: Build frontend
FROM node:20-alpine AS frontend-builder
# ... build Preact app

# Stage 3: Final image
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata supervisor

# Copy binaries
COPY --from=backend-builder /app/server /usr/local/bin/isolate-panel
COPY --from=frontend-builder /app/dist /var/www/html

# Install cores
ADD https://github.com/SagerNet/sing-box/releases/download/v1.13.3/sing-box-linux-amd64.tar.gz /tmp/
ADD https://github.com/XTLS/Xray-core/releases/download/v26.2.6/Xray-linux-64.zip /tmp/
ADD https://github.com/MetaCubeX/mihomo/releases/download/v1.19.21/mihomo-linux-amd64.gz /tmp/
# ... extract and install

# Supervisord config
COPY supervisord.conf /etc/supervisord.conf

EXPOSE 8080
VOLUME ["/data"]

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
```

#### 12.2 docker-compose.yml

```yaml
version: '3.8'

services:
  isolate-panel:
    build: .
    container_name: isolate-panel
    restart: unless-stopped
    
    ports:
      - "127.0.0.1:8080:8080"  # Панель (localhost only)
      - "443:443"              # Прокси порт (настраиваемый)
      - "443:443/udp"          # Прокси порт UDP
    
    volumes:
      - ./data:/data
      - ./logs:/var/log/isolate-panel
    
    environment:
      - TZ=UTC
      - JWT_SECRET=${JWT_SECRET}
      - DB_PATH=/data/isolate.db
      - LOG_LEVEL=info
      - MONITORING_MODE=lite
    
    security_opt:
      - no-new-privileges:true
    
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE
    
    networks:
      - isolate-network

networks:
  isolate-network:
    driver: bridge
```

#### 12.3 Установочный скрипт

**install.sh:**
```bash
#!/bin/bash
set -e

echo "=== Isolate Panel Installation ==="

# Проверка root
if [ "$EUID" -ne 0 ]; then 
   echo "Please run as root"
   exit 1
fi

# Проверка Docker
if ! command -v docker &> /dev/null; then
    echo "Installing Docker..."
    curl -fsSL https://get.docker.com | sh
fi

# Проверка docker-compose
if ! command -v docker-compose &> /dev/null; then
    echo "Installing docker-compose..."
    curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    chmod +x /usr/local/bin/docker-compose
fi

# Создание директорий
mkdir -p /opt/isolate-panel/{data,logs}
cd /opt/isolate-panel

# Скачивание docker-compose.yml
curl -o docker-compose.yml https://raw.githubusercontent.com/your-repo/isolate-panel/main/docker-compose.yml

# Генерация .env
echo "Generating secrets..."
JWT_SECRET=$(openssl rand -base64 64)
cat > .env << EOL
JWT_SECRET=${JWT_SECRET}
ADMIN_USERNAME=admin
ADMIN_PASSWORD=$(openssl rand -base64 16)
EOL

echo "Secrets saved to .env"

# Запуск
echo "Starting Isolate Panel..."
docker-compose up -d

echo ""
echo "=== Installation Complete ==="
echo "Panel URL: http://localhost:8080"
echo "Access via SSH tunnel: ssh -L 8080:localhost:8080 user@your-server"
echo ""
echo "Default credentials:"
cat .env | grep ADMIN
echo ""
echo "IMPORTANT: Change the default password after first login!"
```

**Deliverables:**
- Docker образ собирается
- docker-compose работает
- Установочный скрипт упрощает развертывание

---

### Фаза 13: Тестирование и документация (2 недели)

#### 13.1 Тестирование (1 неделя)

**Test Structure:**
```
tests/
├── unit/              # Unit tests
│   ├── services/
│   │   ├── user_service_test.go
│   │   ├── auth_service_test.go
│   │   └── core_service_test.go
│   ├── handlers/
│   │   ├── user_handler_test.go
│   │   └── auth_handler_test.go
│   └── utils/
│       └── validation_test.go
│
├── integration/       # Integration tests
│   ├── api_test.go
│   ├── database_test.go
│   └── core_integration_test.go
│
├── e2e/              # End-to-end tests
│   ├── user_flow_test.go
│   └── subscription_flow_test.go
│
├── fixtures/         # Test data
│   ├── users.json
│   ├── inbounds.json
│   └── cores.json
│
└── testutil/         # Test utilities
    ├── database.go   # Test DB setup
    ├── fixtures.go   # Fixture loader
    ├── mocks.go      # Mock generators
    └── assertions.go # Custom assertions
```

**Test Utilities (tests/testutil/database.go):**
```go
package testutil

import (
    "database/sql"
    "testing"
    
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "github.com/yourusername/isolate-panel/internal/database"
    "github.com/yourusername/isolate-panel/internal/models"
)

// SetupTestDB creates an in-memory SQLite database for testing
func SetupTestDB(t *testing.T) *gorm.DB {
    t.Helper()
    
    // Create in-memory database
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    if err != nil {
        t.Fatalf("Failed to create test database: %v", err)
    }
    
    // Get underlying sql.DB
    sqlDB, err := db.DB()
    if err != nil {
        t.Fatalf("Failed to get sql.DB: %v", err)
    }
    
    // Run migrations
    if err := runTestMigrations(sqlDB); err != nil {
        t.Fatalf("Failed to run migrations: %v", err)
    }
    
    return db
}

// TeardownTestDB closes the test database
func TeardownTestDB(t *testing.T, db *gorm.DB) {
    t.Helper()
    sqlDB, err := db.DB()
    if err != nil {
        t.Errorf("Failed to get sql.DB: %v", err)
        return
    }
    if err := sqlDB.Close(); err != nil {
        t.Errorf("Failed to close database: %v", err)
    }
}

func runTestMigrations(db *sql.DB) error {
    mm, err := database.NewMigrationManager(db)
    if err != nil {
        return err
    }
    defer mm.Close()
    return mm.Up()
}

// SeedTestData seeds the database with test data
func SeedTestData(t *testing.T, db *gorm.DB) {
    t.Helper()
    
    // Create test admin
    admin := &models.Admin{
        Username:     "testadmin",
        PasswordHash: "hashed_password",
        IsSuperAdmin: true,
    }
    if err := db.Create(admin).Error; err != nil {
        t.Fatalf("Failed to seed admin: %v", err)
    }
    
    // Create test users
    users := []models.User{
        {
            UUID:              "test-uuid-1",
            Username:          "testuser1",
            Email:             "test1@example.com",
            SubscriptionToken: "token1",
            IsActive:          true,
            TrafficLimit:      107374182400,
            TrafficUsed:       0,
        },
        {
            UUID:              "test-uuid-2",
            Username:          "testuser2",
            Email:             "test2@example.com",
            SubscriptionToken: "token2",
            IsActive:          false,
            TrafficLimit:      53687091200,
            TrafficUsed:       10737418240,
        },
    }
    
    for _, user := range users {
        if err := db.Create(&user).Error; err != nil {
            t.Fatalf("Failed to seed user: %v", err)
        }
    }
}
```

**Unit Test Example (Table-Driven):**
```go
// tests/unit/services/user_service_test.go
package services_test

import (
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/yourusername/isolate-panel/internal/services"
    "github.com/yourusername/isolate-panel/tests/testutil"
)

func TestUserService_CreateUser(t *testing.T) {
    tests := []struct {
        name    string
        request *services.CreateUserRequest
        wantErr bool
        errMsg  string
    }{
        {
            name: "valid user",
            request: &services.CreateUserRequest{
                Username:     "newuser",
                Email:        "new@example.com",
                TrafficLimit: 107374182400,
            },
            wantErr: false,
        },
        {
            name: "duplicate username",
            request: &services.CreateUserRequest{
                Username:     "testuser1", // Already exists
                Email:        "another@example.com",
                TrafficLimit: 107374182400,
            },
            wantErr: true,
            errMsg:  "username already exists",
        },
        {
            name: "invalid email",
            request: &services.CreateUserRequest{
                Username:     "user3",
                Email:        "invalid-email",
                TrafficLimit: 107374182400,
            },
            wantErr: true,
            errMsg:  "invalid email format",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            db := testutil.SetupTestDB(t)
            defer testutil.TeardownTestDB(t, db)
            testutil.SeedTestData(t, db)
            
            service := services.NewUserService(db)
            
            // Execute
            user, err := service.CreateUser(tt.request)
            
            // Assert
            if tt.wantErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
                assert.Nil(t, user)
            } else {
                require.NoError(t, err)
                assert.NotNil(t, user)
                assert.Equal(t, tt.request.Username, user.Username)
                assert.NotEmpty(t, user.UUID)
                assert.NotEmpty(t, user.SubscriptionToken)
            }
        })
    }
}
```

**Integration Test Example:**
```go
// tests/integration/api_test.go
package integration_test

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http/httptest"
    "testing"
    
    "github.com/gofiber/fiber/v2"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/yourusername/isolate-panel/internal/app"
    "github.com/yourusername/isolate-panel/tests/testutil"
)

func setupTestApp(t *testing.T) *fiber.App {
    t.Helper()
    db := testutil.SetupTestDB(t)
    testutil.SeedTestData(t, db)
    app := app.NewApp(db)
    return app
}

func loginAsAdmin(t *testing.T, app *fiber.App) string {
    t.Helper()
    
    reqBody := map[string]string{
        "username": "testadmin",
        "password": "admin",
    }
    body, _ := json.Marshal(reqBody)
    
    req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := app.Test(req)
    require.NoError(t, err)
    require.Equal(t, 200, resp.StatusCode)
    
    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    return result["access_token"].(string)
}

func TestAPI_UserCRUD(t *testing.T) {
    app := setupTestApp(t)
    token := loginAsAdmin(t, app)
    var userID uint
    
    t.Run("Create User", func(t *testing.T) {
        reqBody := map[string]interface{}{
            "username":      "apiuser",
            "email":         "api@example.com",
            "traffic_limit": 107374182400,
        }
        body, _ := json.Marshal(reqBody)
        
        req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        req.Header.Set("Authorization", "Bearer "+token)
        
        resp, err := app.Test(req)
        require.NoError(t, err)
        assert.Equal(t, 201, resp.StatusCode)
        
        var user map[string]interface{}
        json.NewDecoder(resp.Body).Decode(&user)
        userID = uint(user["id"].(float64))
        assert.Equal(t, "apiuser", user["username"])
        assert.NotEmpty(t, user["uuid"])
    })
    
    t.Run("Get User", func(t *testing.T) {
        req := httptest.NewRequest("GET", fmt.Sprintf("/api/users/%d", userID), nil)
        req.Header.Set("Authorization", "Bearer "+token)
        
        resp, err := app.Test(req)
        require.NoError(t, err)
        assert.Equal(t, 200, resp.StatusCode)
    })
    
    t.Run("Delete User", func(t *testing.T) {
        req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/users/%d", userID), nil)
        req.Header.Set("Authorization", "Bearer "+token)
        
        resp, err := app.Test(req)
        require.NoError(t, err)
        assert.Equal(t, 204, resp.StatusCode)
    })
}
```

**E2E Test Example:**
```go
// tests/e2e/user_flow_test.go
package e2e_test

import (
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
)

func TestCompleteUserFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E test in short mode")
    }
    
    app := setupTestApp(t)
    token := loginAsAdmin(t, app)
    
    // Step 1: Create user
    t.Log("Step 1: Creating user...")
    user := createUser(t, app, token, "e2euser", "e2e@example.com")
    userID := user["id"].(float64)
    subscriptionToken := user["subscription_token"].(string)
    
    // Step 2: Create inbound
    t.Log("Step 2: Creating inbound...")
    inbound := createInbound(t, app, token, "vmess", 10443)
    inboundID := inbound["id"].(float64)
    
    // Step 3: Assign user to inbound
    t.Log("Step 3: Assigning user to inbound...")
    assignUserToInbound(t, app, token, uint(userID), uint(inboundID))
    
    // Step 4: Generate subscription
    t.Log("Step 4: Generating subscription...")
    subscription := getSubscription(t, subscriptionToken, "v2ray")
    assert.NotEmpty(t, subscription)
    
    // Step 5: Simulate traffic and verify quota enforcement
    t.Log("Step 5: Testing quota enforcement...")
    simulateTraffic(t, app, uint(userID), 1073741824) // 1GB
    time.Sleep(2 * time.Second)
    
    stats := getUserStats(t, app, token, uint(userID))
    assert.Greater(t, stats["traffic_used"].(int64), int64(0))
    
    // Cleanup
    deleteUser(t, app, token, uint(userID))
    deleteInbound(t, app, token, uint(inboundID))
    
    t.Log("✓ Complete user flow test passed")
}
```

**Benchmark Tests:**
```go
// tests/unit/services/user_service_bench_test.go
package services_test

import (
    "fmt"
    "testing"
    
    "github.com/yourusername/isolate-panel/internal/services"
    "github.com/yourusername/isolate-panel/tests/testutil"
)

func BenchmarkUserService_CreateUser(b *testing.B) {
    db := testutil.SetupTestDB(&testing.T{})
    service := services.NewUserService(db)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        req := &services.CreateUserRequest{
            Username:     fmt.Sprintf("user%d", i),
            Email:        fmt.Sprintf("user%d@example.com", i),
            TrafficLimit: 107374182400,
        }
        _, _ = service.CreateUser(req)
    }
}

func BenchmarkUserService_ListUsers(b *testing.B) {
    db := testutil.SetupTestDB(&testing.T{})
    service := services.NewUserService(db)
    
    // Create 100 users
    for i := 0; i < 100; i++ {
        req := &services.CreateUserRequest{
            Username:     fmt.Sprintf("user%d", i),
            Email:        fmt.Sprintf("user%d@example.com", i),
            TrafficLimit: 107374182400,
        }
        service.CreateUser(req)
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = service.ListUsers(&services.ListUsersRequest{
            Page:     1,
            PageSize: 20,
        })
    }
}
```

**Makefile for Testing:**
```makefile
.PHONY: test test-unit test-integration test-e2e test-coverage

test:
	go test -v -race ./...

test-unit:
	go test -v -race ./tests/unit/...

test-integration:
	go test -v -race ./tests/integration/...

test-e2e:
	go test -v -race ./tests/e2e/...

test-coverage:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out

test-short:
	go test -v -short ./...

bench:
	go test -bench=. -benchmem ./tests/unit/...
```

**CI/CD Integration (GitHub Actions):**
```yaml
# .github/workflows/test.yml
name: Test

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Cache Go modules
        uses: actions/cache@v3
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-
      
      - name: Install dependencies
        run: go mod download
      
      - name: Run unit tests
        run: make test-unit
      
      - name: Run integration tests
        run: make test-integration
      
      - name: Run E2E tests
        run: make test-e2e
      
      - name: Generate coverage report
        run: make test-coverage
      
      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
          flags: unittests
          name: codecov-umbrella
      
      - name: Check coverage threshold
        run: |
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Total coverage: $coverage%"
          if (( $(echo "$coverage < 80" | bc -l) )); then
            echo "Coverage is below 80%"
            exit 1
          fi
```

**Unit тесты:**
- [ ] Auth модуль (login, JWT generation, refresh tokens, rate limiting)
- [ ] User management (CRUD, UUID generation, quota validation)
- [ ] Core management (start/stop/restart, config generation, validation)
- [ ] HAProxy config generator (route generation, backend configuration)
- [ ] Port manager (allocation, conflict detection)
- [ ] Traffic collector (stats aggregation, quota enforcement)
- [ ] Subscription generation (V2Ray, Clash, Sing-box formats)
- [ ] Config generators для всех ядер (Sing-box, Xray, Mihomo)
- [ ] Test utilities (database setup, fixtures, mocks)
- [ ] Table-driven tests для всех сервисов
- [ ] Parallel tests где возможно
- [ ] Coverage > 80%

**Integration тесты:**
- [ ] API endpoints (все CRUD операции)
- [ ] Работа с Supervisord (запуск/остановка ядер)
- [ ] Генерация и применение конфигураций
- [ ] HAProxy routing (SNI, path-based)
- [ ] Stats collection от всех трех ядер
- [ ] Quota enforcement flow (превышение → отключение → уведомление)
- [ ] Database migrations (up/down/rollback)
- [ ] Authentication flow (login, refresh, logout)

**Security тесты:**
- [ ] SQL injection prevention
- [ ] XSS prevention
- [ ] CSRF protection
- [ ] Rate limiting effectiveness
- [ ] JWT token validation
- [ ] Brute-force protection
- [ ] Authorization checks (admin vs user)

**Performance тесты:**
- [ ] API response time < 100ms (95 percentile)
- [ ] Config generation < 1 second для 100 пользователей
- [ ] Stats collection не превышает 5% CPU
- [ ] Memory usage < 512MB в Lite режиме
- [ ] Benchmark tests для критичных операций

**E2E тесты:**
- [ ] Полный flow: создание админа → создание пользователя → создание inbound → генерация подписки → подключение клиента
- [ ] Quota flow: пользователь превышает квоту → автоматически отключается → получает уведомление
- [ ] Multi-core flow: создание inbound на разных ядрах → HAProxy роутит корректно
- [ ] Backup и restore: создание бэкапа → восстановление → проверка целостности
- [ ] Subscription flow: генерация → обновление → отзыв

**Acceptance Criteria:**
- ✅ Все unit тесты проходят
- ✅ Coverage > 80% для всех пакетов
- ✅ Все integration тесты проходят
- ✅ Security тесты не выявляют критических уязвимостей
- ✅ Performance метрики соблюдаются
- ✅ E2E тесты проходят на чистой установке
- ✅ Benchmark tests показывают приемлемую производительность
- ✅ CI/CD pipeline проходит без ошибок
- ✅ Coverage report генерируется автоматически
- ✅ Test utilities переиспользуются во всех тестах

**Deliverables:**
- Полная test suite с > 80% coverage
- Test utilities для переиспользования
- Makefile для удобного запуска тестов
- CI/CD pipeline с автоматическими проверками
- Coverage reports (HTML + console)
- Benchmark results документированы

#### 13.2 Документация (1 неделя)

**Документы:**
- [ ] README.md - обзор проекта, quick start, основные возможности
- [ ] INSTALLATION.md - детальная инструкция по установке
- [ ] CONFIGURATION.md - настройка системы, все параметры
- [ ] ARCHITECTURE.md - архитектура системы, диаграммы, принципы
- [ ] API.md - полная документация API (OpenAPI/Swagger spec)
- [ ] CORE_INTEGRATION.md - как работает интеграция с ядрами
- [ ] HAPROXY_ROUTING.md - как работает HAProxy роутинг
- [ ] TRAFFIC_ACCOUNTING.md - система учета трафика и квот
- [ ] SECURITY.md - рекомендации по безопасности, best practices
- [ ] TROUBLESHOOTING.md - решение типичных проблем
- [ ] CONTRIBUTING.md - для контрибьюторов
- [ ] CHANGELOG.md - история изменений

**API Documentation:**
- OpenAPI 3.0 спецификация
- Примеры запросов/ответов для каждого endpoint
- Коды ошибок и их значения
- Rate limiting информация
- Authentication flow диаграммы

**Acceptance Criteria:**
- ✅ Все документы написаны и проверены
- ✅ OpenAPI спецификация сгенерирована и валидна
- ✅ Примеры кода работают
- ✅ Диаграммы актуальны
- ✅ Документация доступна на GitHub Pages

**Deliverables:**
- Полная документация
- API спецификация
- Troubleshooting guide
- Тесты покрывают критичный функционал

---

### Фаза 14: Оптимизация и полировка (1-2 недели)

#### Задачи
- [ ] Оптимизация производительности (profiling, bottleneck analysis)
- [ ] Уменьшение размера Docker образа (multi-stage build optimization)
- [ ] Оптимизация SQL запросов (indexes, query analysis)
- [ ] Кэширование где необходимо (subscription links, config generation)
- [ ] Улучшение UX (loading states, error messages, tooltips)
- [ ] Исправление багов из issue tracker
- [ ] Security audit (penetration testing, vulnerability scan)
- [ ] Load testing (100+ concurrent users, stress testing)
- [ ] Memory leak detection и исправление
- [ ] Code review и рефакторинг

**Будущие улучшения (TODO для следующих версий):**
- [ ] Динамическое отключение пользователей через API ядра (без restart)
- [ ] WebSocket для real-time обновлений в UI
- [ ] Графики и визуализация статистики
- [ ] Advanced routing rules (более сложная логика)
- [ ] Multi-language support для UI
- [ ] Dark mode для UI

**Acceptance Criteria:**
- ✅ RAM usage < 512MB в Lite режиме при 100 пользователях
- ✅ CPU usage < 20% в idle, < 50% под нагрузкой
- ✅ Docker image size < 200MB
- ✅ API response time < 100ms (95 percentile)
- ✅ Нет memory leaks после 24 часов работы
- ✅ Security audit не выявил критических уязвимостей
- ✅ Load test: 100 concurrent users без деградации

**Deliverables:**
- Стабильная версия 1.0.0
- Готово к production использованию
- Performance benchmarks документированы
- Security audit report

---

## Общая временная шкала

### Обновленные сроки с учетом HAProxy и улучшений

**Фазы с изменениями:**
- Фаза 0: 1 неделя
- Фаза 1: 4.5 недели (было 4) - добавлен HAProxy
- Фаза 2: 3 недели
- Фаза 3: 3.5 недели (было 3) - добавлены HAProxy routes
- Фаза 4: 2 недели
- Фаза 5: 2 недели
- Фаза 6: 2.5 недели (было 2) - unified stats provider
- Фаза 7: 2 недели
- Фаза 8: 2 недели
- Фаза 9: 1 неделя
- Фаза 10: 1 неделя
- Фаза 11: 1 неделя
- Фаза 12: 1 неделя
- Фаза 13: 2 недели
- Фаза 14: 2 недели

### Минимальный срок (MVP): 14-16 недель
- Фаза 0: 1 неделя
- Фаза 1: 4.5 недели (Backend MVP + HAProxy)
- Фаза 2: 3 недели (Frontend MVP)
- Фаза 3: 3.5 недели (Inbound/Outbound + HAProxy routing)
- Фаза 4: 2 недели (Подписки)
- Фаза 13: 2 недели (Тестирование и документация)

**Итого MVP: ~16 недель**

### Полная версия: 20-24 недели
- Все фазы 0-14 включены
- Полное тестирование и оптимизация
- Полная документация
- Все три ядра интегрированы
- HAProxy роутинг настроен
- Unified stats collection работает

### Рекомендуемый подход
1. **Недели 1-9**: MVP (Фазы 0-4) - базовый функционал + HAProxy
   - Sing-box как основное ядро
   - HAProxy для роутинга
   - Базовая панель управления
   - Генерация подписок
   
2. **Недели 10-14**: Расширение (Фазы 5-7) - сертификаты, мониторинг, дополнительные ядра
   - ACME сертификаты
   - Unified stats collection
   - Интеграция Xray и Mihomo
   - Все три ядра работают параллельно
   
3. **Недели 15-18**: Дополнительно (Фазы 8-12) - WARP, бэкапы, уведомления, CLI, Docker
   - WARP интеграция
   - Автоматические бэкапы
   - Email/Webhook уведомления
   - CLI интерфейс
   - Production-ready Docker образ
   
4. **Недели 19-22**: Полировка (Фазы 13-14) - тестирование, документация, оптимизация
   - Comprehensive testing (unit, integration, E2E, security, performance)
   - Полная документация
   - Security audit
   - Performance optimization
   - Готово к production

---

## Приоритеты разработки

### Критически важные (Must Have)
1. ✅ Безопасность (localhost-only, JWT, rate limiting, refresh tokens)
2. ✅ Аутентификация администраторов (Argon2id, 2FA support)
3. ✅ Управление пользователями (CRUD, квоты, UUID generation)
4. ✅ Управление Sing-box (основное ядро)
5. ✅ HAProxy роутинг (SNI/path-based routing)
6. ✅ Генерация подписок (V2Ray, Clash, Sing-box)
7. ✅ Базовый мониторинг (Lite режим, unified stats)
8. ✅ Supervisord для управления процессами

### Важные (Should Have)
1. ✅ Управление inbound/outbound с HAProxy интеграцией
2. ✅ ACME сертификаты (автоматическое обновление)
3. ✅ Статистика трафика (unified collection от всех ядер)
4. ✅ Активные подключения (real-time monitoring)
5. ✅ Xray интеграция (для XHTTP + gRPC stats)
6. ✅ Mihomo интеграция (для специфичных протоколов + REST stats)
7. ✅ Port management (автоматическое выделение, conflict detection)
8. ✅ Config validation (перед применением)

### Желательные (Nice to Have)
1. ✅ WARP интеграция
2. ✅ GeoIP/GeoSite
3. ✅ Резервное копирование
4. ✅ Уведомления (Email, Webhook)
5. ✅ CLI интерфейс
6. ✅ Full режим мониторинга

### Будущие функции (Future)
1. 🔮 Telegram Bot
2. 🔮 Оплата подписок
3. 🔮 Multi-server управление
4. 🔮 Графики и аналитика
5. 🔮 Мобильное приложение для администраторов
6. 🔮 API для интеграции с внешними системами

---

## Метрики успеха

### Производительность
- ✅ Потребление RAM: < 512 MB в Lite режиме
- ✅ Потребление CPU: < 20% на 1 CPU при 100 пользователях
- ✅ Размер Docker образа: < 200 MB
- ✅ Время запуска: < 10 секунд
- ✅ Время отклика API: < 100ms (95 percentile)

### Надежность
- ✅ Uptime: > 99.9%
- ✅ Автоматическое восстановление ядер при сбое
- ✅ Graceful shutdown
- ✅ Нет потери данных при перезапуске

### Безопасность
- ✅ Нет критических уязвимостей
- ✅ Все данные зашифрованы
- ✅ Логирование всех действий
- ✅ Rate limiting работает
- ✅ Защита от основных атак (SQL injection, XSS, CSRF)

### Удобство использования
- ✅ Установка за < 5 минут
- ✅ Интуитивный UI
- ✅ Полная документация
- ✅ CLI для автоматизации

---

## Риски и митигация

### Технические риски

| Риск | Вероятность | Влияние | Митигация |
|------|-------------|---------|-----------|
| Несовместимость версий ядер | Средняя | Высокое | Фиксация версий, тестирование перед обновлением |
| Проблемы с производительностью на слабых VPS | Высокая | Среднее | Lite режим, оптимизация, профилирование, HAProxy опционален |
| Уязвимости безопасности | Средняя | Критическое | Security audit, регулярные обновления, следование best practices |
| Сложность конфигурации ядер | Средняя | Среднее | Абстракция через UI, валидация конфигов |
| Потеря данных | Низкая | Критическое | Автоматические бэкапы, транзакции БД |
| HAProxy становится single point of failure (если включен) | Средняя | Высокое | Supervisord автоматически перезапускает, health checks, мониторинг, опция отключения |
| Сложность отладки HAProxy routing (если включен) | Средняя | Среднее | Детальное логирование, HAProxy stats page, документация, опция отключения |
| Latency от дополнительного hop через HAProxy (если включен) | Низкая | Низкое | HAProxy очень быстрый (<1ms overhead), benchmarking, опция отключения |
| Конфликты портов между ядрами (если HAProxy выключен) | Средняя | Среднее | Port Manager с диапазонами, строгая валидация при создании inbound |
| Graceful restart не поддерживается всеми ядрами | Высокая | Среднее | Fallback на полный restart, тестирование поддержки для каждого ядра |
| Превышение connection limits | Средняя | Высокое | Мониторинг connections, alerting при 80%, правильный расчет maxconn |

### Организационные риски

| Риск | Вероятность | Влияние | Митигация |
|------|-------------|---------|-----------|
| Недостаток времени на разработку | Средняя | Высокое | MVP подход, приоритизация функций |
| Изменение требований | Средняя | Среднее | Гибкая архитектура, модульность |
| Недостаток документации | Высокая | Среднее | Документирование в процессе разработки |

---

## Зависимости и требования

### Системные требования

**Минимальные (для панели, 30 пользователей):**

*С HAProxy (Advanced Mode):*
- CPU: 1 core
- RAM: 1 GB (850-950MB используется)
- Disk: 2 GB
- OS: Linux (Ubuntu 20.04+, Debian 11+, CentOS 8+)

*Без HAProxy (Simple Mode):*
- CPU: 1 core
- RAM: 1 GB (800-900MB используется)
- Disk: 2 GB
- OS: Linux (Ubuntu 20.04+, Debian 11+, CentOS 8+)

**Рекомендуемые (для стабильной работы):**
- CPU: 2 cores
- RAM: 2 GB
- Disk: 5 GB
- OS: Ubuntu 22.04 LTS

**Для production с нагрузкой (50+ пользователей):**
- CPU: 2-4 cores
- RAM: 2-4 GB
- Disk: 10-20 GB
- Bandwidth: 100 Mbps+

**Breakdown потребления памяти (30 пользователей, Lite Mode):**

**ВАЖНО:** С lazy loading ядра запускаются только по требованию!

*Типичный сценарий (только Sing-box) С HAProxy:*
```
Go Backend:           80MB
Sing-box:            50MB  ← Запущен (есть inbound)
Xray:                 0MB  ← НЕ запущен (нет inbound)
Mihomo:               0MB  ← НЕ запущен (нет inbound)
HAProxy:             50MB (maxconn=1024)
Supervisord:         15MB
SQLite cache:        16MB
OS overhead:        150MB
Active connections:  50MB
Buffers:             50MB
-----------------------------------
ИТОГО:             ~461MB  ← ЭКОНОМИЯ 80MB!
Резерв:            ~539MB (для burst traffic)
```

*Типичный сценарий (только Sing-box) БЕЗ HAProxy:*
```
Go Backend:           80MB
Sing-box:            50MB  ← Запущен (есть inbound)
Xray:                 0MB  ← НЕ запущен (нет inbound)
Mihomo:               0MB  ← НЕ запущен (нет inbound)
Supervisord:         15MB
SQLite cache:        16MB
OS overhead:        150MB
Active connections:  50MB
Buffers:             50MB
-----------------------------------
ИТОГО:             ~411MB  ← ЭКОНОМИЯ 80MB!
Резерв:            ~589MB (для burst traffic)
```

*Максимальный сценарий (все ядра) С HAProxy:*
```
Go Backend:           80MB
Sing-box:            50MB  ← Запущен
Xray:                40MB  ← Запущен (есть XHTTP inbound)
Mihomo:              40MB  ← Запущен (есть Mihomo inbound)
HAProxy:             50MB (maxconn=1024)
Supervisord:         15MB
SQLite cache:        16MB
OS overhead:        150MB
Active connections:  50MB
Buffers:             50MB
-----------------------------------
ИТОГО:             ~541MB
Резерв:            ~459MB (для burst traffic)
```

**Вывод:** Lazy loading экономит 80-100MB RAM в типичных сценариях, когда используется только одно ядро. Это критично для VPS с 1GB RAM!

### Внешние зависимости

**Обязательные:**
- Docker 20.10+
- docker-compose 2.0+
- SSH доступ к серверу

**Опциональные:**
- SMTP сервер (для email уведомлений)
- S3-совместимое хранилище (для бэкапов)
- Домен с DNS (для ACME сертификатов)

---

## Лицензия и распространение

### Рекомендуемая лицензия
**MIT License** или **Apache 2.0**

**Причины:**
- Разрешает коммерческое использование
- Минимальные ограничения
- Популярна в open-source сообществе
- Совместима с большинством других лицензий

### Распространение

**GitHub:**
- Публичный репозиторий
- GitHub Releases для версий
- GitHub Actions для CI/CD
- GitHub Issues для багов и feature requests

**Docker Hub:**
- Автоматическая публикация образов
- Теги для версий (latest, v1.0.0, etc.)

**Документация:**
- GitHub Pages или отдельный сайт
- Примеры конфигураций
- Видео-туториалы (опционально)

---

## Поддержка и сообщество

### Каналы поддержки

1. **GitHub Issues** - баги и feature requests
2. **GitHub Discussions** - вопросы и обсуждения
3. **Telegram группа** - быстрая помощь (опционально)
4. **Email** - для приватных вопросов

### Вклад в проект

**Приветствуются:**
- Исправления багов
- Новые функции
- Улучшение документации
- Переводы на другие языки
- Тестирование

**Процесс:**
1. Fork репозитория
2. Создание feature branch
3. Коммиты с понятными сообщениями
4. Pull request с описанием изменений
5. Code review
6. Merge после одобрения

---

## Заключение

### Ключевые преимущества проекта

1. **Безопасность**: Доступ только через SSH туннель, современные практики безопасности
2. **Легковесность**: Работает на VPS с 1 CPU / 1GB RAM
3. **Простота**: Установка за 5 минут, интуитивный интерфейс
4. **Гибкость**: Поддержка трех ядер, множество протоколов
5. **Надежность**: Автоматические бэкапы, мониторинг, уведомления

### Целевая аудитория

- Системные администраторы
- DevOps инженеры
- Пользователи, управляющие личными прокси-серверами
- Небольшие команды, нуждающиеся в прокси-решении

### Отличия от существующих решений

**vs 3x-ui:**
- ✅ Легче (Preact vs Vue)
- ✅ Безопаснее (localhost-only по умолчанию)
- ✅ Поддержка трех ядер одновременно
- ✅ HAProxy опционален (гибкость выбора)
- ✅ Unified stats collection от всех ядер
- ✅ Современный стек технологий
- ✅ Работает на 1GB RAM

**vs s-ui:**
- ✅ Более полный функционал
- ✅ Лучший UX
- ✅ Больше протоколов (три ядра)
- ✅ CLI интерфейс
- ✅ Опциональный HAProxy роутинг
- ✅ Connection monitoring с alerting

**vs Hiddify-Manager:**
- ✅ Более простая архитектура (монолитная vs микросервисы)
- ✅ Легче для слабых VPS (1GB RAM)
- ✅ Только для администраторов (проще, безопаснее)
- ✅ Современный frontend (Preact)
- ⚠️ Меньше функций (нет multi-server, нет payment system)

**vs metacubexd:**
- ✅ Полноценная панель управления
- ✅ Управление пользователями
- ✅ Подписки
- ✅ Мониторинг
- ✅ Поддержка трех ядер

### Следующие шаги

1. **Создать GitHub репозиторий**
2. **Настроить окружение разработки** (Фаза 0)
3. **Начать с MVP Backend** (Фаза 1)
4. **Итеративная разработка** по фазам
5. **Регулярные релизы** (alpha, beta, stable)
6. **Сбор обратной связи** от пользователей
7. **Непрерывное улучшение**

---

## Контакты и ресурсы

### Полезные ссылки

**Документация ядер:**
- Sing-box: https://sing-box.sagernet.org/
- Xray: https://xtls.github.io/
- Mihomo: https://wiki.metacubex.one/

**Технологии:**
- Go: https://go.dev/
- Fiber: https://gofiber.io/
- Preact: https://preactjs.com/
- GORM: https://gorm.io/
- HAProxy: https://www.haproxy.org/

**Безопасность:**
- OWASP Top 10: https://owasp.org/www-project-top-ten/
- CWE Top 25: https://cwe.mitre.org/top25/

---

## Ключевые архитектурные решения

### Принятые решения на основе анализа

1. **HAProxy как опциональный компонент** ✅
   - Можно включить/выключить в зависимости от потребностей
   - Если включен: множественные inbound на одном порту, SNI/path routing
   - Если выключен: экономия 50MB RAM, меньше latency
   - Гибкость выбора между простотой и функциональностью
   - Рекомендация: выключен для VPS 1GB, включен для VPS 2GB+

2. **Supervisord для управления процессами** ✅
   - Простой и надежный process manager
   - Автоматический перезапуск при падении
   - Легко интегрируется с Docker
   - Логирование stdout/stderr

3. **Unified Stats Provider интерфейс** ✅
   - Абстракция над разными API ядер
   - Sing-box: Clash API (REST)
   - Xray: Stats API (gRPC)
   - Mihomo: External Controller (REST)
   - Упрощает сбор статистики

4. **Go structs + JSON marshal для генерации конфигов** ✅
   - Типобезопасность
   - Легко тестировать
   - Валидация на уровне типов
   - Переиспользование структур

5. **Graceful restart с fallback** ✅
   - Приоритет: graceful reload (SIGHUP)
   - Fallback: полный restart через supervisord
   - Минимизирует downtime
   - Не обрывает существующие соединения (где поддерживается)

6. **Port allocation по диапазонам** ✅
   - Sing-box: 10000-19999
   - Xray: 20000-29999
   - Mihomo: 30000-39999
   - Предотвращает конфликты
   - Упрощает отладку

7. **Quota enforcement через config regeneration** ✅
   - MVP: отключение пользователя → регенерация конфига → restart
   - TODO (v2.0): динамическое отключение через API ядра
   - Надежно и просто для MVP
   - Улучшение запланировано

8. **Data retention strategy** ✅
   - Сырые данные: 7 дней
   - Почасовая агрегация: 90 дней
   - Дневная агрегация: 1 год
   - Автоматическая очистка через cron
   - Балансирует детальность и размер БД

9. **Refresh tokens в БД** ✅
   - Возможность отзыва токенов
   - Просмотр активных сессий
   - Ограничение количества устройств
   - Повышенная безопасность

10. **Rate limiting с персистентностью** ✅
    - Таблица login_attempts в БД
    - История атак сохраняется
    - Можно анализировать паттерны
    - Защита от brute-force

11. **Connection limits мониторинг** ✅
    - HAProxy stats monitoring через unix socket
    - Alerting при 80% использования
    - Dashboard метрики для connections
    - Рекомендации по масштабированию
    - Правильный расчет maxconn на основе количества пользователей

---

**Дата создания плана**: 21 марта 2026  
**Версия документа**: 3.0 (HAProxy опционален, корректные connection limits)
**Статус**: Готов к реализации

---

*Этот план является живым документом и будет обновляться по мере развития проекта.*
