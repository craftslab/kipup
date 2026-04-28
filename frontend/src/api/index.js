import axios from 'axios'

const api = axios.create({ baseURL: '/api/v1' })

export const listBuckets = () => api.get('/buckets')

export const createBucket = (name, region = 'us-east-1') =>
  api.post('/buckets', { name, region })

export const deleteBucket = (bucket) => api.delete(`/buckets/${bucket}`)

export const listObjects = (bucket, prefix = '') =>
  api.get(`/objects/${encodeURIComponent(bucket)}`, { params: { prefix } })

/**
 * Returns the URL that triggers a streaming download.
 * Using a plain anchor href lets the browser stream the file
 * without loading it into JS memory.
 */
export const downloadUrl = (bucket, key) =>
  `/api/v1/objects/${encodeURIComponent(bucket)}/${encodeURIComponent(key)}`

/**
 * Uploads files with progress reporting.
 * Files are sent as multipart/form-data; the backend streams them to S3.
 */
export const uploadObjects = (bucket, files, prefix = '', onProgress) => {
  const form = new FormData()
  for (const file of files) {
    form.append('file', file)
  }
  return api.post(`/objects/${encodeURIComponent(bucket)}`, form, {
    params: { prefix },
    headers: { 'Content-Type': 'multipart/form-data' },
    onUploadProgress: onProgress
  })
}

export const deleteObject = (bucket, key) =>
  api.delete(`/objects/${encodeURIComponent(bucket)}/${encodeURIComponent(key)}`)

/**
 * Returns a presigned download URL for a specific object.
 * expirySeconds defaults to 86400 (24 h).
 */
export const generateDownloadLink = (bucket, key, expirySeconds = 86400) =>
  api.get(`/presign/download/${encodeURIComponent(bucket)}/${encodeURIComponent(key)}`, {
    params: { expiry: expirySeconds }
  })

/**
 * Returns a presigned upload URL for a specific key.
 * expirySeconds defaults to 86400 (24 h).
 */
export const generateUploadLink = (bucket, key, expirySeconds = 86400) =>
  api.get(`/presign/upload/${encodeURIComponent(bucket)}/${encodeURIComponent(key)}`, {
    params: { expiry: expirySeconds }
  })
