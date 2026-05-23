// =======================================================
// API ERROR INTERFACES
// =======================================================


export interface BackendErrorBody {
  code: number
  message: string
  fields?: Record<string, string> | null
}

export interface BackendErrorResponse {
  success: false
  error: BackendErrorBody
}

export interface BackendSuccessResponse<T> {
  success: true
  data?: T
  message?: string
  [key: string]: unknown
}

export type BackendResponse<T> = BackendSuccessResponse<T> | BackendErrorResponse

export class ApiError extends Error {
  readonly status: number
  readonly code: number
  readonly fields?: Record<string, string> | null

  constructor(
    message: string,
    status: number,
    code: number,
    fields?: Record<string, string> | null,
  ) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.fields = fields
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, ApiError)
    }
  }

  isUnauthorized() { return this.status === 401 }
  isForbidden() { return this.status === 403 }
  isNotFound() { return this.status === 404 }
  isConflict() { return this.status === 409 }
  isValidation() { return this.status === 400 && !!this.fields }
  isRateLimited() { return this.status === 429 }
  isServerError() { return this.status >= 500 }
  isNetworkError() { return this.status === 0 }
}