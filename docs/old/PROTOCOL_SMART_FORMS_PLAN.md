# 🎯 Protocol-Aware Smart Forms & User Management System

> **Связано с:** [PROJECT_PLAN.md](./PROJECT_PLAN.md) - Фаза 1.3, Фаза 3  
> **Статус:** Спецификация завершена, реализация не начата  
> **Версия:** 1.0  
> **Дата создания:** 2026-03-22

---

## Безопасность и Best Practices

### Хранение чувствительных данных

**MVP Подход (упрощенный):**
- Credentials хранятся в plaintext для упрощения разработки
- Админ имеет полный доступ ко всем credentials пользователей
- Это приемлемо для MVP, так как панель доступна только через SSH tunnel
- Пароли админов хешируются с Argon2id (memory-hard алгоритм)

**Post-MVP Security Improvements (v1.1):**
- Шифрование user credentials (AES-256-GCM)
- Encryption key в переменной окружения `ENCRYPTION_KEY`
- Показ credentials только при создании/регенерации
- Rotation ключа шифрования через CLI
- CAPTCHA для subscription endpoints
- Geographic restrictions

**API Security:**
- Все endpoints требуют JWT аутентификацию
- Rate limiting на endpoints создания пользователей (10 req/min)
- CSRF protection для всех POST/PUT/DELETE запросов
- Security headers (CSP, X-Frame-Options, X-XSS-Protection)

**Subscription Security:**
- Multi-level rate limiting (IP: 30/hour, Token: 10/hour, Global: 1000/hour)
- Failed request tracking и IP blocking (20 неудачных попыток)
- User-Agent validation (блокировка ботов)
- Enhanced access logging (IP, UA, country, response time)
- См. [SECURITY_PLAN.md](./SECURITY_PLAN.md) для деталей

### Валидация данных

**Backend Validation:**
```go
// Username validation
func validateUsername(username string) error {
    if len(username) < 3 || len(username) > 50 {
        return errors.New("username must be 3-50 characters")
    }
    if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(username) {
        return errors.New("username can only contain alphanumeric and underscore")
    }
    return nil
}

// UUID validation
func validateUUID(id string) error {
    if _, err := uuid.Parse(id); err != nil {
        return errors.New("invalid UUID format")
    }
    return nil
}
```

**Frontend Validation:**
- Real-time валидация при вводе
- Показ ошибок inline
- Блокировка submit до исправления ошибок

### Audit Logging

**Логирование критических операций:**
- Создание/удаление пользователя
- Регенерация credentials
- Добавление/удаление пользователя из inbound
- Изменение квот

```go
// internal/audit/logger.go
func LogUserCreation(adminID uint, userID uint, username string) {
    log.Info().
        Uint("admin_id", adminID).
        Uint("user_id", userID).
        Str("username", username).
        Str("action", "user_created").
        Msg("User created")
}

func LogCredentialsRegenerated(adminID uint, userID uint) {
    log.Warn().
        Uint("admin_id", adminID).
        Uint("user_id", userID).
        Str("action", "credentials_regenerated").
        Msg("User credentials regenerated")
}
```

---

## Performance Considerations

### Database Optimization

**Индексы для быстрого поиска:**
- `idx_users_uuid` - поиск пользователя по UUID (используется при подключении)
- `idx_users_subscription_token` - генерация подписок
- `idx_user_inbound_user_id` - получение inbound для пользователя
- `idx_user_inbound_inbound_id` - получение пользователей для inbound

**Query Optimization:**
```go
// Плохо: N+1 query problem
for _, inbound := range inbounds {
    users := getUsersByInboundID(inbound.ID) // N queries
}

// Хорошо: Eager loading
inbounds := getInboundsWithUsers() // 1 query with JOIN
```

### Caching Strategy

**Redis Cache (опционально):**
- Protocol Schema Registry (TTL: 1 hour)
- User credentials lookup by UUID (TTL: 5 minutes)
- Inbound configuration (TTL: 1 minute)

**In-Memory Cache:**
- Protocol schemas загружаются при старте приложения
- Invalidation при изменении

### Config Generation Performance

**Оптимизация:**
- Генерация конфига только для измененного ядра (не всех)
- Batch updates при массовом добавлении пользователей
- Debouncing для множественных изменений

```go
// Batch add users
func (s *InboundService) AddUsersBatch(inboundID uint, userIDs []uint) error {
    // 1. Create all mappings in transaction
    tx := s.db.Begin()
    for _, userID := range userIDs {
        tx.Create(&UserInboundMapping{...})
    }
    tx.Commit()
    
    // 2. Regenerate config ONCE (not per user)
    s.regenerateCoreConfig(inbound.CoreID)
    
    // 3. Reload core ONCE
    s.coreManager.ReloadCore(inbound.CoreID)
}
```

---

## Roadmap и Future Enhancements

### Phase 1 (Current)
- ✅ User Management System
- ✅ Protocol-Aware Smart Forms
- ✅ User-Inbound Association
- ✅ CLI Interface

### Phase 2 (Future)
- [ ] **User Groups**: Группировка пользователей для массовых операций
- [ ] **Templates**: Шаблоны конфигураций inbound
- [ ] **Bulk Import**: Импорт пользователей из CSV/JSON
- [ ] **Advanced Filtering**: Фильтрация пользователей по статусу, квотам, inbound

### Phase 3 (Future)
- [ ] **User Portal**: Отдельный портал для пользователей (просмотр статистики, смена пароля)
- [ ] **QR Code Generation**: Генерация QR кодов для быстрого подключения
- [ ] **Multi-Admin**: Поддержка нескольких администраторов с разными правами
- [ ] **Webhooks**: Уведомления о событиях (создание пользователя, превышение квоты)

### Phase 4 (Future)
- [ ] **Analytics Dashboard**: Детальная аналитика по пользователям и inbound
- [ ] **Automated Actions**: Автоматическое отключение при превышении квоты
- [ ] **Backup/Restore**: Резервное копирование пользователей и конфигураций
- [ ] **API Documentation**: Swagger/OpenAPI документация

---

## Заключение

Этот документ описывает полную систему **Protocol-Aware Smart Forms & User Management** для Isolate Panel. Ключевые достижения:

### ✅ Что реализовано в плане

1. **Разделение сущностей**: Users и Inbounds независимы, связаны через mapping
2. **Универсальные credentials**: Все типы credentials генерируются при создании пользователя
3. **Protocol-Aware Forms**: Динамические формы адаптируются под протокол
4. **Wizard-based UI**: Пошаговое создание inbound с валидацией
5. **Гибкое управление**: Пользователи добавляются/удаляются из inbound после создания
6. **CLI Support**: Полный функционал доступен через CLI
7. **Security**: Шифрование приватных ключей, audit logging
8. **Performance**: Оптимизация запросов, batch operations

### 📊 Метрики успеха

- **UX**: Создание inbound за 5 шагов вместо одной сложной формы
- **Security**: 100% приватных ключей зашифрованы
- **Performance**: Batch добавление 100 пользователей < 5 секунд
- **Reliability**: Автоматическая регенерация конфигов при изменениях

### 🔗 Связанные документы

- [PROJECT_PLAN.md](./PROJECT_PLAN.md) - Основной план проекта

### 📝 Следующие шаги

1. **Обновить PROJECT_PLAN.md** с ссылками на этот документ
2. **Создать миграции** для обновления таблицы users
3. **Реализовать Protocol Schema Registry** в backend
4. **Начать разработку User Management System** (Фаза 1.2)

---

**Дата последнего обновления:** 2026-03-23  
**Версия:** 1.1  
**Статус:** Спецификация завершена, реализация не начата


### Изменения в существующих фазах

#### Фаза 1: MVP Backend

**Добавить новую подфазу 1.2 после 1.1:**

```markdown
### Фаза 1.2: User Management System (5 дней)

См. детали в [PROTOCOL_SMART_FORMS_PLAN.md](./PROTOCOL_SMART_FORMS_PLAN.md#фаза-12-user-management-system)

**Задачи:**
- [ ] Расширить модель User с универсальными credentials
- [ ] Реализовать UserService с auto-generation
- [ ] API endpoints для управления пользователями
- [ ] Шифрование чувствительных данных

**Deliverables:**
- Users CRUD через API
- Auto-generation всех типов credentials
- Безопасное хранение приватных ключей
```

#### Фаза 3: Inbound/Outbound

**Обновить существующую Фазу 3.1 и 3.2:**

```markdown
### Фаза 3: Inbound/Outbound (3 недели)

#### 3.1 Backend для Inbound + Protocol Schema Registry (1.5 недели)

См. детали в [PROTOCOL_SMART_FORMS_PLAN.md](./PROTOCOL_SMART_FORMS_PLAN.md#фаза-3-inbound-management-с-protocol-aware-forms)

**Задачи:**
- [ ] Модели Inbound, UserInboundMapping, HAProxyRoute в БД
- [ ] **Protocol Schema Registry** (новое!)
- [ ] CRUD endpoints для inbound
- [ ] **API для получения protocol schemas** (новое!)
- [ ] **API для управления пользователями в inbound** (новое!)
- [ ] Генерация конфигурации для ядер из БД
- [ ] **Динамическая регенерация конфига при изменении пользователей** (новое!)
- [ ] Генерация HAProxy routes при создании inbound
- [ ] Port Manager для выделения портов
- [ ] Валидация портов

**Deliverables:**
- Inbound создаются и управляются через API
- Protocol-Aware валидация параметров
- Пользователи добавляются/удаляются из inbound динамически
- Конфигурация ядер обновляется автоматически

#### 3.2 Frontend для Inbound + Protocol-Aware Forms (1.5 недели)

См. детали в [PROTOCOL_SMART_FORMS_PLAN.md](./PROTOCOL_SMART_FORMS_PLAN.md#32-frontend-для-inbound-полная-переработка-15-недели)

**Задачи:**
- [ ] **Wizard для создания inbound (5 шагов)** (новое!)
- [ ] **Динамические формы на основе Protocol Schema** (новое!)
- [ ] **Детальная страница inbound с управлением пользователями** (новое!)
- [ ] Редактирование inbound
- [ ] Удаление inbound

**Deliverables:**
- Wizard-based создание inbound
- Protocol-Aware формы с auto-generation
- Управление пользователями через UI
```

#### Фаза 11: CLI интерфейс

**Добавить команды для User и Inbound management:**

```markdown
### Фаза 11: CLI интерфейс (1 неделя)

См. детали в [PROTOCOL_SMART_FORMS_PLAN.md](./PROTOCOL_SMART_FORMS_PLAN.md#cli-интерфейс)

**Дополнительные команды:**

**User Management:**
- isolate-panel user create <username>
- isolate-panel user list
- isolate-panel user show <id>
- isolate-panel user credentials <id>
- isolate-panel user regenerate <id>
- isolate-panel user delete <id>

**Inbound-User Association:**
- isolate-panel inbound add-users <inbound-id> <user-ids...>
- isolate-panel inbound remove-user <inbound-id> <user-id>
- isolate-panel inbound users <inbound-id>
```

### Обновление таймлайна проекта

**Было:**
```
Фаза 1: MVP Backend (3-4 недели)
  1.1 Базовая инфраструктура (1 неделя)
  1.2 Аутентификация (1 неделя)
  1.3 Core Management (1 неделя)
  1.4 Lazy Core Loading (3 дня)
```

**Стало:**
```
Фаза 1: MVP Backend (4-5 недель)
  1.1 Базовая инфраструктура (1 неделя)
  1.2 User Management System (5 дней) ← НОВОЕ
  1.3 Аутентификация (1 неделя)
  1.4 Core Management (1 неделя)
  1.5 Lazy Core Loading (3 дня)
```

**Общее увеличение времени:** +5 дней на Фазу 1

---

## Стратегия тестирования

### Unit Tests

**Backend:**

```go
// internal/services/user_service_test.go
func TestUserService_CreateUser(t *testing.T) {
    tests := []struct {
        name    string
        req     CreateUserRequest
        wantErr bool
    }{
        {
            name: "valid user creation",
            req: CreateUserRequest{
                Username: "testuser",
                Email:    "test@example.com",
            },
            wantErr: false,
        },
        {
            name: "duplicate username",
            req: CreateUserRequest{
                Username: "existing",
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            user, err := service.CreateUser(tt.req)
            if (err != nil) != tt.wantErr {
                t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
            }
            if !tt.wantErr {
                // Verify all credentials generated
                assert.NotEmpty(t, user.UUID)
                assert.NotEmpty(t, user.Password)
                assert.NotEmpty(t, user.Token)
                assert.NotEmpty(t, user.SSHPublicKey)
                assert.NotEmpty(t, user.WireguardPrivateKey)
            }
        })
    }
}

func TestUserService_AddUsersToInbound(t *testing.T) {
    // Test adding users to inbound
    // Test duplicate prevention
    // Test config regeneration
    // Test core reload
}
```

**Frontend:**

```typescript
// src/pages/Users/UsersList.test.tsx
import { render, fireEvent, waitFor } from '@testing-library/preact';
import { UsersList } from './UsersList';

describe('UsersList', () => {
  it('should display list of users', async () => {
    const { getByText } = render(<UsersList />);
    await waitFor(() => {
      expect(getByText('alice_user')).toBeInTheDocument();
    });
  });

  it('should open create modal on button click', () => {
    const { getByText, getByRole } = render(<UsersList />);
    fireEvent.click(getByText('+ Create User'));
    expect(getByRole('dialog')).toBeInTheDocument();
  });

  it('should show credentials only once after creation', async () => {
    // Test that credentials are shown in modal after creation
    // Test that subsequent views don't show credentials
  });
});
```

### Integration Tests

```go
// tests/integration/user_inbound_test.go
func TestUserInboundIntegration(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer db.Close()
    
    // Create user
    user := createTestUser(t, db, "testuser")
    
    // Create inbound
    inbound := createTestInbound(t, db, "vmess-443", "vmess")
    
    // Add user to inbound
    err := addUserToInbound(t, db, user.ID, inbound.ID)
    assert.NoError(t, err)
    
    // Verify mapping created
    mapping := getUserInboundMapping(t, db, user.ID, inbound.ID)
    assert.NotNil(t, mapping)
    
    // Verify config regenerated
    config := getCoreConfig(t, inbound.CoreID)
    assert.Contains(t, config, user.UUID)
    
    // Remove user from inbound
    err = removeUserFromInbound(t, db, user.ID, inbound.ID)
    assert.NoError(t, err)
    
    // Verify mapping deleted
    mapping = getUserInboundMapping(t, db, user.ID, inbound.ID)
    assert.Nil(t, mapping)
    
    // Verify config regenerated without user
    config = getCoreConfig(t, inbound.CoreID)
    assert.NotContains(t, config, user.UUID)
}
```

### E2E Tests

```typescript
// e2e/user-inbound-workflow.spec.ts
import { test, expect } from '@playwright/test';

test('complete user-inbound workflow', async ({ page }) => {
  // Login
  await page.goto('http://localhost:8080/login');
  await page.fill('[name=username]', 'admin');
  await page.fill('[name=password]', 'admin');
  await page.click('button[type=submit]');

  // Create user
  await page.goto('http://localhost:8080/users');
  await page.click('text=+ Create User');
  await page.fill('[name=username]', 'e2e-test-user');
  await page.click('text=Create User');
  
  // Save credentials (shown once)
  const uuid = await page.textContent('[data-testid=user-uuid]');
  expect(uuid).toBeTruthy();
  await page.click('text=Close');
  
  // Create inbound
  await page.goto('http://localhost:8080/inbounds');
  await page.click('text=+ Create Inbound');
  // ... wizard steps
  
  // Add user to inbound
  await page.goto('http://localhost:8080/inbounds/1');
  await page.click('text=Users');
  await page.click('text=+ Add Users');
  await page.check('text=e2e-test-user');
  await page.click('text=Add Users');
  
  // Verify user added
  await expect(page.locator('text=e2e-test-user')).toBeVisible();
});
```

### Test Coverage Goals

- **Backend:** ≥ 80% code coverage
- **Frontend:** ≥ 70% code coverage
- **Critical paths:** 100% coverage (user creation, credential generation, inbound-user association)

---


### Пример 1: Создание пользователя и добавление в Inbound

**Сценарий:** Администратор создает нового пользователя и добавляет его в существующий VMess inbound.

**Шаги:**

1. **Создать пользователя через UI:**
   - Перейти на `/users`
   - Нажать `[+ Create User]`
   - Ввести username: `alice`
   - Установить квоту: 100 GB
   - Нажать `[Create User]`
   - **Сохранить все credentials** (показываются один раз!)

2. **Добавить пользователя в inbound:**
   - Перейти на `/inbounds/5` (vmess-443)
   - Перейти на таб `[Users]`
   - Нажать `[+ Add Users]`
   - Выбрать `alice` из списка
   - Нажать `[Add Users]`
   - Система автоматически:
     - Создаст запись в `user_inbound_mapping`
     - Регенерирует конфиг Sing-box
     - Перезагрузит Sing-box core

3. **Результат:**
   - Пользователь `alice` может подключаться к VMess inbound на порту 443
   - Используя UUID, сгенерированный при создании

### Пример 2: Создание Trojan Inbound с TLS

**Сценарий:** Администратор создает Trojan inbound, который требует TLS.

**Wizard Flow:**

```
Step 1: Select Core → Sing-box
Step 2: Select Protocol → Trojan

⚠️ Trojan requires TLS configuration

Step 3: Basic Settings
  Name: trojan-443
  Port: 443

Step 4: Transport → Skip

Step 5: TLS (Required) ← Автоматически раскрыта!
  ☑ Enable TLS
  Certificate: [Select existing certificate]
  
Review & Create
  ✓ All required fields filled
  [Create]
```

**Что происходит:**
- При выборе Trojan на Step 2, система проверяет `RequiresTLS: true` в Protocol Schema
- На Step 5 секция TLS автоматически раскрывается и помечается как обязательная
- Кнопка `[Create]` блокируется до заполнения TLS конфигурации

### Пример 3: Массовое добавление пользователей

**Сценарий:** Администратор хочет добавить 10 пользователей в один inbound.

**CLI подход:**

```bash
# Создать 10 пользователей
for i in {1..10}; do
  isolate-panel user create user$i --traffic-limit=50
done

# Получить ID пользователей
isolate-panel user list --format=id > user_ids.txt

# Массово добавить в inbound
isolate-panel inbound add-users vmess-443 $(cat user_ids.txt)
```

**API подход:**

```bash
# Bulk add через API
curl -X POST http://localhost:8080/api/inbounds/5/users/bulk \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_ids": [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
  }'
```

### Пример 4: Регенерация credentials пользователя

**Сценарий:** Credentials пользователя скомпрометированы, нужно их обновить.

**UI подход:**

1. Перейти на `/users`
2. Раскрыть детали пользователя `bob`
3. Нажать `[Regenerate Credentials]`
4. Подтвердить действие (предупреждение!)
5. Получить новые credentials (показываются один раз!)
6. Система автоматически:
   - Генерирует новые UUID, Password, Token, SSH keys, WG keys
   - Обновляет запись в БД
   - Регенерирует конфиги всех inbound, где используется пользователь
   - Перезагружает соответствующие ядра

**CLI подход:**

```bash
isolate-panel user regenerate bob --confirm

⚠️  WARNING: This will invalidate all existing connections!
   All inbounds using this user will be reloaded.

Proceed? [y/N]: y

✓ Credentials regenerated successfully!

New credentials:
UUID: new-uuid-here
Password: new-password-here
...
```

---


> **Добавить в Фазу 11 PROJECT_PLAN.md**

### User Management Commands

```bash
# Создать пользователя
isolate-panel user create <username> [--email=<email>] [--traffic-limit=<GB>] [--expiry=<date>]

# Примеры:
isolate-panel user create alice
isolate-panel user create bob --email=bob@example.com --traffic-limit=100 --expiry=2026-12-31

# Список пользователей
isolate-panel user list [--active] [--expired] [--limit=<n>]

# Детали пользователя
isolate-panel user show <username|id>

# Показать credentials (только для администратора)
isolate-panel user credentials <username|id>

# Регенерировать credentials
isolate-panel user regenerate <username|id> [--confirm]

# Удалить пользователя
isolate-panel user delete <username|id> [--force]

# Обновить квоты
isolate-panel user update <username|id> --traffic-limit=<GB> --expiry=<date>
```

### Inbound Management Commands

```bash
# Создать inbound (интерактивный wizard)
isolate-panel inbound create

# Создать inbound (неинтерактивно)
isolate-panel inbound create \
  --core=singbox \
  --protocol=vmess \
  --name=vmess-443 \
  --port=443 \
  --listen=0.0.0.0

# Список inbound
isolate-panel inbound list [--core=<name>] [--protocol=<name>]

# Детали inbound
isolate-panel inbound show <id|name>

# Удалить inbound
isolate-panel inbound delete <id|name> [--force]

# Добавить пользователей к inbound
isolate-panel inbound add-users <inbound-id> <user-id1> [user-id2] [...]

# Примеры:
isolate-panel inbound add-users 1 5 6 7
isolate-panel inbound add-users vmess-443 alice bob charlie

# Удалить пользователя из inbound
isolate-panel inbound remove-user <inbound-id> <user-id>

# Список пользователей inbound
isolate-panel inbound users <inbound-id>
```

### Interactive Wizard Example

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
4) Shadowsocks
5) Hysteria2
6) TUIC v5
7) Naive
8) WireGuard

Select protocol [1]: 1

Step 3/5: Basic Settings
------------------------
Name [vmess-443]: 
Listen address [0.0.0.0]: 
Port [443]: 

✓ Port 443 is available

Step 4/5: Transport (Optional)
-------------------------------
Enable transport layer? [y/N]: n

Step 5/5: TLS (Optional)
------------------------
Enable TLS? [y/N]: n

Review Configuration
--------------------
Core: Sing-box
Protocol: VMess
Name: vmess-443
Port: 443
Transport: None
TLS: Disabled

Create inbound? [Y/n]: y

✓ Inbound created successfully!
  ID: 5
  Name: vmess-443
  Port: 443

You can now add users to this inbound:
  isolate-panel inbound add-users 5 <user-id>
```

### Implementation

```go
// cmd/cli/commands/user.go
package commands

import (
    "fmt"
    "github.com/spf13/cobra"
)

var userCmd = &cobra.Command{
    Use:   "user",
    Short: "Manage users",
}

var userCreateCmd = &cobra.Command{
    Use:   "create <username>",
    Short: "Create a new user",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        username := args[0]
        email, _ := cmd.Flags().GetString("email")
        trafficLimit, _ := cmd.Flags().GetInt64("traffic-limit")
        expiry, _ := cmd.Flags().GetString("expiry")
        
        // Call API
        user, err := apiClient.CreateUser(CreateUserRequest{
            Username:       username,
            Email:          email,
            TrafficLimitGB: trafficLimit,
            ExpiryDate:     parseDate(expiry),
        })
        if err != nil {
            return fmt.Errorf("failed to create user: %w", err)
        }
        
        // Display credentials (ONLY ONCE!)
        fmt.Println("✓ User created successfully!")
        fmt.Println()
        fmt.Println("⚠️  IMPORTANT: Save these credentials now!")
        fmt.Println("   They will NOT be shown again!")
        fmt.Println()
        fmt.Printf("Username: %s\n", user.Username)
        fmt.Printf("UUID: %s\n", user.UUID)
        fmt.Printf("Password: %s\n", user.Password)
        fmt.Printf("Token: %s\n", user.Token)
        fmt.Printf("Subscription Token: %s\n", user.SubscriptionToken)
        fmt.Println()
        fmt.Println("SSH Public Key:")
        fmt.Println(user.SSHPublicKey)
        fmt.Println()
        fmt.Println("SSH Private Key:")
        fmt.Println(user.SSHPrivateKey)
        fmt.Println()
        fmt.Printf("WireGuard Private Key: %s\n", user.WireguardPrivateKey)
        fmt.Printf("WireGuard Public Key: %s\n", user.WireguardPublicKey)
        
        return nil
    },
}

func init() {
    userCreateCmd.Flags().String("email", "", "User email")
    userCreateCmd.Flags().Int64("traffic-limit", 0, "Traffic limit in GB (0 = unlimited)")
    userCreateCmd.Flags().String("expiry", "", "Expiry date (YYYY-MM-DD)")
    
    userCmd.AddCommand(userCreateCmd)
    // ... other subcommands
}
```

---


### Детальная страница Inbound

**URL:** `/inbounds/:id`

**Layout:**

```
┌─────────────────────────────────────────────────────────────┐
│ ← Back to Inbounds                                          │
├─────────────────────────────────────────────────────────────┤
│ vmess-443                                    [Edit] [Delete]│
│ VMess · Port 443 · Sing-box · ● Running                    │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│ [Overview] [Users] [Config] [Stats]                        │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                              │
│ Users (2)                              [+ Add Users]        │
│                                                              │
│ ┌─────────────────────────────────────────────────────┐    │
│ │ alice_user                                          │    │
│ │ UUID: 12345678-abcd...                              │    │
│ │ Traffic: 5.2 GB / 100 GB  │  Status: ● Active      │    │
│ │ Added: 2026-03-20                          [Remove] │    │
│ ├─────────────────────────────────────────────────────┤    │
│ │ bob_user                                            │    │
│ │ UUID: 87654321-dcba...                              │    │
│ │ Traffic: 12.5 GB / ∞      │  Status: ● Active      │    │
│ │ Added: 2026-03-19                          [Remove] │    │
│ └─────────────────────────────────────────────────────┘    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Add Users Modal

**Открывается при клике на [+ Add Users]:**

```
┌─ Add Users to vmess-443 ──────────────────────────────┐
│                                                        │
│ Select users to add:                                  │
│                                                        │
│ Search: [_________________] 🔍                        │
│                                                        │
│ ┌────────────────────────────────────────────────┐   │
│ │ ☑ Select All (3 available)                     │   │
│ ├────────────────────────────────────────────────┤   │
│ │ ☑ alice_user                                   │   │
│ │   UUID: 12345678-abcd...                       │   │
│ │   Traffic: 5.2 GB / 100 GB                     │   │
│ ├────────────────────────────────────────────────┤   │
│ │ ☑ bob_user                                     │   │
│ │   UUID: 87654321-dcba...                       │   │
│ │   Traffic: 12.5 GB / ∞                         │   │
│ ├────────────────────────────────────────────────┤   │
│ │ ☐ charlie_user                                 │   │
│ │   UUID: abcdef12-3456...                       │   │
│ │   Traffic: 0 B / 50 GB                         │   │
│ └────────────────────────────────────────────────┘   │
│                                                        │
│ Selected: 2 users                                     │
│                                                        │
│ ⚠️ Core will be reloaded after adding users           │
│                                                        │
│                          [Cancel]  [Add Users]        │
└────────────────────────────────────────────────────────┘
```

### Компонент InboundUsersManager.tsx

```typescript
// src/pages/Inbounds/InboundDetails/InboundUsersManager.tsx
import { useState } from 'preact/hooks';
import { useInboundUsers } from '@/hooks/useInboundUsers';
import { useUsers } from '@/hooks/useUsers';

interface Props {
  inboundId: number;
}

export function InboundUsersManager({ inboundId }: Props) {
  const { users: inboundUsers, loading, addUsers, removeUser } = useInboundUsers(inboundId);
  const { users: allUsers } = useUsers();
  const [showAddModal, setShowAddModal] = useState(false);
  const [selectedUserIds, setSelectedUserIds] = useState<number[]>([]);

  // Фильтруем пользователей, которых еще нет в inbound
  const availableUsers = allUsers.filter(
    user => !inboundUsers.some(iu => iu.id === user.id)
  );

  const handleAddUsers = async () => {
    try {
      await addUsers(selectedUserIds);
      setShowAddModal(false);
      setSelectedUserIds([]);
      // Показать success toast
    } catch (error) {
      // Показать error toast
    }
  };

  const handleRemoveUser = async (userId: number) => {
    if (confirm('Remove this user from inbound? Core will be reloaded.')) {
      try {
        await removeUser(userId);
        // Показать success toast
      } catch (error) {
        // Показать error toast
      }
    }
  };

  const toggleSelectAll = () => {
    if (selectedUserIds.length === availableUsers.length) {
      setSelectedUserIds([]);
    } else {
      setSelectedUserIds(availableUsers.map(u => u.id));
    }
  };

  return (
    <div class="space-y-4">
      <div class="flex justify-between items-center">
        <h2 class="text-xl font-semibold">
          Users ({inboundUsers.length})
        </h2>
        <button 
          onClick={() => setShowAddModal(true)}
          class="btn btn-primary"
          disabled={availableUsers.length === 0}
        >
          + Add Users
        </button>
      </div>

      {loading ? (
        <div>Loading...</div>
      ) : (
        <div class="space-y-2">
          {inboundUsers.map(user => (
            <div key={user.id} class="border rounded-lg p-4">
              <div class="flex justify-between items-start">
                <div>
                  <div class="font-semibold">{user.username}</div>
                  <div class="text-sm text-gray-600">
                    UUID: {user.uuid.substring(0, 16)}...
                  </div>
                  <div class="text-sm text-gray-600">
                    Traffic: {formatBytes(user.traffic_used_bytes)} / 
                    {user.traffic_limit_bytes ? formatBytes(user.traffic_limit_bytes) : '∞'}
                  </div>
                  <div class="text-sm text-gray-600">
                    Added: {formatDate(user.added_at)}
                  </div>
                </div>
                <button
                  onClick={() => handleRemoveUser(user.id)}
                  class="btn btn-sm btn-danger"
                >
                  Remove
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {showAddModal && (
        <div class="modal">
          <div class="modal-content">
            <h3 class="text-lg font-semibold mb-4">
              Add Users to Inbound
            </h3>

            <input
              type="text"
              placeholder="Search users..."
              class="input mb-4"
            />

            <div class="border rounded-lg max-h-96 overflow-y-auto">
              <div class="p-2 border-b">
                <label class="flex items-center">
                  <input
                    type="checkbox"
                    checked={selectedUserIds.length === availableUsers.length}
                    onChange={toggleSelectAll}
                    class="mr-2"
                  />
                  Select All ({availableUsers.length} available)
                </label>
              </div>

              {availableUsers.map(user => (
                <div key={user.id} class="p-2 border-b hover:bg-gray-50">
                  <label class="flex items-center">
                    <input
                      type="checkbox"
                      checked={selectedUserIds.includes(user.id)}
                      onChange={(e) => {
                        if (e.currentTarget.checked) {
                          setSelectedUserIds([...selectedUserIds, user.id]);
                        } else {
                          setSelectedUserIds(selectedUserIds.filter(id => id !== user.id));
                        }
                      }}
                      class="mr-2"
                    />
                    <div>
                      <div class="font-semibold">{user.username}</div>
                      <div class="text-sm text-gray-600">
                        UUID: {user.uuid.substring(0, 16)}...
                      </div>
                    </div>
                  </label>
                </div>
              ))}
            </div>

            <div class="mt-4 text-sm text-gray-600">
              Selected: {selectedUserIds.length} users
            </div>

            <div class="mt-4 p-3 bg-yellow-50 border border-yellow-200 rounded">
              ⚠️ Core will be reloaded after adding users
            </div>

            <div class="flex justify-end space-x-2 mt-4">
              <button
                onClick={() => {
                  setShowAddModal(false);
                  setSelectedUserIds([]);
                }}
                class="btn btn-secondary"
              >
                Cancel
              </button>
              <button
                onClick={handleAddUsers}
                disabled={selectedUserIds.length === 0}
                class="btn btn-primary"
              >
                Add Users
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
```

### Hook useInboundUsers

```typescript
// src/hooks/useInboundUsers.ts
import { useState, useEffect } from 'preact/hooks';
import { api } from '@/services/api';

export function useInboundUsers(inboundId: number) {
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const fetchUsers = async () => {
    try {
      setLoading(true);
      const response = await api.get(`/api/inbounds/${inboundId}/users`);
      setUsers(response.data);
    } catch (err) {
      setError(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchUsers();
  }, [inboundId]);

  const addUsers = async (userIds: number[]) => {
    await api.post(`/api/inbounds/${inboundId}/users`, { user_ids: userIds });
    await fetchUsers(); // Refresh list
  };

  const removeUser = async (userId: number) => {
    await api.delete(`/api/inbounds/${inboundId}/users/${userId}`);
    await fetchUsers(); // Refresh list
  };

  return { users, loading, error, addUsers, removeUser, refresh: fetchUsers };
}
```

**Acceptance Criteria:**
- ✅ Детальная страница inbound с табами (Overview, Users, Config, Stats)
- ✅ Список пользователей inbound с возможностью удаления
- ✅ Modal для добавления пользователей с поиском и фильтрацией
- ✅ Checkbox "Select All" для массового выбора
- ✅ Показ только доступных пользователей (не добавленных в inbound)
- ✅ Предупреждение о перезагрузке ядра
- ✅ Real-time обновление списка после добавления/удаления
- ✅ Loading states и error handling

---


> **Обновление существующей Фазы 3 в PROJECT_PLAN.md**  
> **Длительность:** 3 недели (1.5 недели backend + 1.5 недели frontend)

### 3.1 Backend для Inbound (расширение, 1.5 недели)

**Дополнительные задачи к существующим:**
- [ ] Интеграция Protocol Schema Registry
- [ ] Валидация параметров inbound на основе schema
- [ ] API endpoint для получения schema протокола
- [ ] Генерация конфига на основе schema + user input
- [ ] API для управления пользователями в inbound
- [ ] Динамическая регенерация конфига ядра при изменении пользователей

**Новые API Endpoints:**

```
# Protocol Schema
GET    /api/protocols                    # Список всех протоколов
GET    /api/protocols/:name/schema       # Schema конкретного протокола
GET    /api/protocols/by-core/:core      # Протоколы для конкретного ядра

# Inbound (дополнение к существующим CRUD)
POST   /api/inbounds/:id/users           # Добавить пользователей к inbound
DELETE /api/inbounds/:id/users/:userId   # Удалить пользователя из inbound
GET    /api/inbounds/:id/users           # Список пользователей inbound
POST   /api/inbounds/:id/users/bulk      # Массовое добавление пользователей
```

**Inbound Service с поддержкой пользователей:**

```go
// internal/services/inbound_service.go
package services

type InboundService struct {
    repo           *repositories.InboundRepository
    userRepo       *repositories.UserRepository
    mappingRepo    *repositories.UserInboundMappingRepository
    configGen      *config.ConfigGenerator
    coreManager    *core.CoreManager
}

// AddUsersToInbound добавляет пользователей к inbound
func (s *InboundService) AddUsersToInbound(inboundID uint, userIDs []uint, adminID uint) error {
    // 1. Проверить существование inbound
    inbound, err := s.repo.GetByID(inboundID)
    if err != nil {
        return fmt.Errorf("inbound not found: %w", err)
    }
    
    // 2. Проверить существование всех пользователей
    users, err := s.userRepo.GetByIDs(userIDs)
    if err != nil {
        return fmt.Errorf("failed to get users: %w", err)
    }
    if len(users) != len(userIDs) {
        return fmt.Errorf("some users not found")
    }
    
    // 3. Создать mappings
    for _, userID := range userIDs {
        mapping := &models.UserInboundMapping{
            UserID:         userID,
            InboundID:      inboundID,
            AddedByAdminID: &adminID,
        }
        if err := s.mappingRepo.Create(mapping); err != nil {
            // Игнорируем ошибку дубликата (UNIQUE constraint)
            if !errors.Is(err, gorm.ErrDuplicatedKey) {
                return fmt.Errorf("failed to create mapping: %w", err)
            }
        }
    }
    
    // 4. Регенерировать конфиг ядра
    if err := s.regenerateCoreConfig(inbound.CoreID); err != nil {
        return fmt.Errorf("failed to regenerate config: %w", err)
    }
    
    // 5. Применить изменения (reload или restart ядра)
    if err := s.coreManager.ReloadCore(inbound.CoreID); err != nil {
        return fmt.Errorf("failed to reload core: %w", err)
    }
    
    return nil
}

// RemoveUserFromInbound удаляет пользователя из inbound
func (s *InboundService) RemoveUserFromInbound(inboundID uint, userID uint) error {
    // 1. Удалить mapping
    if err := s.mappingRepo.Delete(userID, inboundID); err != nil {
        return fmt.Errorf("failed to delete mapping: %w", err)
    }
    
    // 2. Получить inbound для определения ядра
    inbound, err := s.repo.GetByID(inboundID)
    if err != nil {
        return fmt.Errorf("inbound not found: %w", err)
    }
    
    // 3. Регенерировать конфиг ядра
    if err := s.regenerateCoreConfig(inbound.CoreID); err != nil {
        return fmt.Errorf("failed to regenerate config: %w", err)
    }
    
    // 4. Применить изменения
    if err := s.coreManager.ReloadCore(inbound.CoreID); err != nil {
        return fmt.Errorf("failed to reload core: %w", err)
    }
    
    return nil
}

// regenerateCoreConfig регенерирует конфиг для ядра
func (s *InboundService) regenerateCoreConfig(coreID uint) error {
    // 1. Получить все inbound для этого ядра
    inbounds, err := s.repo.GetByCoreID(coreID)
    if err != nil {
        return err
    }
    
    // 2. Для каждого inbound получить пользователей
    for i := range inbounds {
        users, err := s.userRepo.GetByInboundID(inbounds[i].ID)
        if err != nil {
            return err
        }
        inbounds[i].Users = users
    }
    
    // 3. Сгенерировать конфиг
    config, err := s.configGen.GenerateConfig(coreID, inbounds)
    if err != nil {
        return err
    }
    
    // 4. Сохранить конфиг в файл
    if err := s.configGen.SaveConfig(coreID, config); err != nil {
        return err
    }
    
    return nil
}
```

**Config Generator с поддержкой пользователей:**

```go
// internal/config/generator.go
package config

// GenerateConfig генерирует конфиг для ядра на основе inbound и пользователей
func (g *ConfigGenerator) GenerateConfig(coreID uint, inbounds []models.Inbound) (string, error) {
    core, err := g.coreRepo.GetByID(coreID)
    if err != nil {
        return "", err
    }
    
    switch core.Name {
    case "singbox":
        return g.generateSingboxConfig(inbounds)
    case "xray":
        return g.generateXrayConfig(inbounds)
    case "mihomo":
        return g.generateMihomoConfig(inbounds)
    default:
        return "", fmt.Errorf("unknown core: %s", core.Name)
    }
}

// generateSingboxConfig генерирует конфиг для Sing-box
func (g *ConfigGenerator) generateSingboxConfig(inbounds []models.Inbound) (string, error) {
    config := map[string]interface{}{
        "log": map[string]interface{}{
            "level": "info",
        },
        "inbounds": []map[string]interface{}{},
    }
    
    for _, inbound := range inbounds {
        inboundConfig := map[string]interface{}{
            "type":   inbound.Protocol,
            "tag":    inbound.Name,
            "listen": inbound.ListenAddress,
            "port":   inbound.Port,
        }
        
        // Добавить пользователей
        if len(inbound.Users) > 0 {
            users := []map[string]interface{}{}
            for _, user := range inbound.Users {
                userConfig := g.generateUserConfig(inbound.Protocol, user)
                users = append(users, userConfig)
            }
            inboundConfig["users"] = users
        }
        
        // Добавить дополнительные параметры из config_json
        var additionalConfig map[string]interface{}
        if err := json.Unmarshal([]byte(inbound.ConfigJSON), &additionalConfig); err == nil {
            for k, v := range additionalConfig {
                if k != "users" { // не перезаписываем users
                    inboundConfig[k] = v
                }
            }
        }
        
        config["inbounds"] = append(config["inbounds"].([]map[string]interface{}), inboundConfig)
    }
    
    jsonBytes, err := json.MarshalIndent(config, "", "  ")
    if err != nil {
        return "", err
    }
    
    return string(jsonBytes), nil
}

// generateUserConfig генерирует конфиг пользователя для протокола
func (g *ConfigGenerator) generateUserConfig(protocol string, user models.User) map[string]interface{} {
    switch protocol {
    case "vmess", "vless":
        return map[string]interface{}{
            "name": user.Username,
            "uuid": user.UUID,
        }
    case "trojan":
        return map[string]interface{}{
            "name":     user.Username,
            "password": user.Password,
        }
    case "shadowsocks":
        return map[string]interface{}{
            "name":     user.Username,
            "password": user.Password,
        }
    // ... другие протоколы
    default:
        return map[string]interface{}{
            "name": user.Username,
        }
    }
}
```

**Acceptance Criteria:**
- ✅ Protocol Schema Registry интегрирован
- ✅ API возвращает schema для каждого протокола
- ✅ Валидация параметров на основе schema
- ✅ Пользователи добавляются/удаляются из inbound через API
- ✅ Конфиг ядра автоматически регенерируется при изменении пользователей
- ✅ Поддержка массового добавления пользователей
- ✅ Unit и integration тесты

### 3.2 Frontend для Inbound (полная переработка, 1.5 недели)

**Новая структура компонентов:**

```
src/pages/Inbounds/
├── InboundsList.tsx                    # Список inbound
├── InboundCreate/
│   ├── InboundCreateWizard.tsx         # Главный wizard компонент
│   ├── Step1CoreSelection.tsx          # Шаг 1: Выбор ядра
│   ├── Step2ProtocolSelection.tsx      # Шаг 2: Выбор протокола
│   ├── Step3BasicSettings.tsx          # Шаг 3: Базовые настройки
│   ├── Step4Transport.tsx              # Шаг 4: Транспорт (опционально)
│   ├── Step5TLS.tsx                    # Шаг 5: TLS (опционально/обязательно)
│   └── StepReview.tsx                  # Обзор и создание
├── InboundDetails/
│   ├── InboundDetailsPage.tsx          # Детальная страница inbound
│   ├── InboundUsersManager.tsx         # Управление пользователями
│   ├── InboundConfigPreview.tsx        # Превью конфига
│   └── InboundStats.tsx                # Статистика inbound
└── components/
    ├── ProtocolForm.tsx                # Динамическая форма протокола
    ├── ParameterField.tsx              # Поле параметра с auto-gen
    ├── ExpandableSection.tsx           # Раскрывающаяся секция
    └── ValidationSummary.tsx           # Сводка валидации
```

**Wizard Flow (пошаговое создание):**

```
Шаг 1: Выбор ядра
┌─────────────────────────────────────────────────────────┐
│ Create Inbound - Step 1 of 5: Select Core              │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                          │
│ Which core do you want to use?                          │
│                                                          │
│ ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│ │ Sing-box │  │   Xray   │  │  Mihomo  │              │
│ │          │  │          │  │          │              │
│ │ ✓ Recom. │  │  XHTTP   │  │ Mihomo-  │              │
│ │  Most    │  │  support │  │ specific │              │
│ │ protocols│  │          │  │ protocols│              │
│ │ [Select] │  │ [Select] │  │ [Select] │              │
│ └──────────┘  └──────────┘  └──────────┘              │
│                                                          │
│                              [Cancel]  [Next →]         │
└─────────────────────────────────────────────────────────┘

Шаг 2: Выбор протокола
┌─────────────────────────────────────────────────────────┐
│ Create Inbound - Step 2 of 5: Select Protocol          │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                          │
│ Core: Sing-box                                          │
│                                                          │
│ Available protocols:                                    │
│                                                          │
│ ○ VMess - Versatile protocol                           │
│ ○ VLESS - Lightweight, modern                          │
│ ○ Trojan - TLS-based (requires certificate)           │
│ ○ Shadowsocks - Fast and simple                        │
│ ○ Hysteria2 - Low latency, QUIC-based                 │
│ ○ TUIC v5 - QUIC-based                                 │
│ ○ Naive - Chrome-based (Sing-box only)                │
│ ○ WireGuard - VPN protocol                             │
│                                                          │
│                    [← Back]  [Cancel]  [Next →]        │
└─────────────────────────────────────────────────────────┘

Шаг 3: Базовые настройки
┌─────────────────────────────────────────────────────────┐
│ Create Inbound - Step 3 of 5: Basic Settings           │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                          │
│ Core: Sing-box  │  Protocol: VMess                     │
│                                                          │
│ ▼ Network Settings                                      │
│   Name: [vmess-443_____________]                        │
│   Listen Address: [0.0.0.0__] (default)                │
│   Port: [443___] [🎲 Random]                           │
│         ✓ Port available                                │
│                                                          │
│ ▼ Protocol Parameters                                   │
│   Cipher: [auto ▼]                                     │
│   AlterID: [0] (recommended)                           │
│                                                          │
│                    [← Back]  [Cancel]  [Next →]        │
└─────────────────────────────────────────────────────────┘

Шаг 4: Transport (опционально)
┌─────────────────────────────────────────────────────────┐
│ Create Inbound - Step 4 of 5: Transport (Optional)     │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                          │
│ ☐ Enable transport layer                               │
│                                                          │
│ ▶ Transport Configuration (click to expand)            │
│                                                          │
│                    [← Back]  [Cancel]  [Next →]        │
└─────────────────────────────────────────────────────────┘

Шаг 5: TLS (опционально или обязательно для Trojan)
┌─────────────────────────────────────────────────────────┐
│ Create Inbound - Step 5 of 5: TLS                      │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                          │
│ ☐ Enable TLS                                            │
│                                                          │
│ ▶ TLS Configuration (click to expand)                  │
│                                                          │
│                    [← Back]  [Cancel]  [Next →]        │
└─────────────────────────────────────────────────────────┘

Обзор и создание
┌─────────────────────────────────────────────────────────┐
│ Create Inbound - Review & Create                        │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                          │
│ Core: Sing-box                                          │
│ Protocol: VMess                                         │
│ Name: vmess-443                                         │
│ Port: 443                                               │
│ Transport: None                                         │
│ TLS: Disabled                                           │
│                                                          │
│ ℹ️ You can add users to this inbound after creation    │
│                                                          │
│                    [← Back]  [Cancel]  [Create]        │
└─────────────────────────────────────────────────────────┘
```

**Acceptance Criteria:**
- ✅ Wizard с 5 шагами для создания inbound
- ✅ Выбор ядра фильтрует доступные протоколы
- ✅ Динамические формы на основе Protocol Schema
- ✅ Auto-generation параметров с возможностью кастомизации
- ✅ Автоматическое раскрытие обязательных секций (TLS для Trojan)
- ✅ Валидация на каждом шаге
- ✅ Блокировка кнопки "Create" до заполнения всех обязательных полей
- ✅ Preview конфига перед созданием

---


> **Добавить в PROJECT_PLAN.md после Фазы 1.1**  
> **Длительность:** 5 дней (3 дня backend + 2 дня frontend)

### 1.2.1 Backend для Users (3 дня)

**Задачи:**
- [ ] Расширить модель `User` с универсальными credentials
- [ ] Создать `UserService` с методами CRUD
- [ ] Реализовать auto-generation всех типов credentials при создании
- [ ] API endpoints для управления пользователями
- [ ] Валидация уникальности UUID, Token, Username
- [ ] Шифрование чувствительных данных (SSH private keys)
- [ ] Endpoint для регенерации credentials

**API Endpoints:**

```
POST   /api/users              # Создать пользователя
GET    /api/users              # Список пользователей (с пагинацией)
GET    /api/users/:id          # Детали пользователя
PUT    /api/users/:id          # Обновить пользователя
DELETE /api/users/:id          # Удалить пользователя
POST   /api/users/:id/regenerate  # Регенерировать credentials
GET    /api/users/:id/inbounds    # Список inbound для пользователя
```

**User Creation Flow:**

```go
// internal/services/user_service.go
package services

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "isolate-panel/internal/models"
    "isolate-panel/internal/protocol"
    "golang.org/x/crypto/bcrypt"
)

type UserService struct {
    repo          *repositories.UserRepository
    encryptionKey []byte // 32 bytes для AES-256
}

func (s *UserService) CreateUser(req CreateUserRequest) (*models.User, error) {
    // Валидация
    if err := s.validateUsername(req.Username); err != nil {
        return nil, err
    }
    
    user := &models.User{
        Username: req.Username,
        Email:    req.Email,
        
        // Auto-generate ALL credentials
        UUID:              protocol.GenerateUUIDv4(),
        Password:          protocol.GeneratePassword(16),
        Token:             protocol.GenerateBase64Token(32),
        SubscriptionToken: protocol.GenerateBase64Token(32),
    }
    
    // Generate SSH keypair
    sshPriv, sshPub, err := protocol.GenerateSSHKeypair()
    if err != nil {
        return nil, fmt.Errorf("failed to generate SSH keys: %w", err)
    }
    user.SSHPublicKey = sshPub
    user.SSHPrivateKeyEncrypted = s.encryptPrivateKey(sshPriv)
    
    // Generate WireGuard keypair
    wgPriv, wgPub, err := protocol.GenerateWireGuardKeypair()
    if err != nil {
        return nil, fmt.Errorf("failed to generate WireGuard keys: %w", err)
    }
    user.WireguardPrivateKey = wgPriv
    user.WireguardPublicKey = wgPub
    
    // Hash password for storage
    passwordHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
    if err != nil {
        return nil, fmt.Errorf("failed to hash password: %w", err)
    }
    user.PasswordHash = string(passwordHash)
    
    // Set quotas if provided
    if req.TrafficLimitGB > 0 {
        user.TrafficLimitBytes = req.TrafficLimitGB * 1024 * 1024 * 1024
    }
    user.ExpiryDate = req.ExpiryDate
    
    // Save to database
    if err := s.repo.Create(user); err != nil {
        return nil, err
    }
    
    return user, nil
}

func (s *UserService) encryptPrivateKey(privateKey string) string {
    block, _ := aes.NewCipher(s.encryptionKey)
    gcm, _ := cipher.NewGCM(block)
    nonce := make([]byte, gcm.NonceSize())
    rand.Read(nonce)
    ciphertext := gcm.Seal(nonce, nonce, []byte(privateKey), nil)
    return base64.StdEncoding.EncodeToString(ciphertext)
}

func (s *UserService) decryptPrivateKey(encrypted string) (string, error) {
    data, _ := base64.StdEncoding.DecodeString(encrypted)
    block, _ := aes.NewCipher(s.encryptionKey)
    gcm, _ := cipher.NewGCM(block)
    nonceSize := gcm.NonceSize()
    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }
    return string(plaintext), nil
}
```

**Request/Response Models:**

```go
// internal/api/dto/user_dto.go
package dto

type CreateUserRequest struct {
    Username       string    `json:"username" validate:"required,min=3,max=50"`
    Email          string    `json:"email" validate:"omitempty,email"`
    TrafficLimitGB int64     `json:"traffic_limit_gb"`
    ExpiryDate     *time.Time `json:"expiry_date"`
}

type UserResponse struct {
    ID                  uint      `json:"id"`
    Username            string    `json:"username"`
    Email               string    `json:"email"`
    UUID                string    `json:"uuid"`
    TrafficLimitBytes   int64     `json:"traffic_limit_bytes"`
    TrafficUsedBytes    int64     `json:"traffic_used_bytes"`
    ExpiryDate          *time.Time `json:"expiry_date"`
    IsActive            bool      `json:"is_active"`
    IsOnline            bool      `json:"is_online"`
    CreatedAt           time.Time `json:"created_at"`
    InboundCount        int       `json:"inbound_count"`
}

// Показывается ТОЛЬКО при создании!
type UserCredentialsResponse struct {
    UserResponse
    Password              string `json:"password"`
    Token                 string `json:"token"`
    SubscriptionToken     string `json:"subscription_token"`
    SSHPublicKey          string `json:"ssh_public_key"`
    SSHPrivateKey         string `json:"ssh_private_key"` // расшифрованный
    WireguardPrivateKey   string `json:"wireguard_private_key"`
    WireguardPublicKey    string `json:"wireguard_public_key"`
}
```

**Acceptance Criteria:**
- ✅ Пользователь создается с автогенерацией всех credentials
- ✅ UUID, Token, Username уникальны (проверка на уровне БД и сервиса)
- ✅ SSH private key зашифрован в БД с использованием AES-256-GCM
- ✅ API возвращает все credentials при создании (один раз!)
- ✅ Последующие GET запросы НЕ возвращают приватные ключи
- ✅ Unit тесты для всех CRUD операций
- ✅ Integration тесты для API endpoints

### 1.2.2 Frontend для Users (2 дня)

**Страница:** `/users`

**Компоненты:**

```
src/pages/Users/
├── UsersList.tsx              # Главная страница со списком
├── UserCreateModal.tsx        # Modal для создания пользователя
├── UserDetailsPanel.tsx       # Детальная панель (раскрывается)
├── UserCredentialsDisplay.tsx # Отображение credentials (один раз!)
├── UserInboundsList.tsx       # Список inbound пользователя
└── UserRegenerateModal.tsx    # Подтверждение регенерации
```

**UI Layout - Главная страница:**

```
┌─────────────────────────────────────────────────────────────┐
│ Users Management                        [+ Create User]     │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│ ┌─ Users List (Compact with Expansion) ─────────────────┐  │
│ │                                                         │  │
│ │ ▶ alice_user                                           │  │
│ │   UUID: 12345678-abcd...  │  Used in: 2 inbounds      │  │
│ │   Created: 2026-03-20     │  Traffic: 5.2 GB / 100 GB │  │
│ │                                                         │  │
│ │ ▼ bob_user                                             │  │
│ │   UUID: 87654321-dcba...  │  Used in: 1 inbound       │  │
│ │   Created: 2026-03-19     │  Traffic: 12.5 GB / ∞     │  │
│ │   ┌─ Details ──────────────────────────────────────┐  │  │
│ │   │ Email: bob@example.com                         │  │  │
│ │   │ Status: ● Active  │  Online: ○ Offline         │  │  │
│ │   │ Expires: 2026-12-31                            │  │  │
│ │   │                                                 │  │  │
│ │   │ Credentials:                                   │  │  │
│ │   │ • UUID: 87654321-dcba-4321... [📋 Copy]       │  │  │
│ │   │ • Password: **************** [👁️ Show] [📋]   │  │  │
│ │   │ • Token: ****************** [👁️ Show] [📋]    │  │  │
│ │   │ • SSH Public Key: [View] [📋 Copy]            │  │  │
│ │   │                                                 │  │  │
│ │   │ Inbounds (1):                                  │  │  │
│ │   │ • vmess-443 (VMess, port 443)                 │  │  │
│ │   │                                                 │  │  │
│ │   │ [Edit] [Regenerate Credentials] [Delete]      │  │  │
│ │   └─────────────────────────────────────────────────┘  │  │
│ │                                                         │  │
│ │ ▶ charlie_user                                         │  │
│ │   UUID: abcdef12-3456...  │  Used in: 0 inbounds      │  │
│ │   Created: 2026-03-18     │  Traffic: 0 B / 50 GB     │  │
│ │                                                         │  │
│ └─────────────────────────────────────────────────────────┘  │
│                                                              │
│ Showing 3 of 15 users                    [1] 2 3 4 5 →     │
└─────────────────────────────────────────────────────────────┘
```

**Create User Modal:**

```
┌─ Create New User ─────────────────────────────────────┐
│                                                        │
│ Username: [alice_user___________]                     │
│           ℹ️ 3-50 characters, alphanumeric + _        │
│                                                        │
│ Email: [alice@example.com____] (optional)             │
│                                                        │
│ ▼ Quotas (optional)                                   │
│   Traffic Limit: [100] GB  ☐ Unlimited                │
│   Expiry Date: [2026-12-31] ☐ Never expires           │
│                                                        │
│ ℹ️ All credentials will be auto-generated             │
│    You will see them ONCE after creation!             │
│                                                        │
│                          [Cancel]  [Create User]      │
└────────────────────────────────────────────────────────┘
```

**После создания - Credentials Display (ОДИН РАЗ!):**

```
┌─ User Created Successfully! ──────────────────────────┐
│                                                        │
│ ⚠️ IMPORTANT: Save these credentials now!             │
│    They will NOT be shown again!                      │
│                                                        │
│ Username: alice_user                                  │
│                                                        │
│ UUID:                                                 │
│ 12345678-abcd-1234-5678-123456789abc                  │
│ [📋 Copy]                                             │
│                                                        │
│ Password:                                             │
│ Xy9#mK2$pL4@qR7!                                      │
│ [📋 Copy] [👁️ Show/Hide]                              │
│                                                        │
│ Token:                                                │
│ aGVsbG8gd29ybGQgdGhpcyBpcyBhIHRva2Vu                  │
│ [📋 Copy]                                             │
│                                                        │
│ Subscription Token:                                   │
│ c3Vic2NyaXB0aW9uX3Rva2VuX2hlcmU=                      │
│ [📋 Copy]                                             │
│                                                        │
│ SSH Public Key:                                       │
│ ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC...           │
│ [📋 Copy] [💾 Download .pub]                          │
│                                                        │
│ SSH Private Key:                                      │
│ -----BEGIN OPENSSH PRIVATE KEY-----                   │
│ b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9u...         │
│ [📋 Copy] [💾 Download .pem]                          │
│                                                        │
│ WireGuard Private Key:                                │
│ YNXtAzepDqRv9H52osJVDQnznT5AM11eCK3ESpwSt04=          │
│ [📋 Copy]                                             │
│                                                        │
│ WireGuard Public Key:                                 │
│ Z1XXLsKYkYxuiYjJIkRvtIKFepCYHTgON+GwPq7SOV4=          │
│ [📋 Copy]                                             │
│                                                        │
│ [💾 Download All as JSON] [💾 Download as TXT]        │
│                                                        │
│                                            [Close]     │
└────────────────────────────────────────────────────────┘
```

**Компонент UsersList.tsx:**

```typescript
// src/pages/Users/UsersList.tsx
import { useState } from 'preact/hooks';
import { useUsers } from '@/hooks/useUsers';

export function UsersList() {
  const { users, loading, error } = useUsers();
  const [expandedUserId, setExpandedUserId] = useState<number | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);

  const toggleExpand = (userId: number) => {
    setExpandedUserId(expandedUserId === userId ? null : userId);
  };

  return (
    <div class="container mx-auto p-6">
      <div class="flex justify-between items-center mb-6">
        <h1 class="text-2xl font-bold">Users Management</h1>
        <button 
          onClick={() => setShowCreateModal(true)}
          class="btn btn-primary"
        >
          + Create User
        </button>
      </div>

      <div class="space-y-2">
        {users.map(user => (
          <div key={user.id} class="border rounded-lg">
            <div 
              class="flex items-center justify-between p-4 cursor-pointer hover:bg-gray-50"
              onClick={() => toggleExpand(user.id)}
            >
              <div class="flex items-center space-x-4">
                <span class="text-lg">
                  {expandedUserId === user.id ? '▼' : '▶'}
                </span>
                <div>
                  <div class="font-semibold">{user.username}</div>
                  <div class="text-sm text-gray-600">
                    UUID: {user.uuid.substring(0, 16)}...
                  </div>
                </div>
              </div>
              <div class="text-sm text-gray-600">
                Used in: {user.inbound_count} inbounds
              </div>
            </div>

            {expandedUserId === user.id && (
              <UserDetailsPanel user={user} />
            )}
          </div>
        ))}
      </div>

      {showCreateModal && (
        <UserCreateModal onClose={() => setShowCreateModal(false)} />
      )}
    </div>
  );
}
```

**Acceptance Criteria:**
- ✅ Компактный список с раскрытием деталей при клике
- ✅ Credentials показываются ОДИН РАЗ после создания в modal
- ✅ Возможность копирования каждого credential
- ✅ Возможность скачать все credentials (JSON/TXT)
- ✅ Показ inbound, в которых используется пользователь
- ✅ Кнопка "Regenerate Credentials" с подтверждением и предупреждением
- ✅ Responsive design для мобильных устройств
- ✅ Loading states и error handling

---


### Структура данных

Protocol Schema Registry - это центральный реестр всех поддерживаемых протоколов с их параметрами, требованиями и возможностями авто-генерации.

**Расположение:** `internal/protocol/registry.go`

```go
// internal/protocol/registry.go
package protocol

type ParameterType string

const (
    TypeString  ParameterType = "string"
    TypeInteger ParameterType = "integer"
    TypeBoolean ParameterType = "boolean"
    TypeSelect  ParameterType = "select"
    TypeUUID    ParameterType = "uuid"
    TypeArray   ParameterType = "array"
    TypeObject  ParameterType = "object"
)

type Parameter struct {
    Name         string          `json:"name"`
    Type         ParameterType   `json:"type"`
    Required     bool            `json:"required"`
    Default      interface{}     `json:"default,omitempty"`
    AutoGenerate bool            `json:"auto_generate"`
    AutoGenFunc  string          `json:"auto_gen_func,omitempty"` // "generate_uuid", "generate_password", etc.
    Options      []string        `json:"options,omitempty"`       // для select
    Description  string          `json:"description,omitempty"`
    Example      string          `json:"example,omitempty"`
    DependsOn    []Dependency    `json:"depends_on,omitempty"`
}

type Dependency struct {
    Field     string      `json:"field"`
    Value     interface{} `json:"value"`
    Condition string      `json:"condition"` // "equals", "not_equals", "in", "not_in"
}

type ProtocolSchema struct {
    Protocol    string               `json:"protocol"`
    Core        []string             `json:"core"` // ["sing-box", "xray", "mihomo"]
    Direction   string               `json:"direction"` // "inbound", "outbound", "both"
    RequiresTLS bool                 `json:"requires_tls"`
    Parameters  map[string]Parameter `json:"parameters"`
    Transport   []string             `json:"transport,omitempty"` // ["websocket", "grpc", "http"]
}
```

### Примеры схем протоколов

**VMess:**
```go
"vmess": {
    Protocol:    "vmess",
    Core:        []string{"sing-box", "xray", "mihomo"},
    Direction:   "both",
    RequiresTLS: false,
    Parameters: map[string]Parameter{
        "uuid": {
            Name:         "uuid",
            Type:         TypeUUID,
            Required:     true,
            AutoGenerate: true,
            AutoGenFunc:  "generate_uuid_v4",
            Description:  "User UUID for VMess protocol",
        },
        "alter_id": {
            Name:        "alter_id",
            Type:        TypeInteger,
            Required:    false,
            Default:     0,
            Description: "AlterID (recommended: 0)",
        },
        "cipher": {
            Name:        "cipher",
            Type:        TypeSelect,
            Required:    false,
            Default:     "auto",
            Options:     []string{"auto", "aes-128-gcm", "chacha20-poly1305", "none"},
            Description: "Encryption cipher",
        },
    },
    Transport: []string{"websocket", "grpc", "http", "httpupgrade"},
}
```

**Trojan (требует TLS):**
```go
"trojan": {
    Protocol:    "trojan",
    Core:        []string{"sing-box", "xray", "mihomo"},
    Direction:   "both",
    RequiresTLS: true, // КРИТИЧНО!
    Parameters: map[string]Parameter{
        "password": {
            Name:         "password",
            Type:         TypeString,
            Required:     true,
            AutoGenerate: true,
            AutoGenFunc:  "generate_password_16",
            Description:  "Trojan password",
        },
    },
    Transport: []string{"websocket", "grpc"},
}
```

### Auto-generation функции

**Расположение:** `internal/protocol/generators.go`

```go
// internal/protocol/generators.go
package protocol

import (
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "github.com/google/uuid"
)

// GenerateUUIDv4 генерирует UUID v4
func GenerateUUIDv4() string {
    return uuid.New().String()
}

// GeneratePassword генерирует криптостойкий пароль
func GeneratePassword(length int) string {
    const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
    b := make([]byte, length)
    rand.Read(b)
    for i := range b {
        b[i] = charset[int(b[i])%len(charset)]
    }
    return string(b)
}

// GenerateBase64Token генерирует base64 токен
func GenerateBase64Token(bytes int) string {
    b := make([]byte, bytes)
    rand.Read(b)
    return base64.StdEncoding.EncodeToString(b)
}

// GenerateRandomPath генерирует случайный путь
func GenerateRandomPath(prefix string) string {
    return fmt.Sprintf("/%s_%s", prefix, GeneratePassword(8))
}

// GenerateWireGuardKeypair генерирует пару ключей WireGuard
func GenerateWireGuardKeypair() (privateKey, publicKey string, err error) {
    // Реализация генерации WireGuard ключей
    // Использовать golang.zx2c4.com/wireguard/wgctrl/wgtypes
    return
}

// GenerateSSHKeypair генерирует пару SSH ключей
func GenerateSSHKeypair() (privateKey, publicKey string, err error) {
    // Реализация генерации SSH ключей
    // Использовать golang.org/x/crypto/ssh
    return
}

// GenerateECDSAKeypair генерирует пару ECDSA ключей для MASQUE
func GenerateECDSAKeypair() (privateKey, publicKey string, err error) {
    // Реализация генерации ECDSA ключей
    return
}
```

### API для работы с Registry

```go
// internal/api/handlers/protocol_handler.go
package handlers

// GET /api/protocols - список всех протоколов
func (h *ProtocolHandler) ListProtocols(c *fiber.Ctx) error {
    protocols := protocol.GetAllProtocols()
    return c.JSON(protocols)
}

// GET /api/protocols/:name/schema - схема конкретного протокола
func (h *ProtocolHandler) GetProtocolSchema(c *fiber.Ctx) error {
    name := c.Params("name")
    schema, err := protocol.GetProtocolSchema(name)
    if err != nil {
        return c.Status(404).JSON(fiber.Map{"error": "Protocol not found"})
    }
    return c.JSON(schema)
}

// GET /api/protocols/by-core/:core - протоколы для конкретного ядра
func (h *ProtocolHandler) GetProtocolsByCore(c *fiber.Ctx) error {
    coreName := c.Params("core")
    protocols := protocol.GetProtocolsByCore(coreName)
    return c.JSON(protocols)
}
```

---


1. [Обзор концепции](#обзор-концепции)
2. [Архитектура данных](#архитектура-данных)
3. [Расширение структуры БД](#расширение-структуры-бд)
4. [Protocol Schema Registry](#protocol-schema-registry)
5. [Фаза 1.2: User Management System](#фаза-12-user-management-system)
6. [Фаза 3: Inbound Management с Protocol-Aware Forms](#фаза-3-inbound-management-с-protocol-aware-forms)
7. [Управление пользователями в Inbound](#управление-пользователями-в-inbound)
8. [CLI интерфейс](#cli-интерфейс)
9. [Примеры реализации](#примеры-реализации)

---

## Обзор концепции

### Ключевые принципы

1. **Разделение сущностей**: Users и Inbounds - независимые сущности
2. **Универсальные credentials**: При создании пользователя генерируются ВСЕ типы credentials (UUID, Password, Token, SSH keys)
3. **Гибкая ассоциация**: Пользователи добавляются к inbound после его создания
4. **Protocol-Aware Forms**: Динамические формы адаптируются под выбранный протокол
5. **Auto-generation**: Все сложные параметры генерируются автоматически с возможностью кастомизации

### Workflow

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Создание пользователя (Users Management)                 │
│    - Генерация UUID, Password, Token, SSH keys, WG keys     │
│    - Сохранение в БД                                         │
│    - Показ credentials ОДИН РАЗ                              │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ 2. Создание Inbound (Inbound Management)                    │
│    - Выбор ядра (Sing-box / Xray / Mihomo)                  │
│    - Выбор протокола (фильтруется по ядру)                  │
│    - Настройка параметров (Protocol-Aware Form)             │
│    - Создание БЕЗ пользователей                              │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ 3. Добавление пользователей к Inbound                       │
│    - Открыть детали Inbound                                  │
│    - Кнопка "Add Users"                                      │
│    - Выбрать пользователей из списка                         │
│    - Создать mapping в user_inbound_mapping                  │
│    - Регенерировать конфиг ядра                              │
└─────────────────────────────────────────────────────────────┘
```

### Архитектура данных

```
┌─────────────┐       ┌──────────────────┐       ┌─────────────┐
│   Users     │       │ user_inbound_    │       │  Inbounds   │
│             │◄──────┤    mapping       ├──────►│             │
│ - UUID      │  N:M  │                  │  N:M  │ - Protocol  │
│ - Password  │       │ - user_id        │       │ - Port      │
│ - Token     │       │ - inbound_id     │       │ - Config    │
│ - SSH keys  │       │ - created_at     │       │ - Core      │
│ - WG keys   │       │                  │       │             │
└─────────────┘       └──────────────────┘       └─────────────┘
```

---

## Расширение структуры БД

### Обновление таблицы `users`

**Изменения:**
- Добавлены поля для всех типов credentials
- Добавлено шифрование для приватных ключей
- Добавлены индексы для быстрого поиска

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100),
    
    -- Universal Credentials (генерируются при создании)
    uuid VARCHAR(36) UNIQUE NOT NULL,              -- для VMess, VLESS, TUIC v5
    password VARCHAR(255) NOT NULL,                -- для Trojan, Shadowsocks, SSH
    password_hash VARCHAR(255),                    -- bcrypt hash для безопасности
    token VARCHAR(64) UNIQUE,                      -- для TUIC v4
    ssh_public_key TEXT,                           -- для SSH протокола
    ssh_private_key_encrypted TEXT,                -- зашифрованный приватный ключ
    wireguard_private_key VARCHAR(44),             -- для WireGuard
    wireguard_public_key VARCHAR(44),              -- для WireGuard
    
    -- Квоты
    traffic_limit_bytes BIGINT DEFAULT NULL,       -- NULL = безлимит
    traffic_used_bytes BIGINT DEFAULT 0,
    expiry_date DATETIME DEFAULT NULL,             -- NULL = бессрочно
    
    -- Статус
    is_active BOOLEAN DEFAULT TRUE,
    is_online BOOLEAN DEFAULT FALSE,
    
    -- Метаданные
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_connected_at DATETIME,
    
    -- Subscription
    subscription_token VARCHAR(64) UNIQUE NOT NULL,
    
    -- Связи
    created_by_admin_id INTEGER,
    FOREIGN KEY (created_by_admin_id) REFERENCES admins(id) ON DELETE SET NULL
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_uuid ON users(uuid);
CREATE INDEX idx_users_token ON users(token);
CREATE INDEX idx_users_subscription_token ON users(subscription_token);
CREATE INDEX idx_users_is_active ON users(is_active);
```

### Таблица `user_inbound_mapping` (уже существует)

**Назначение:** Связь многие-ко-многим между пользователями и inbound

```sql
CREATE TABLE user_inbound_mapping (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    inbound_id INTEGER NOT NULL,
    
    -- Метаданные ассоциации
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    added_by_admin_id INTEGER,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (inbound_id) REFERENCES inbounds(id) ON DELETE CASCADE,
    FOREIGN KEY (added_by_admin_id) REFERENCES admins(id) ON DELETE SET NULL,
    
    UNIQUE(user_id, inbound_id)
);

CREATE INDEX idx_user_inbound_user_id ON user_inbound_mapping(user_id);
CREATE INDEX idx_user_inbound_inbound_id ON user_inbound_mapping(inbound_id);
```

### Migration файлы

**Обновить существующую миграцию:**
```
migrations/000002_create_users_table.up.sql  # Добавить новые поля
migrations/000002_create_users_table.down.sql
```

**Уже существует:**
```
migrations/000006_create_user_inbound_mapping_table.up.sql
migrations/000006_create_user_inbound_mapping_table.down.sql
```

---

