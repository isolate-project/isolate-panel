/**
 * Human-readable error messages for the application
 * Supports English, Russian, and Chinese
 */

export interface ErrorMessages {
  [key: string]: {
    en: string;
    ru: string;
    zh: string;
  };
}

export const errorMessages: ErrorMessages = {
  // Authentication errors
  'invalid_credentials': {
    en: 'Invalid username or password. Please try again.',
    ru: 'Неверное имя пользователя или пароль. Попробуйте снова.',
    zh: '用户名或密码无效。请重试。',
  },
  'token_expired': {
    en: 'Your session has expired. Please log in again.',
    ru: 'Ваша сессия истекла. Пожалуйста, войдите снова.',
    zh: '您的会话已过期。请重新登录。',
  },
  'unauthorized': {
    en: 'You do not have permission to access this resource.',
    ru: 'У вас нет доступа к этому ресурсу.',
    zh: '您无权访问此资源。',
  },
  
  // User errors
  'user_not_found': {
    en: 'User not found. Please check the username.',
    ru: 'Пользователь не найден. Проверьте имя пользователя.',
    zh: '未找到用户。请检查用户名。',
  },
  'user_already_exists': {
    en: 'A user with this username already exists.',
    ru: 'Пользователь с таким именем уже существует.',
    zh: '该用户名已存在。',
  },
  'username_required': {
    en: 'Username is required.',
    ru: 'Имя пользователя обязательно.',
    zh: '用户名为必填项。',
  },
  'username_too_long': {
    en: 'Username is too long (max 50 characters).',
    ru: 'Имя пользователя слишком длинное (максимум 50 символов).',
    zh: '用户名过长（最多 50 个字符）。',
  },
  'invalid_email': {
    en: 'Please enter a valid email address.',
    ru: 'Пожалуйста, введите корректный email адрес.',
    zh: '请输入有效的电子邮件地址。',
  },
  'email_required': {
    en: 'Email address is required.',
    ru: 'Email адрес обязателен.',
    zh: '电子邮件为必填项。',
  },
  
  // Inbound errors
  'inbound_not_found': {
    en: 'Inbound not found. It may have been deleted.',
    ru: 'Inbound не найден. Возможно, он был удален.',
    zh: '未找到入站配置。可能已被删除。',
  },
  'port_in_use': {
    en: 'This port is already in use. Please choose another port.',
    ru: 'Этот порт уже используется. Пожалуйста, выберите другой порт.',
    zh: '该端口已被使用。请选择其他端口。',
  },
  'invalid_port': {
    en: 'Port must be between 1 and 65535.',
    ru: 'Порт должен быть от 1 до 65535.',
    zh: '端口必须在 1 到 65535 之间。',
  },
  'protocol_required': {
    en: 'Protocol is required.',
    ru: 'Протокол обязателен.',
    zh: '协议为必填项。',
  },
  
  // Settings errors
  'setting_not_found': {
    en: 'Setting not found. Please check the key.',
    ru: 'Настройка не найдена. Проверьте ключ.',
    zh: '未找到设置。请检查键。',
  },
  'invalid_setting_value': {
    en: 'Invalid setting value. Please check the format.',
    ru: 'Неверное значение настройки. Проверьте формат.',
    zh: '设置值无效。请检查格式。',
  },
  
  // Subscription errors
  'subscription_not_found': {
    en: 'Subscription not found. Please contact administrator.',
    ru: 'Подписка не найдена. Пожалуйста, обратитесь к администратору.',
    zh: '未找到订阅。请联系管理员。',
  },
  'subscription_expired': {
    en: 'Your subscription has expired.',
    ru: 'Ваша подписка истекла.',
    zh: '您的订阅已过期。',
  },
  'quota_exceeded': {
    en: 'Traffic quota exceeded. Contact administrator to increase limit.',
    ru: 'Превышен лимит трафика. Обратитесь к администратору для увеличения лимита.',
    zh: '流量配额已用尽。联系管理员增加限制。',
  },
  
  // Network errors
  'network_error': {
    en: 'Connection failed. Please check your internet connection.',
    ru: 'Ошибка подключения. Пожалуйста, проверьте интернет-соединение.',
    zh: '连接失败。请检查您的网络连接。',
  },
  'server_error': {
    en: 'Server error. Please try again later.',
    ru: 'Ошибка сервера. Пожалуйста, попробуйте позже.',
    zh: '服务器错误。请稍后重试。',
  },
  'timeout': {
    en: 'Request timed out. Please try again.',
    ru: 'Время запроса истекло. Пожалуйста, попробуйте снова.',
    zh: '请求超时。请重试。',
  },
  
  // Validation errors
  'validation_error': {
    en: 'Please check your input and try again.',
    ru: 'Пожалуйста, проверьте введенные данные и попробуйте снова.',
    zh: '请检查您的输入并重试。',
  },
  'required_field': {
    en: 'This field is required.',
    ru: 'Это поле обязательно для заполнения.',
    zh: '此字段为必填项。',
  },
  'invalid_format': {
    en: 'Invalid format. Please check the expected format.',
    ru: 'Неверный формат. Пожалуйста, проверьте ожидаемый формат.',
    zh: '格式无效。请检查预期格式。',
  },
  
  // Core errors
  'core_not_running': {
    en: 'Core is not running. Please start it first.',
    ru: 'Ядро не запущено. Пожалуйста, запустите его сначала.',
    zh: '核心未运行。请先启动它。',
  },
  'core_config_error': {
    en: 'Core configuration error. Please check the logs.',
    ru: 'Ошибка конфигурации ядра. Пожалуйста, проверьте логи.',
    zh: '核心配置错误。请检查日志。',
  },
  
  // Backup errors
  'backup_failed': {
    en: 'Backup failed. Please check the logs.',
    ru: 'Ошибка создания резервной копии. Пожалуйста, проверьте логи.',
    zh: '备份失败。请检查日志。',
  },
  'restore_failed': {
    en: 'Restore failed. Please check the backup file.',
    ru: 'Ошибка восстановления. Пожалуйста, проверьте файл резервной копии.',
    zh: '恢复失败。请检查备份文件。',
  },
  
  // Generic errors
  'unknown_error': {
    en: 'An unknown error occurred. Please try again.',
    ru: 'Произошла неизвестная ошибка. Пожалуйста, попробуйте снова.',
    zh: '发生未知错误。请重试。',
  },
  'operation_failed': {
    en: 'Operation failed. Please try again.',
    ru: 'Операция не удалась. Пожалуйста, попробуйте снова.',
    zh: '操作失败。请重试。',
  },
  'permission_denied': {
    en: 'Permission denied. You do not have access to this resource.',
    ru: 'Отказано в доступе. У вас нет доступа к этому ресурсу.',
    zh: '权限被拒绝。您无权访问此资源。',
  },
};

/**
 * Get error message in the specified language
 * @param errorCode - The error code key
 * @param lang - Language code ('en', 'ru', 'zh')
 * @returns Human-readable error message
 */
export function getErrorMessage(errorCode: string, lang: string = 'en'): string {
  const error = errorMessages[errorCode];
  
  if (!error) {
    return errorMessages['unknown_error'][lang] || errorMessages['unknown_error']['en'];
  }
  
  return error[lang] || error['en'] || errorMessages['unknown_error']['en'];
}

/**
 * Get all error codes
 * @returns Array of error code keys
 */
export function getErrorCodes(): string[] {
  return Object.keys(errorMessages);
}
