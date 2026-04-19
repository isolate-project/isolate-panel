# Isolate Panel - Панель управления прокси-ядрами

**Версия:** MVP 1.0  
**Дата:** 23 марта 2026  
**Статус:** Документация готова к реализации

---

## 📚 Навигация по документации

### 🎯 Основные документы

#### [PROJECT_PLAN.md](./PROJECT_PLAN.md)
**Основной план проекта** - полная спецификация системы
- Технологический стек
- Архитектура системы
- Структура базы данных
- Фазы реализации (0-14)
- Оценки времени

#### [SECURITY_PLAN.md](./SECURITY_PLAN.md) ⭐ НОВЫЙ
**План безопасности** - критически важный документ
- Защита панели управления (JWT, Argon2id)
- Защита subscription endpoints (5 уровней)
- Хеширование паролей
- Защита от атак (SQL injection, XSS, CSRF)
- Мониторинг и алерты

#### [DECISIONS_SUMMARY.md](./DECISIONS_SUMMARY.md) ⭐ НОВЫЙ
**Итоговые решения** - результат обсуждения критики
- 10 ключевых решений
- Subscription security (критическая находка)
- Финальный MVP scope
- Post-MVP roadmap
- Чеклист реализации

### 📋 Дополнительные документы

#### [CHANGES_SUMMARY.md](./CHANGES_SUMMARY.md)
**Список изменений** - что было изменено в документации
- Критические изменения (выполнено)
- MVP scope (финальный)
- Post-MVP roadmap
- Оценки времени

#### [PROTOCOL_SMART_FORMS_PLAN.md](./PROTOCOL_SMART_FORMS_PLAN.md)
**Protocol-Aware Smart Forms** - система управления пользователями
- User Management System
- Protocol-Aware Forms
- Security и Best Practices
- Performance Considerations

#### [plan_critik.md](./plan_critik.md)
**Критический анализ** - 50 проблем в документации
- Критические противоречия
- Архитектурные несостыковки
- Технические ошибки
- План исправлений

---

## 🚀 Быстрый старт

### Для разработчиков

1. **Начните с:** [PROJECT_PLAN.md](./PROJECT_PLAN.md) - общее понимание проекта
2. **Затем изучите:** [SECURITY_PLAN.md](./SECURITY_PLAN.md) - критически важно для безопасности
3. **Ознакомьтесь с:** [DECISIONS_SUMMARY.md](./DECISIONS_SUMMARY.md) - понимание принятых решений
4. **Детали форм:** [PROTOCOL_SMART_FORMS_PLAN.md](./PROTOCOL_SMART_FORMS_PLAN.md)

### Для менеджеров проекта

1. **MVP scope:** [DECISIONS_SUMMARY.md](./DECISIONS_SUMMARY.md#mvp-scope-финальный)
2. **Оценки времени:** [DECISIONS_SUMMARY.md](./DECISIONS_SUMMARY.md#оценки-времени)
3. **Post-MVP roadmap:** [DECISIONS_SUMMARY.md](./DECISIONS_SUMMARY.md#post-mvp-roadmap)

---

## 🎯 MVP Scope (кратко)

### ✅ Включено в MVP

**Ядра:**
- Sing-box v1.13.8
- Xray v26.3.27
- Mihomo v1.19.23

**Протоколы:**
- HTTP, SOCKS5, Mixed
- Shadowsocks, VMess, VLESS, Trojan
- Hysteria2, TUIC v4/v5, Naive
- Redirect, XHTTP
- Mieru, Sudoku, TrustTunnel, ShadowsocksR, Snell

**Security:**
- Argon2id хеширование паролей админов
- Multi-level rate limiting (IP + Token + Global)
- Failed request tracking и IP blocking
- User-Agent validation
- Enhanced access logging
- CSRF protection, Security headers

**Функциональность:**
- User Management, Inbound Management
- Подписки (V2Ray, Clash, Sing-box) с защитой
- Web UI (Preact), CLI
- JWT auth, SQLite + golang-migrate
- WARP, GeoIP/GeoSite, ACME

### ❌ Исключено из MVP

**Протоколы:** TProxy, TUN, WireGuard, SSH → Post-MVP v1.3  
**Инфраструктура:** HAProxy → Post-MVP v1.5  
**Credentials:** SSH/WG ключи → Post-MVP v1.3  
**Security:** Шифрование credentials, CAPTCHA → Post-MVP v1.1

---

## 🔒 Критически важно: Subscription Security

Subscription endpoints (`/sub/*`) публично доступны из интернета и требуют **многоуровневой защиты**:

1. **Request Validation** - User-Agent, IP blocking
2. **Multi-level Rate Limiting** - IP (30/h) + Token (10/h) + Global (1000/h)
3. **Failed Request Tracking** - автоблокировка после 20 попыток
4. **Enhanced Access Logging** - IP, UA, country, response time
5. **Anomaly Detection** - фоновый мониторинг

**Подробнее:** [SECURITY_PLAN.md](./SECURITY_PLAN.md#защита-subscription-endpoints)

---

## ⏱️ Оценки времени

**MVP:** ~16-17 недель (4 месяца)
- Фаза 0: 1 неделя (Setup)
- Фаза 1: 5-6 недель (Backend MVP)
- Фаза 2: 3 недели (Frontend MVP)
- Фаза 3: 3 недели (Inbound/Outbound)
- Фаза 4: 2 недели (Подписки + Security)
- Фаза 13: 2 недели (Тестирование)

**Полная версия:** ~25-30 недель (6-7 месяцев)

---

## 🔄 Post-MVP Roadmap

### v1.1 - Security Improvements (2-3 недели)
- Шифрование credentials (AES-256-GCM)
- CAPTCHA для subscription endpoints
- Geographic restrictions
- Advanced anomaly detection

### v1.2 - Notifications & Backup (2 недели)
- Email/Webhook уведомления
- Автоматическое резервное копирование
- Шифрование бэкапов

### v1.3 - Advanced Protocols (3 недели)
- TUN, TProxy, WireGuard support
- SSH как прокси-протокол
- SSH/WireGuard ключи для пользователей

### v1.4 - User Experience (2 недели)
- User portal
- QR code генерация
- Multi-admin с ролями

### v1.5 - HAProxy Integration (2 недели)
- HAProxy для продвинутого роутинга
- SNI-based routing
- Path-based routing

---

## 📊 Статистика проекта

**Строк кода (оценка):** ~15,000-20,000 LOC  
**Файлов документации:** 6  
**Таблиц в БД:** 20+  
**API endpoints:** 50+  
**Поддерживаемых протоколов:** 15+  
**Поддерживаемых ядер:** 3

---

## 🛠️ Технологический стек

### Backend
- **Go** 1.26.1
- **Fiber** v3.1.0 (Web framework)
- **GORM** v1.31.1 (ORM)
- **SQLite** 3.x (Database)
- **Zerolog** v1.34.0 (Logging)
- **JWT** v5.3.1 (Auth)

### Frontend
- **Preact** 10.29.0 (UI framework, 3-4 KB)
- **Vite** 6.x (Build tool)
- **TypeScript** 5.9.3
- **Zustand** v5.0.12 (State management)
- **Tailwind CSS** v4.2.2

### Прокси-ядра
- **Sing-box** v1.13.8
- **Xray-core** v26.3.27
- **Mihomo** v1.19.23

---

## 🎉 Ключевые решения

1. ✅ **HAProxy → Post-MVP v1.5** - упрощение MVP
2. ✅ **SSH/WG ключи → Post-MVP v1.3** - фокус на основном функционале
3. ✅ **Plaintext credentials в MVP** - упрощение разработки
4. ✅ **Argon2id для паролей** - современный стандарт безопасности
5. ✅ **Lazy loading ядер** - экономия ресурсов
6. ✅ **Пользователь выбирает порты** - гибкость конфигурации
7. ✅ **Multi-level rate limiting** - защита subscription endpoints
8. ✅ **UUID v4 автогенерация** - упрощение для пользователя
9. ✅ **UTC в БД** - стандартный подход к timezone
10. ✅ **CORS не требуется** - localhost:8080 для панели

---

## 📞 Контакты и поддержка

**Документация:** Все файлы в корне проекта  
**Вопросы:** См. [DECISIONS_SUMMARY.md](./DECISIONS_SUMMARY.md) для деталей решений  
**Безопасность:** См. [SECURITY_PLAN.md](./SECURITY_PLAN.md) для критических аспектов

---

## ✅ Статус документации

- [x] Основной план проекта
- [x] План безопасности
- [x] Итоговые решения
- [x] Список изменений
- [x] Protocol Smart Forms
- [x] Критический анализ
- [ ] Реализация кода (следующий этап)

**Документация готова к началу разработки MVP.**

---

**Последнее обновление:** 23 марта 2026  
**Версия документации:** 1.0  
**Статус:** ✅ ГОТОВО К РЕАЛИЗАЦИИ
