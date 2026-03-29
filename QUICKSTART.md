# Isolate Panel - Быстрый старт 🚀

## 1. Подготовка

```bash
cd /mnt/Games/syncthing-shared-folder/isolate-panel
cp docker/.env.example docker/.env
```

Откройте `docker/.env` и установите:
```bash
JWT_SECRET=x7K9mN2pQ5vR8wY3zA6bC0dE1fG4hJ7kL9nM2sT5uV8xA3yB6cD9eF2gH5jK8mN1pQ4r
ADMIN_USERNAME=admin
ADMIN_PASSWORD=admin123  # Измените на сложный пароль!
```

## 2. Сборка и запуск Docker

```bash
docker compose -f docker/docker-compose.yml up -d --build
```

**Первый запуск:**
- Сборка образа: ~3-5 минут
- Загрузка ядер: ~1-2 минуты (зависит от интернета)
- **Итого:** 5-7 минут

**Последующие запуски:**
- Ядра уже скачаны и сохраняются в `./data/cores/`
- Запуск: ~10 секунд

## 3. Доступ к панели

Откройте в браузере: **http://localhost:8080**

- **Логин:** `admin`
- **Пароль:** `admin123` (или ваш из .env)

## 4. Проверка загрузки ядер

```bash
# Посмотреть логи загрузки
docker logs -f isolate-panel

# Проверить наличие ядер
docker exec isolate-panel ls -la /app/data/cores/

# Статус ядер в Supervisor
docker exec isolate-panel supervisorctl status
```

Ожидаемый вывод:
```
✅ Xray core already exists
✅ Mihomo core already exists
✅ Sing-box core already exists
```

## 5. Первое использование

1. **Запустите ядра:**
   - Перейдите в раздел **Cores**
   - Нажмите ▶️ Start для каждого ядра

2. **Создайте пользователя:**
   - Перейдите в **Users** → **Add User**
   - Введите имя и лимиты

3. **Создайте подключение:**
   - Перейдите в **Inbounds** → **Add Inbound**
   - Выберите протокол (VLESS, VMess, Trojan)
   - Настройте порт и TLS

4. **Подключите клиент:**
   - Скопируйте ссылку из карточки подключения
   - Вставьте в ваш прокси-клиент (v2rayN, Clash, etc.)

## 🔧 Troubleshooting

### Ядра не загружаются
```bash
# Проверить логи
docker logs isolate-panel | grep "Downloading"

# Попробовать вручную
docker exec -it isolate-panel /bin/sh
cd /app/data/cores/xray
wget https://github.com/XTLS/Xray-core/releases/download/v26.2.6/Xray-linux-64.zip
```

### Ошибка "Connection refused"
```bash
# Проверить, запущен ли контейнер
docker ps | grep isolate-panel

# Перезапустить
docker compose restart
```

### Ядра не запускаются через панель
```bash
# Проверить логи Supervisor
docker exec isolate-panel supervisorctl status

# Перезапустить Supervisor
docker exec isolate-panel supervisorctl restart all
```

### Посмотреть все логи
```bash
# Логи панели
docker logs -f isolate-panel

# Логи Supervisor
docker exec isolate-panel tail -f /var/log/supervisor/supervisord.log

# Логи ядер
docker exec isolate-panel tail -f /var/log/isolate-panel/xray.out.log
```

## 📱 Мобильный доступ

Для доступа с телефона/VPS:

1. **Откройте порт в фаерволе:**
   ```bash
   ufw allow 8080/tcp
   ```

2. **Используйте HTTPS (рекомендуется):**
   - Настройте Nginx/Caddy как reverse proxy
   - Получите Let's Encrypt сертификат

3. **Или SSH туннель (безопасно):**
   ```bash
   ssh -L 8080:localhost:8080 user@your-vps
   # Затем откройте http://localhost:8080
   ```

## 📦 Структура данных

```
isolate-panel/
└── data/                    # Persistent data (сохраняется)
    ├── isolate-panel.db     # База данных
    └── cores/
        ├── xray/
        │   ├── xray         # Бинарник
        │   └── config.json  # Конфиг
        ├── mihomo/
        │   ├── mihomo
        │   └── config.yaml
        └── singbox/
            ├── sing-box
            └── config.json
```

---

**Готово!** Наслаждайтесь новым UI! 🎨

При возникновении проблем смотрите `docs/DEPLOYMENT.md` или `README.UI.MD`.
