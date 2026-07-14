import axios from 'axios'

declare module 'axios' {
  export interface AxiosRequestConfig {
    // Opt out of the global 401-redirect interceptor below, for a request
    // whose caller already handles its own 401 as an expected response (a
    // session probe, a login attempt, a logout) rather than a session that
    // died out from under it.
    skipAuthRedirect?: boolean
  }
}

const apiClient = axios.create({
  baseURL: '/api',
  withCredentials: true,
  // Double-submit CSRF protection: the backend pairs the session cookie
  // with a non-HttpOnly wdsplit_csrf cookie the browser can read; axios
  // echoes its value back as X-CSRF-Token on every request.
  xsrfCookieName: 'wdsplit_csrf',
  xsrfHeaderName: 'X-CSRF-Token',
  withXSRFToken: true,
  headers: {
    'Content-Type': 'application/json',
  },
})

apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (
      axios.isAxiosError(error) &&
      error.response?.status === 401 &&
      !error.config?.skipAuthRedirect &&
      !window.location.pathname.endsWith('/login')
    ) {
      window.location.href = '/login'
    }
    return Promise.reject(error)
  },
)

export default apiClient
