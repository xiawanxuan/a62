import axios, { type AxiosResponse } from 'axios'
import type { Category } from '@/types'

const USER_ID_KEY = 'sonar-user-id'
const USER_NAME_KEY = 'sonar-user-name'

const getOrCreateUserId = (): string => {
  let uid = localStorage.getItem(USER_ID_KEY)
  if (!uid) {
    uid = 'user_' + Math.random().toString(36).substr(2, 9) + Date.now().toString(36)
    localStorage.setItem(USER_ID_KEY, uid)
  }
  return uid
}

const getUserName = (): string => {
  return localStorage.getItem(USER_NAME_KEY) || '标注员'
}

const setUserName = (name: string) => {
  localStorage.setItem(USER_NAME_KEY, name)
}

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

api.interceptors.request.use((config) => {
  const uid = getOrCreateUserId()
  if (uid) {
    config.headers['X-User-Id'] = uid
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    console.error('API Error:', error)
    return Promise.reject(error)
  }
)

const categoryApi = {
  list: () => api.get<Category[]>('/categories'),

  create: (data: { name: string; color: string; description?: string }) =>
    api.post<Category>('/categories', data),

  update: (id: string, data: { name?: string; color?: string; description?: string }) =>
    api.put<Category>(`/categories/${id}`, data),

  remove: (id: string) => api.delete(`/categories/${id}`)
}

export {
  api,
  categoryApi,
  getOrCreateUserId,
  getUserName,
  setUserName
}

export type ApiResponse<T> = AxiosResponse<T>
