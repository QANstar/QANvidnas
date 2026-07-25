import axios, { AxiosError } from 'axios';

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
});

// Request interceptor: attach JWT token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Response interceptor: handle 401
api.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    if (error.response?.status === 401) {
      // Only redirect if not already on auth pages
      const path = window.location.pathname;
      if (!path.includes('/login') && !path.includes('/register') && !path.includes('/setup')) {
        localStorage.removeItem('token');
        window.location.href = '/login';
      }
    }
    return Promise.reject(error);
  },
);

// Auth API
export const authAPI = {
  checkSetup: () => api.get('/auth/check-setup'),
  setup: (username: string, password: string) =>
    api.post('/auth/setup', { username, password }),
  login: (username: string, password: string) =>
    api.post('/auth/login', { username, password }),
  register: (inviteCode: string, username: string, password: string) =>
    api.post('/auth/register', { invite_code: inviteCode, username, password }),
  me: () => api.get('/auth/me'),
  generateDeviceCode: () => api.get('/auth/device-code'),
  pollDeviceCode: (code: string) => api.get(`/auth/device-code/poll?code=${code}`),
  authorizeDeviceCode: (code: string) => api.post('/auth/device-code/authorize', { code }),
  changePassword: (oldPassword: string, newPassword: string) =>
    api.post('/auth/change-password', { old_password: oldPassword, new_password: newPassword }),
};

// Media API
export const mediaAPI = {
  list: (params?: Record<string, string>) => api.get('/media', { params }),
  get: (id: number) => api.get(`/media/${id}`),
  update: (id: number, data: { title?: string; description?: string; tag_ids?: number[] }) =>
    api.put(`/media/${id}`, data),
  uploadCover: (id: number, file: File) => {
    const form = new FormData();
    form.append('cover', file);
    return api.post(`/media/${id}/cover`, form);
  },
  deleteCover: (id: number) => api.delete(`/media/${id}/cover`),
  getSprite: (id: number) => api.get(`/media/${id}/sprite`),
  scan: () => api.post('/media/scan'),
  scanProgress: () => api.get('/media/scan/progress'),
};

// Search API
export const searchAPI = {
  search: (params: { q?: string; tags?: string; type?: string }) =>
    api.get('/search', { params }),
  getTags: () => api.get('/tags'),
  createTag: (name: string, color?: string) => api.post('/tags', { name, color }),
  deleteTag: (id: number) => api.delete(`/tags/${id}`),
};

// Playlist API
export const playlistAPI = {
  list: () => api.get('/playlists'),
  get: (id: number) => api.get(`/playlists/${id}`),
  create: (name: string, folderPath: string) =>
    api.post('/playlists', { name, folder_path: folderPath }),
  update: (id: number, playMode: string) =>
    api.put(`/playlists/${id}`, { play_mode: playMode }),
  delete: (id: number) => api.delete(`/playlists/${id}`),
  playNow: (folderPath: string) =>
    api.post('/playlists/play-now', { folder_path: folderPath }),
};

// Upload API
export const uploadAPI = {
  upload: (file: File, targetDir: string, onProgress?: (pct: number) => void) => {
    const form = new FormData();
    form.append('file', file);
    form.append('target_dir', targetDir);
    return api.post('/upload', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: (e) => {
        if (e.total && onProgress) {
          onProgress(Math.round((e.loaded / e.total) * 100));
        }
      },
    });
  },
};

// Admin API
export const adminAPI = {
  getInviteCodes: () => api.get('/admin/invite-codes'),
  createInviteCode: (data: { code: string; description?: string; max_uses?: number; expires_at?: string }) =>
    api.post('/admin/invite-codes', data),
  deleteInviteCode: (code: string) => api.delete(`/admin/invite-codes/${code}`),
  getScanFolders: () => api.get('/admin/scan-folders'),
  addScanFolder: (path: string) => api.post('/admin/scan-folders', { path }),
  deleteScanFolder: (id: number) => api.delete(`/admin/scan-folders/${id}`),
  getConfig: () => api.get('/admin/config'),
};

export default api;
