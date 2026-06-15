import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Annotation, SonarFile, Category, Snapshot, OnlineUser } from '@/types'
import { api } from '@/utils/api'

export const useAnnotationStore = defineStore('annotation', {
  state: () => ({
    currentFile: ref<SonarFile | null>(null),
    files: ref<SonarFile[]>([]),
    annotations: ref<Annotation[]>([]),
    categories: ref<Category[]>([]),
    snapshots: ref<Snapshot[]>([]),
    onlineUsers: ref<OnlineUser[]>([]),
    selectedAnnotationId: ref<string | null>(null),
    userId: ref<string>('user_' + Math.random().toString(36).substr(2, 9)),
    userName: ref<string>('标注员'),
    isLoading: ref(false),
    draftCacheKey: ref<Record<string, Annotation[]>>({})
  }),

  getters: {
    selectedAnnotation: (state) => {
      return state.annotations.find(a => a.id === state.selectedAnnotationId) || null
    },

    annotationsByCategory: (state) => {
      const grouped: Record<string, Annotation[]> = {}
      state.annotations.forEach(a => {
        if (!grouped[a.categoryId]) {
          grouped[a.categoryId] = []
        }
        grouped[a.categoryId].push(a)
      })
      return grouped
    }
  },

  actions: {
    async loadFiles() {
      return new Promise<void>((resolve, reject) => {
        api.get<SonarFile[]>('/files').then(res => {
          this.files = res.data
          resolve()
        }).catch(reject)
      })
    },

    async loadCategories() {
      return new Promise<void>((resolve, reject) => {
        api.get<Category[]>('/categories').then(res => {
          this.categories = res.data
          resolve()
        }).catch(reject)
      })
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

      const ann = { ...annotation, fileId: this.currentFile.id }
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
