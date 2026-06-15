export interface Point {
  x: number
  y: number
}

export type AnnotationType = 'rectangle' | 'polygon'

export interface Annotation {
  id: string
  fileId: string
  type: AnnotationType
  points: Point[]
  categoryId: string
  label: string
  color: string
  createdAt: number
  updatedAt: number
  createdBy: string
  confidence?: number
}

export interface SonarFile {
  id: string
  name: string
  path: string
  width: number
  height: number
  size: number
  createdAt: number
  annotationCount: number
}

export interface Category {
  id: string
  name: string
  color: string
  description: string
}

export interface Snapshot {
  id: string
  fileId: string
  annotations: Annotation[]
  createdAt: number
  createdBy: string
  message: string
}

export interface WSMessage {
  type: 'annotation-create' | 'annotation-update' | 'annotation-delete' | 'cursor-move' | 'user-join' | 'user-leave'
  payload: any
  userId: string
  timestamp: number
}

export interface OnlineUser {
  id: string
  name: string
  color: string
  cursor: Point | null
}

export interface ViewTransform {
  scale: number
  offsetX: number
  offsetY: number
}

export interface DraftAnnotation {
  type: AnnotationType | null
  points: Point[]
  categoryId: string | null
}

export interface FileNode {
  id: string
  name: string
  type: 'folder' | 'file'
  children?: FileNode[]
  fileInfo?: SonarFile
}
