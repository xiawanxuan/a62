import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Annotation, SonarFile, Category, Snapshot, OnlineUser } from '@/types'
import { api, categoryApi, getOrCreateUserId, getUserName, setUserName as saveUserName } from '@/utils/api'

export const useAnnotationStore = defineStore('annotation', {
  state: () => ({
    currentFile: ref<SonarFile | null>(null),
    files: ref<SonarFile[]>([]),
    annotations: ref<Annotation[]>([]),
    categories: ref<Category[]>([]),
    snapshots: ref<Snapshot[]>([]),
    onlineUsers: ref<OnlineUser[]>([]),
    selectedAnnotationId: ref<string | null>(null),
    selectedCategoryId: ref<string | null>(null),
    userId: ref<string>(''),
    userName: ref<string>(''),
    isLoading: ref(false),
    isCategoriesLoading: ref(false),
    draftCacheKey: ref<Record<string, Annotation[]>>({})
  }),

  getters: {
    selectedAnnotation: (state) => {
      return state.annotations.find(a => a.id === state.selectedAnnotationId) || null
    },

    selectedCategory: (state) => {
      return state.categories.find(c => c.id === state.selectedCategoryId) || null
    },

    globalCategories: (state) => {
      return state.categories.filter(c => !c.userId)
    },

    userCategories: (state) => {
      return state.categories.filter(c => !!c.userId)
    },

    annotationsByCategory: (state) => {
      const grouped: Record<string, Annotation[]> = {}
      state.annotations.forEach(a => {
        const key = a.categoryId || '__uncategorized__'
        if (!grouped[key]) {
          grouped[key] = []
        }
        grouped[key].push(a)
      })
      return grouped
    },

  actions: {
    initIdentity() {
      if (!this.userId) {
        this.userId = getOrCreateUserId()
      }
      if (!this.userName) {
        this.userName = getUserName()
      }
    },

    updateUserName(name: string) {
      this.userName = name
      saveUserName(name)
    },

    async loadFiles() {
      return new Promise<void>((resolve, reject) => {
        api.get<SonarFile[]>('/files').then(res => {
          this.files = res.data
          resolve()
        }).catch(reject)
      })
    },

    async loadCategories() {
      this.isCategoriesLoading = true
      try {
        const res = await categoryApi.list()
        this.categories = res.data
        if (this.categories.length > 0 && !this.selectedCategoryId) {
          this.selectedCategoryId = this.categories[0].id
        }
      } finally {
        this.isCategoriesLoading = false
      }
    },

    async createCategory(data: { name: string; color: string; description?: string }) {
      const res = await categoryApi.create(data)
      this.categories.push(res.data)
      if (!this.selectedCategoryId) {
        this.selectedCategoryId = res.data.id
      }
      return res.data
    },

    async updateCategory(id: string, data: { name?: string; color?: string; description?: string }) {
      const res = await categoryApi.update(id, data)
      const index = this.categories.findIndex(c => c.id === id)
      if (index !== -1) {
        this.categories[index] = res.data
      }
      return res.data
    },

    async deleteCategory(id: string) {
      await categoryApi.remove(id)
      this.categories = this.categories.filter(c => c.id !== id)
      this.annotations.forEach(a => {
        if (a.categoryId === id) {
          a.categoryId = null
        }
      })
      if (this.selectedCategoryId === id) {
        this.selectedCategoryId = this.categories.length > 0 ? this.categories[0].id : null
      }
    },

    setSelectedCategory(id: string | null) {
      this.selectedCategoryId = id
    },

    async selectFile(fileId: string) {
      this.isLoading = true
      try {
        const [fileRes, annRes, snapRes] = await Promise.all([
          api.get<SonarFile>(`/files/${fileId}`),
          api.get<Annotation[]>(`/annotations/file/${fileId}`),
          api.get<Snapshot[]>(`/snapshots/file/${fileId}`)
        ])

        this.currentFile = fileRes.data
        this.annotations = annRes.data
        this.snapshots = snapRes.data
        this.selectedAnnotationId = null

        const cached = this.draftCacheKey[fileId]
        if (cached && cached.length > this.annotations.length) {
          console.log('Found local draft cache')
        }
      } finally {
        this.isLoading = false
      }
    },

    async createAnnotation(annotation: Omit<Annotation, 'id' | 'createdAt' | 'updatedAt'>) {
      if (!this.currentFile) return null

      const ann = {
        ...annotation,
        fileId: this.currentFile.id,
        createdBy: this.userId || annotation.createdBy
      }
      const res = await api.post<Annotation>('/annotations', ann)

      const newAnn = res.data
      this.annotations.push(newAnn)

      this.saveDraftCache()

      return newAnn
    },

    async updateAnnotation(id: string, updates: Partial<Annotation>) {
      const res = await api.put<Annotation>(`/annotations/${id}`, updates)
      const index = this.annotations.findIndex(a => a.id === id)
      if (index !== -1) {
        this.annotations[index] = res.data
      }
      this.saveDraftCache()
      return res.data
    },

    async deleteAnnotation(id: string) {
      await api.delete(`/annotations/${id}`)
      this.annotations = this.annotations.filter(a => a.id !== id)
      if (this.selectedAnnotationId === id) {
        this.selectedAnnotationId = null
      }
      this.saveDraftCache()
    },

    async restoreSnapshot(snapshotId: string) {
      const res = await api.post(`/snapshots/restore/${snapshotId}`)
      if (res.data && res.data.annotations) {
        this.annotations = res.data.annotations
      }
      this.saveDraftCache()
    },

    addAnnotationFromWS(annotation: Annotation) {
      const exists = this.annotations.find(a => a.id === annotation.id)
      if (!exists) {
        this.annotations.push(annotation)
        this.saveDraftCache()
      }
    },

    updateAnnotationFromWS(annotation: Annotation) {
      const index = this.annotations.findIndex(a => a.id === annotation.id)
      if (index !== -1) {
        this.annotations[index] = annotation
        this.saveDraftCache()
      }
    },

    deleteAnnotationFromWS(id: string) {
      this.annotations = this.annotations.filter(a => a.id !== id)
      if (this.selectedAnnotationId === id) {
        this.selectedAnnotationId = null
      }
      this.saveDraftCache()
    },

    saveDraftCache() {
      if (this.currentFile) {
        this.draftCacheKey[this.currentFile.id] = [...this.annotations]
      }
    },

    clearCurrentFile() {
      this.currentFile = null
      this.annotations = []
      this.snapshots = []
      this.selectedAnnotationId = null
    },

    setSelectedAnnotation(id: string | null) {
      this.selectedAnnotationId = id
    },

    addOnlineUser(user: OnlineUser) {
      const exists = this.onlineUsers.find(u => u.id === user.id)
      if (!exists) {
        this.onlineUsers.push(user)
      }
    },

    removeOnlineUser(userId: string) {
      this.onlineUsers = this.onlineUsers.filter(u => u.id !== userId)
    },

    updateUserCursor(userId: string, cursor: { x: number; y: number }) {
      const user = this.onlineUsers.find(u => u.id === userId)
      if (user) {
        user.cursor = cursor
      }
    }
  },

  persist: {
    key: 'sonar-annotation-drafts',
    paths: ['draftCacheKey', 'userId', 'userName']
  }
})
