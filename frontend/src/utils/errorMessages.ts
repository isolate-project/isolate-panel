/**
 * Error messages for the application
 * Provides human-readable error messages in multiple languages
 */

interface ErrorMessage {
  en: string;
  ru: string;
  zh: string;
}

interface ErrorMessages {
  [key: string]: ErrorMessage;
}

export const errorMessages: ErrorMessages = {
  // Authentication errors
  'auth.invalid_credentials': {
    en: 'Invalid username or password',
    ru: 'Неверное имя пользователя или пароль',
    zh: '用户名或密码无效',
  },
  'auth.token_expired': {
    en: 'Session expired. Please login again.',
    ru: 'Сессия истекла. Пожалуйста, войдите снова.',
    zh: '会话已过期。请重新登录。',
  },
  'auth.unauthorized': {
    en: 'Unauthorized access',
    ru: 'Несанкционированный доступ',
    zh: '未经授权的访问',
  },
  'auth.forbidden': {
    en: 'You do not have permission to perform this action',
    ru: 'У вас нет прав для выполнения этого действия',
    zh: '您没有执行此操作的权限',
  },
  
  // User errors
  'user.not_found': {
    en: 'User not found',
    ru: 'Пользователь не найден',
    zh: '用户不存在',
  },
  'user.already_exists': {
    en: 'User with this username already exists',
    ru: 'Пользователь с таким именем уже существует',
    zh: '该用户名已存在',
  },
  'user.create_failed': {
    en: 'Failed to create user',
    ru: 'Не удалось создать пользователя',
    zh: '创建用户失败',
  },
  'user.update_failed': {
    en: 'Failed to update user',
    ru: 'Не удалось обновить пользователя',
    zh: '更新用户失败',
  },
  'user.delete_failed': {
    en: 'Failed to delete user',
    ru: 'Не удалось удалить пользователя',
    zh: '删除用户失败',
  },
  
  // Inbound errors
  'inbound.not_found': {
    en: 'Inbound not found',
    ru: 'Входящее подключение не найдено',
    zh: '入站连接不存在',
  },
  'inbound.create_failed': {
    en: 'Failed to create inbound',
    ru: 'Не удалось создать входящее подключение',
    zh: '创建入站连接失败',
  },
  'inbound.update_failed': {
    en: 'Failed to update inbound',
    ru: 'Не удалось обновить входящее подключение',
    zh: '更新入站连接失败',
  },
  'inbound.delete_failed': {
    en: 'Failed to delete inbound',
    ru: 'Не удалось удалить входящее подключение',
    zh: '删除入站连接失败',
  },
  
  // Core errors
  'core.not_found': {
    en: 'Core not found',
    ru: 'Ядро не найдено',
    zh: '核心不存在',
  },
  'core.start_failed': {
    en: 'Failed to start core',
    ru: 'Не удалось запустить ядро',
    zh: '启动核心失败',
  },
  'core.stop_failed': {
    en: 'Failed to stop core',
    ru: 'Не удалось остановить ядро',
    zh: '停止核心失败',
  },
  'core.restart_failed': {
    en: 'Failed to restart core',
    ru: 'Не удалось перезапустить ядро',
    zh: '重启核心失败',
  },
  
  // Network errors
  'network.error': {
    en: 'Network error. Please check your connection.',
    ru: 'Ошибка сети. Пожалуйста, проверьте подключение.',
    zh: '网络错误。请检查您的连接。',
  },
  'network.timeout': {
    en: 'Request timeout',
    ru: 'Время запроса истекло',
    zh: '请求超时',
  },
  
  // Validation errors
  'validation.required': {
    en: 'This field is required',
    ru: 'Это поле обязательно для заполнения',
    zh: '此字段为必填项',
  },
  'validation.invalid_email': {
    en: 'Invalid email address',
    ru: 'Неверный адрес электронной почты',
    zh: '无效的电子邮件地址',
  },
  'validation.invalid_format': {
    en: 'Invalid format',
    ru: 'Неверный формат',
    zh: '格式无效',
  },
  'validation.too_short': {
    en: 'Value is too short',
    ru: 'Значение слишком короткое',
    zh: '值太短',
  },
  'validation.too_long': {
    en: 'Value is too long',
    ru: 'Значение слишком длинное',
    zh: '值太长',
  },
  
  // System errors
  'system.error': {
    en: 'An unexpected error occurred',
    ru: 'Произошла непредвиденная ошибка',
    zh: '发生意外错误',
  },
  'unknown_error': {
    en: 'Unknown error occurred',
    ru: 'Произошла неизвестная ошибка',
    zh: '发生未知错误',
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
    return errorMessages['unknown_error'][lang as keyof ErrorMessage] || errorMessages['unknown_error']['en'];
  }
  
  return error[lang as keyof ErrorMessage] || error['en'] || errorMessages['unknown_error']['en'];
}

/**
 * Get all error codes
 * @returns Array of error code keys
 */
export function getErrorCodes(): string[] {
  return Object.keys(errorMessages);
}
