# 📋 Итоговые решения по критике плана

**Дата:** 23 марта 2026  
**Версия:** 1.0  
**Статус:** Утверждено и реализовано

---

## 📊 Статистика

**Всего пунктов критики:** 50  
**Согласен:** 38  
**Обсуждено и решено:** 12  
**Новых критических проблем:** 1 (subscription security)

---

## ✅ КЛЮЧЕВЫЕ РЕШЕНИЯ

### 1. HAProxy → Post-MVP v1.5
**Решение:** Полностью исключен из MVP  
**Причина:** Добавляет сложность, 40-50MB RAM, latency overhead  
**MVP подход:** Прямое подключение к ядрам, каждый inbound на уникальном порту

### 2. SSH/WireGuard ключи → Post-MVP v1.3
**Решение:** Исключены из MVP  
**Причина:** Не требуются для базового функционала  
**MVP подход:** Только UUID, Password, Token для прокси-протоколов

### 3. Plaintext credentials в MVP
**Решение:** Credentials хранятся в plaintext  
**Причина:** Упрощение разработки MVP  
**Post-MVP:** Шифрование AES-256-GCM в v1.1  
**Безопасность:** Панель доступна только через SSH tunnel

### 4. Argon2id для паролей админов
**Решение:** Использовать Argon2id вместо bcrypt  
**Обоснование:**
- Современный стандарт (PHC 2015 winner)
- Лучшая защита от GPU/ASIC атак
- Memory-hard алгоритм
- Рекомендован OWASP 2023

**Конфигурация:**
- Time: 1 iteration
- Memory: 64 MB
- Threads: 4
- KeyLength: 32 bytes
- SaltLength: 16 bytes (уникальный для каждого пользователя)

### 5. Lazy loading ядер
**Решение:** Реализовать lazy loading  
**Подход:** `autostart=false` в supervisord, запуск по требованию  
**Преимущества:** Экономия RAM, запуск только нужных ядер

### 6. Пользователь выбирает порты вручную
**Решение:** Убрать автоматическую формулу `Port: 10000 + inbound.ID`  
**Подход:** Пользователь указывает порт при создании inbound  
**Валидация:**
- Диапазон: 1024-65535
- Проверка занятости порта
- Проверка уникальности
- Блокировка системных портов (8080, 22, 80, 443)

### 7. Версии ядер КОРРЕКТНЫ
**Решение:** Версии в плане актуальны, критика ошибочна  
**Проверено на GitHub (23 марта 2026):**
- Xray-core: v26.2.6 (Latest stable, 6 Feb 2026) ✓
- Sing-box: v1.13.3 (Latest stable, 15 Mar 2026) ✓
- Mihomo: v1.19.21 (Latest stable, 9 Mar 2026) ✓

### 8. UUID v4 генерация
**Решение:** Генерируем UUID v4 автоматически при создании пользователя  
**MVP:** Простая валидация без проверки версии  
**Post-MVP:** Добавить warning для non-v4 UUID

### 9. Timezone handling
**Решение:** Стандартный подход  
**Хранение:** Всегда UTC в БД  
**Отображение:** Локальное время браузера (автоматически)  
**API:** ISO 8601 с timezone info

### 10. CORS не требуется
**Решение:** CORS не настраивать в MVP  
**Причина:** Frontend и backend на одном origin (localhost:8080)

---

## 🔒 КРИТИЧЕСКАЯ НАХОДКА: SUBSCRIPTION SECURITY

### Проблема
Subscription endpoints (`/sub/*`) публично доступны из интернета, что создает серьезные риски безопасности.

### Что было в плане
- Rate limiting: 10 req/hour на token
- Access logging
- Token validation

### Что ОТСУТСТВОВАЛО (критично)
1. IP-based rate limiting
2. Failed request tracking
3. User-Agent validation
4. DDoS protection
5. Bot detection

### Решение: Многоуровневая защита

#### Уровень 1: Request Validation
- Проверка блокировки IP
- Требование User-Agent
- Блокировка подозрительных UA (curl, wget, python, scanner, bot)

#### Уровень 2: Multi-level Rate Limiting
- **IP-based:** 30 req/hour per IP
- **Token-based:** 10 req/hour per token (было)
- **Global:** 1000 req/hour total

#### Уровень 3: Failed Request Tracking
- Отслеживание неудачных попыток
- Автоблокировка IP после 20 неудачных попыток
- Блокировка на 24 часа

#### Уровень 4: Enhanced Access Logging
- IP, User-Agent, Country, Format
- Response time, Suspicious flag
- Для анализа атак и аномалий

#### Уровень 5: Anomaly Detection (желательно)
- Фоновый мониторинг подозрительной активности
- Детекция множественных IP/стран
- Алерты админу

### Изменения в БД

**Новая таблица:**
```sql
CREATE TABLE blocked_ips (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ip_address VARCHAR(45) NOT NULL,
    reason VARCHAR(255),
    blocked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL,
    UNIQUE(ip_address)
);
```

**Расширенная таблица:**
```sql
CREATE TABLE subscription_accesses (
    -- Добавлены поля:
    country VARCHAR(2),
    format VARCHAR(20),
    is_suspicious BOOLEAN DEFAULT FALSE,
    response_time_ms INTEGER DEFAULT 0
);
```

### Middleware Stack
```go
app.Get("/sub/:token", 
    subscriptionValidator.Validate(),      // User-Agent, IP blocking
    subscriptionRateLimiter.Middleware(),  // Multi-level rate limiting
    subscriptionHandler.GetSubscription,   // Main handler
)
```

### Документация
Создан [SECURITY_PLAN.md](./SECURITY_PLAN.md) с полной спецификацией защиты.

---

## 📝 ОБНОВЛЕННЫЕ ДОКУМЕНТЫ

### 1. SECURITY_PLAN.md (НОВЫЙ)
- Полная спецификация безопасности
- Защита панели управления
- Защита subscription endpoints
- Argon2id хеширование
- Защита от атак (SQL injection, XSS, CSRF)
- Мониторинг и алерты

### 2. PROJECT_PLAN.md
- Обновлена спецификация подписок (multi-level rate limiting)
- Добавлена таблица `blocked_ips`
- Расширена таблица `subscription_accesses`
- Ссылка на SECURITY_PLAN.md

### 3. CHANGES_SUMMARY.md
- Добавлена информация о subscription security
- Обновлен MVP scope (security features)
- Обновлен Post-MVP roadmap

### 4. PROTOCOL_SMART_FORMS_PLAN.md
- Убраны SSH/WG ключи из MVP
- Обновлен security раздел

---

## 🎯 MVP SCOPE (финальный)

### ✅ Включено в MVP

**Ядра:**
- Sing-box v1.13.3
- Xray v26.2.6
- Mihomo v1.19.21

**Протоколы:**
- HTTP, SOCKS5, Mixed
- Shadowsocks, VMess, VLESS, Trojan
- Hysteria2, TUIC v4/v5, Naive
- Redirect, XHTTP
- Mieru, Sudoku, TrustTunnel, ShadowsocksR, Snell

**Функциональность:**
- User Management (CRUD)
- Inbound Management (CRUD)
- User-Inbound mapping
- Базовая статистика трафика
- Подписки (V2Ray, Clash, Sing-box)
- Web UI (Preact)
- CLI интерфейс
- JWT auth
- SQLite + golang-migrate
- WARP интеграция
- GeoIP/GeoSite
- ACME сертификаты

**Security:**
- Argon2id хеширование паролей админов
- Multi-level rate limiting (IP + Token + Global)
- Failed request tracking и IP blocking
- User-Agent validation
- Enhanced access logging
- CSRF protection
- Security headers (CSP, X-Frame-Options, etc.)
- SQLite WAL mode

**Credentials (plaintext):**
- UUID v4 (для VMess/VLESS/TUIC v5)
- Password (для Trojan/Shadowsocks)
- Token (для TUIC v4)
- Subscription Token (64 символа)

### ❌ Исключено из MVP

**Протоколы:**
- TProxy → Post-MVP v1.3
- TUN → Post-MVP v1.3
- WireGuard → Post-MVP v1.3
- SSH (как прокси-протокол) → Post-MVP v1.3

**Инфраструктура:**
- HAProxy → Post-MVP v1.5

**Credentials:**
- SSH ключи → Post-MVP v1.3
- WireGuard ключи → Post-MVP v1.3

**Функциональность:**
- Email/Webhook уведомления → Post-MVP v1.2
- Автоматическое резервное копирование → Post-MVP v1.2
- Multi-admin с ролями → Post-MVP v1.4
- User portal → Post-MVP v1.4
- QR code генерация → Post-MVP v1.4

**Security:**
- Шифрование credentials (AES-256-GCM) → Post-MVP v1.1
- CAPTCHA → Post-MVP v1.1
- Geographic restrictions → Post-MVP v1.1
- Advanced anomaly detection → Post-MVP v1.1
- Automatic token rotation → Post-MVP v1.1
- 2FA для админов → Post-MVP v1.1

---

## 🔄 Post-MVP Roadmap

### v1.1 - Security Improvements (2-3 недели)
- Шифрование credentials (AES-256-GCM)
- Показ credentials только при создании/регенерации
- Encryption key rotation
- CAPTCHA для subscription endpoints
- Geographic restrictions
- Advanced anomaly detection с ML

### v1.2 - Notifications & Backup (2 недели)
- Email уведомления
- Webhook уведомления
- Автоматическое резервное копирование (S3, FTP, SFTP)
- Шифрование бэкапов

### v1.3 - Advanced Protocols (3 недели)
- TUN support
- TProxy support
- WireGuard support
- SSH как прокси-протокол
- SSH/WireGuard ключи для пользователей

### v1.4 - User Experience (2 недели)
- User portal (для просмотра статистики)
- QR code генерация
- Multi-admin с ролями и правами

### v1.5 - HAProxy Integration (2 недели)
- HAProxy для продвинутого роутинга
- SNI-based routing
- Path-based routing
- Централизованный rate limiting

---

## ⏱️ Оценки времени

### MVP (обновленные)
- Фаза 0: 1 неделя (Setup)
- Фаза 1: 5-6 недель (Backend MVP)
- Фаза 2: 3 недели (Frontend MVP)
- Фаза 3: 3 недели (Inbound/Outbound)
- Фаза 4: 2 недели (Подписки + Security)
- Фаза 13: 2 недели (Тестирование)

**Итого MVP:** ~16-17 недель (4 месяца)

### Полная версия
**Итого с Post-MVP:** ~25-30 недель (6-7 месяцев)

---

## ✅ Чеклист реализации

### Критические изменения (Приоритет 1)
- [x] Создан SECURITY_PLAN.md
- [x] Обновлена спецификация подписок
- [x] Добавлена таблица blocked_ips
- [x] Расширена таблица subscription_accesses
- [x] Обновлен CHANGES_SUMMARY.md
- [ ] Реализовать Argon2id хеширование
- [ ] Реализовать multi-level rate limiting
- [ ] Реализовать failed request tracking
- [ ] Реализовать User-Agent validation
- [ ] Настроить SQLite WAL mode

### Высокий приоритет (Приоритет 2)
- [ ] Реализовать lazy loading ядер
- [ ] Реализовать port validation
- [ ] Протестировать graceful reload
- [ ] Добавить security headers
- [ ] Реализовать CSRF protection
- [ ] Настроить JWT с проверкой длины секрета

### Средний приоритет (Приоритет 3)
- [ ] Исправить broken links
- [ ] Стандартизировать naming conventions
- [ ] Добавить WARP спецификацию
- [ ] Добавить GeoIP/GeoSite спецификацию
- [ ] Настроить log rotation
- [ ] Добавить Docker healthcheck

---

## 📚 Связанные документы

1. [PROJECT_PLAN.md](./PROJECT_PLAN.md) - Основной план проекта
2. [SECURITY_PLAN.md](./SECURITY_PLAN.md) - План безопасности
3. [CHANGES_SUMMARY.md](./CHANGES_SUMMARY.md) - Итоговый список изменений
4. [PROTOCOL_SMART_FORMS_PLAN.md](./PROTOCOL_SMART_FORMS_PLAN.md) - Protocol-Aware Smart Forms
5. [plan_critik.md](./plan_critik.md) - Критический анализ документации

---

## 🎉 ЗАКЛЮЧЕНИЕ

Все критические проблемы из критики плана проанализированы и решены. Документация обновлена и готова к реализации MVP.

**Ключевые достижения:**
1. ✅ Определен четкий MVP scope
2. ✅ Исключены сложные компоненты (HAProxy, SSH/WG ключи)
3. ✅ Выбран современный алгоритм хеширования (Argon2id)
4. ✅ Разработана многоуровневая защита subscription endpoints
5. ✅ Создан полный план безопасности
6. ✅ Обновлены все документы для консистентности

**Готово к началу разработки MVP.**

---

**Дата утверждения:** 23 марта 2026  
**Утверждено:** Пользователь + AI Assistant
