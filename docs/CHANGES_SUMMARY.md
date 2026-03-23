# 📋 Итоговый список изменений документации

**Дата:** 2026-03-23  
**Версия:** 1.0  
**Статус:** Все критичные изменения применены

---

## ✅ Критичные изменения (выполнено)

### 1. Стратегия миграций БД
- ❌ Удалено: `GORM AutoMigrate` как основной способ миграций
- ✅ Добавлено: `golang-migrate` как единственный способ для production
- ✅ Примечание: AutoMigrate только для dev/testing

### 2. Модель доступа к панели
- ✅ Подтверждено: Панель ВСЕГДА на `127.0.0.1:8080`
- ✅ Уточнено: HAProxy маршрутизирует ТОЛЬКО прокси-трафик (исключен из MVP)
- ✅ Обновлена архитектурная диаграмма: прямое подключение к ядрам

### 3. Логика выбора ядра
- ❌ Удалено: "КРИТИЧЕСКИ ВАЖНО: Sing-box - основное ядро"
- ❌ Удалено: Автоматическая логика выбора ядра
- ✅ Добавлено: User-driven выбор (пользователь выбирает ядро первым)
- ✅ Обновлена таблица протоколов: убрана колонка "Примечание" с "Sing-box - основной"
- ✅ Добавлено: Рекомендации в UI, но без ограничений

### 4. Схема таблицы users
- ✅ Обновлена на расширенную версию:
  - `uuid VARCHAR(36) UNIQUE NOT NULL`
  - `password VARCHAR(255) NOT NULL` (plaintext для MVP)
  - `token VARCHAR(64) UNIQUE`
  - `subscription_token VARCHAR(64) UNIQUE NOT NULL`
- ❌ Удалено из MVP: `ssh_public_key`, `ssh_private_key_encrypted`
- ⏸️ Отложено на Post-MVP: `wireguard_private_key`, `wireguard_public_key`
- ✅ Добавлен комментарий: "MVP: plaintext credentials. Post-MVP: шифрование"

### 5. Схема таблицы admins
- ✅ Добавлено поле: `is_super_admin BOOLEAN DEFAULT FALSE`

### 6. Схема таблицы notifications
- ✅ Добавлено поле: `priority VARCHAR(20) DEFAULT 'normal'`
- ✅ Добавлен индекс: `idx_notifications_priority`

### 7. Недостающие таблицы
- ✅ Добавлена схема: `traffic_stats_hourly`
- ✅ Добавлена схема: `subscription_short_urls`
- ✅ Добавлена схема: `subscription_accesses` (расширенная с полями для безопасности)
- ✅ Добавлена схема: `blocked_ips` (для защиты от атак)

### 8. Таблица haproxy_routes
- ⏸️ Помечена как Post-MVP (закомментирована в схеме)

### 9. SQL синтаксис для SQLite
- ✅ Исправлено: `DATE_TRUNC('hour', recorded_at)` → `strftime('%Y-%m-%d %H:00:00', recorded_at)`
- ✅ Исправлено: `NOW() - INTERVAL '7 days'` → `datetime('now', '-7 days')`
- ✅ Исправлено: `sqlDB.Exec(...).Error` → `_, err := sqlDB.Exec(...)`

### 10. Security guidance для Alpine
- ✅ Обновлены команды: `apt-get` → `apk add`
- ✅ Обновлены команды: `systemctl` → `rc-service`
- ✅ Добавлено примечание для Ubuntu/Debian

### 11. Docker hardening
- ✅ Добавлен volume для логов: `./logs:/var/log/isolate-panel:rw`
- ✅ Обновлены capabilities: только `NET_BIND_SERVICE` (без TUN/TProxy/WireGuard)

### 12. HAProxy исключен из MVP
- ❌ Удален раздел "HAProxy: Опциональный режим"
- ✅ Перенесен в раздел "Post-MVP Features"
- ❌ Удалены HAProxy endpoints из API
- ❌ Удален HAProxy из supervisord конфига (закомментирован)
- ✅ Обновлена архитектурная диаграмма

### 13. Протоколы исключены из MVP
- ⏸️ TProxy → Post-MVP
- ⏸️ TUN → Post-MVP
- ⏸️ WireGuard → Post-MVP
- ⏸️ SSH как прокси-протокол → Post-MVP (SSH только для VPS доступа)

### 14. Спецификация подписок
- ✅ Добавлены API endpoints:
  - `GET /sub/:token` (V2Ray base64)
  - `GET /sub/:token/clash` (Clash YAML)
  - `GET /sub/:token/singbox` (Sing-box JSON)
  - `GET /s/:short_code` (Short URL redirect)
- ✅ Добавлена спецификация: auth, rate limiting, форматы
- ✅ Добавлена многоуровневая защита (IP + Token + Global rate limiting)
- ✅ Добавлена защита от DDoS и brute force атак
- ✅ См. [SECURITY_PLAN.md](./SECURITY_PLAN.md) для деталей

### 15. Фазы разработки
- ✅ Обновлена Фаза 1.3: User Management (без SSH/WG ключей)
- ✅ Обновлена Фаза 1.4: Управление ядрами (без HAProxy)
- ✅ Увеличены оценки сроков: 1.5 недели → 2 недели

---

## ✅ PROTOCOL_SMART_FORMS_PLAN.md изменения

### 1. Статус документа
- ✅ Установлен единый статус: "Спецификация завершена, реализация не начата"
- ✅ Обновлена версия: 1.1
- ✅ Обновлена дата: 2026-03-23

### 2. Модель хранения credentials
- ✅ Обновлено на MVP модель (plaintext credentials)
- ✅ Добавлен раздел "Post-MVP Security Improvements"

### 3. SSH credentials
- ❌ Удалена генерация SSH ключей для пользователей
- ✅ Добавлено примечание: "SSH только для доступа к VPS"

### 4. Валидационный пример
- ✅ Исправлено: `validateUUID(uuid string)` → `validateUUID(id string)`

### 5. Ссылки
- ❌ Удалены ссылки на несуществующие файлы (DEVELOPMENT_ROADMAP.md, KRITIK.md)

### 6. Roadmap
- ✅ Обновлен Phase 1: добавлена пометка "⏸️ SSH/WireGuard credentials (Post-MVP)"

---

## 📊 MVP Scope (финальный)

### ✅ Включено в MVP:

**Ядра:**
- Sing-box v1.13.3
- Xray v26.2.6
- Mihomo v1.19.21

**Протоколы (Inbound):**
- HTTP, SOCKS5, Mixed
- Shadowsocks, VMess, VLESS, Trojan
- Hysteria2, TUIC v4/v5, Naive
- Redirect
- XHTTP (Xray эксклюзив)
- Mieru, Sudoku, TrustTunnel, ShadowsocksR, Snell (Mihomo эксклюзивы)

**Функциональность:**
- User Management (CRUD)
- Inbound Management (CRUD)
- User-Inbound mapping
- Базовая статистика трафика
- Подписки (V2Ray, Clash, Sing-box) с многоуровневой защитой
- Web UI (Preact)
- CLI интерфейс
- JWT auth
- SQLite + golang-migrate
- WARP интеграция
- GeoIP/GeoSite
- ACME сертификаты

**Security (MVP):**
- Argon2id хеширование паролей админов
- Multi-level rate limiting (IP + Token + Global)
- Failed request tracking и IP blocking
- User-Agent validation
- Enhanced access logging
- CSRF protection
- Security headers (CSP, X-Frame-Options, etc.)

**Credentials (MVP - plaintext):**
- UUID (для VMess/VLESS/TUIC v5)
- Password (для Trojan/Shadowsocks)
- Token (для TUIC v4)
- Subscription Token

### ❌ Исключено из MVP (Post-MVP):

**Протоколы:**
- TProxy
- TUN
- WireGuard
- SSH (как прокси-протокол)

**Инфраструктура:**
- HAProxy (продвинутый роутинг)

**Credentials:**
- SSH ключи (для пользователей)
- WireGuard ключи

**Функциональность:**
- Email/Webhook уведомления
- Автоматическое резервное копирование
- Multi-admin с ролями
- User portal
- QR code генерация

**Security (Post-MVP):**
- Шифрование credentials (AES-256-GCM)
- CAPTCHA для подозрительных запросов
- Geographic restrictions
- Advanced anomaly detection
- Automatic token rotation
- 2FA для админов

---

## 🎯 Post-MVP Roadmap

### v1.1 - Security Improvements
- Шифрование credentials (AES-256-GCM)
- Показ credentials только при создании/регенерации
- Encryption key rotation
- CAPTCHA для subscription endpoints
- Geographic restrictions
- Advanced anomaly detection с ML

### v1.2 - Notifications & Backup
- Email уведомления
- Webhook уведомления
- Автоматическое резервное копирование (S3, FTP, SFTP)

### v1.3 - Advanced Protocols
- TUN support
- TProxy support
- WireGuard support
- SSH как прокси-протокол

### v1.4 - User Experience
- User portal (для просмотра статистики)
- QR code генерация
- Multi-admin с ролями и правами

### v1.5 - HAProxy Integration
- HAProxy для продвинутого роутинга
- SNI-based routing
- Path-based routing
- Централизованный rate limiting

---

## 📝 Оценки времени (обновленные)

### Фаза 1: MVP Backend
- 1.1 Базовая инфраструктура: **2 недели** (было: 1 неделя)
- 1.2 Аутентификация: **1 неделя**
- 1.3 User Management: **5 дней**
- 1.4 Управление ядрами: **2 недели** (было: 1.5 недели)

**Итого Фаза 1:** ~5-6 недель (было: 3-4 недели)

### Общий MVP
**Реалистичная оценка:** 10-12 недель (2.5-3 месяца)

---

## ✅ Проверка консистентности

### Схема БД
- ✅ Все таблицы описаны
- ✅ Все поля в примерах кода существуют в схеме
- ✅ Naming conventions единообразны (snake_case)

### Примеры кода
- ✅ Go код компилируется
- ✅ SQL синтаксис для SQLite
- ✅ Нет ссылок на несуществующие поля

### Ссылки
- ✅ Внутренние ссылки между документами работают
- ✅ Нет ссылок на несуществующие файлы

---

## 🔄 Следующие шаги

1. **Начать реализацию Фазы 1.1** (Базовая инфраструктура)
2. **Создать миграции** для всех таблиц БД
3. **Настроить CI/CD** для автоматического тестирования
4. **Написать unit тесты** для критичных компонентов

---

**Документация обновлена и готова к реализации MVP.**
