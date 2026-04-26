type ErrorCategory = 'validation' | 'network' | 'server' | 'auth' | 'unknown';

interface SanitizedError {
  message: string;
  category: ErrorCategory;
}

function sanitizeError(error: unknown): SanitizedError {
  // Default safe message
  const defaultMsg = 'An unexpected error occurred. Please try again.';

  if (!error) return { message: defaultMsg, category: 'unknown' };

  if (error instanceof Error) {
    // Check for axios error patterns
    const axiosErr = error as any;

    // Network errors (no response)
    if (axiosErr.code === 'ERR_NETWORK' || axiosErr.message?.includes('Network Error')) {
      return { message: 'Network error. Please check your connection.', category: 'network' };
    }

    // Auth errors (401, 403)
    if (axiosErr.response?.status === 401) {
      return { message: 'Authentication required. Please log in again.', category: 'auth' };
    }
    if (axiosErr.response?.status === 403) {
      return { message: 'You do not have permission to perform this action.', category: 'auth' };
    }

    // Validation errors (400, 422) — backend validation messages are generally safe
    if (axiosErr.response?.status === 400 || axiosErr.response?.status === 422) {
      const backendMsg = axiosErr.response?.data?.error;
      if (typeof backendMsg === 'string' && backendMsg.length < 200) {
        return { message: backendMsg, category: 'validation' };
      }
      return { message: 'Invalid request. Please check your input.', category: 'validation' };
    }

    // Rate limit (429)
    if (axiosErr.response?.status === 429) {
      return { message: 'Too many requests. Please wait a moment.', category: 'network' };
    }

    // Server errors (5xx)
    if (axiosErr.response?.status >= 500) {
      return { message: 'Server error. Please try again later.', category: 'server' };
    }

    // Timeout
    if (axiosErr.code === 'ECONNABORTED' || axiosErr.message?.includes('timeout')) {
      return { message: 'Request timed out. Please try again.', category: 'network' };
    }

    // Default — NEVER expose error.message directly
    return { message: defaultMsg, category: 'unknown' };
  }

  return { message: defaultMsg, category: 'unknown' };
}

export { sanitizeError, type SanitizedError, type ErrorCategory };