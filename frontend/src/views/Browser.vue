<template>
  <div class="browser-layout">
    <!-- ===== Sidebar – bucket list ===== -->
    <aside class="sidebar">
      <div class="sidebar-header">
        <span class="sidebar-title">Buckets</span>
        <el-button
          circle
          type="primary"
          :icon="Plus"
          size="small"
          title="Create bucket"
          @click="openCreateBucket"
        />
      </div>

      <el-scrollbar class="sidebar-scroll">
        <ul class="bucket-list">
          <li
            v-for="b in buckets"
            :key="b.name"
            class="bucket-item"
            :class="{ active: b.name === currentBucket }"
            @click="selectBucket(b.name)"
          >
            <el-icon><Coin /></el-icon>
            <span class="bucket-name">{{ b.name }}</span>
          </li>
        </ul>
        <el-empty v-if="!buckets.length" description="No buckets" :image-size="60" />
      </el-scrollbar>
    </aside>

    <!-- ===== Main content area ===== -->
    <div class="main-area">
      <!-- Toolbar -->
      <div class="toolbar">
        <!-- Breadcrumb navigation -->
        <el-breadcrumb separator="/" class="breadcrumb">
          <el-breadcrumb-item>
            <span class="breadcrumb-link" @click="goBucketRoot">Home</span>
          </el-breadcrumb-item>
          <el-breadcrumb-item v-if="currentBucket">
            <span class="breadcrumb-link" @click="goBucketRoot">{{ currentBucket }}</span>
          </el-breadcrumb-item>
          <el-breadcrumb-item v-for="(part, i) in prefixParts" :key="i">
            <span class="breadcrumb-link" @click="navigateToDepth(i)">{{ part }}</span>
          </el-breadcrumb-item>
        </el-breadcrumb>

        <div class="toolbar-actions">
          <el-button
            v-if="currentBucket"
            type="primary"
            :icon="UploadFilled"
            @click="showUploadDialog = true"
          >Upload</el-button>
          <el-button
            v-if="currentBucket"
            :icon="Share"
            @click="openUploadLinkDialog"
          >Upload Link</el-button>
          <el-button
            v-if="currentBucket && !currentPrefix"
            type="danger"
            :icon="Delete"
            plain
            @click="confirmDeleteBucket"
          >Delete Bucket</el-button>
        </div>
      </div>

      <!-- Object table -->
      <el-table
        v-loading="loading"
        :data="objects"
        style="width: 100%"
        height="calc(100vh - 120px)"
        empty-text="No objects – select a bucket or upload files"
      >
        <!-- Name column -->
        <el-table-column label="Name" min-width="320" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="file-row" @click="handleRowClick(row)">
              <el-icon class="file-icon" :color="row.isDir ? '#e6a23c' : '#909399'">
                <Folder v-if="row.isDir" />
                <Document v-else />
              </el-icon>
              <span :class="row.isDir ? 'folder-name' : ''">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>

        <!-- Size column -->
        <el-table-column label="Size" width="110" align="right">
          <template #default="{ row }">
            <span v-if="!row.isDir">{{ formatSize(row.size) }}</span>
            <span v-else style="color:#bbb">—</span>
          </template>
        </el-table-column>

        <!-- Last modified column -->
        <el-table-column label="Last Modified" width="180">
          <template #default="{ row }">
            <span v-if="!row.isDir">{{ formatDate(row.lastModified) }}</span>
          </template>
        </el-table-column>

        <!-- Actions column -->
        <el-table-column label="Actions" width="240" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="!row.isDir"
              type="primary"
              :icon="Download"
              size="small"
              @click.stop="downloadFile(row)"
            >Download</el-button>
            <el-button
              v-if="!row.isDir"
              :icon="Share"
              size="small"
              @click.stop="openDownloadLinkDialog(row)"
            >Copy Link</el-button>
            <el-button
              type="danger"
              :icon="Delete"
              size="small"
              plain
              @click.stop="confirmDeleteObject(row)"
            >Delete</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- ===== Upload Dialog ===== -->
    <el-dialog v-model="showUploadDialog" title="Upload Files" width="520px" @closed="resetUpload">
      <div
        class="drop-zone"
        :class="{ 'drop-zone--over': isDragging }"
        @dragover.prevent="isDragging = true"
        @dragleave="isDragging = false"
        @drop.prevent="onDrop"
        @click="triggerFileInput"
      >
        <el-icon :size="48" color="#409eff"><UploadFilled /></el-icon>
        <p>Drop files here or <strong>click</strong> to select</p>
        <p class="hint">Large files are streamed directly – no size limit</p>
      </div>
      <input ref="fileInputRef" type="file" multiple style="display:none" @change="onFileInputChange" />

      <!-- Selected files list with per-file progress -->
      <div v-if="uploadFiles.length" class="upload-list">
        <div v-for="f in uploadFiles" :key="f.name" class="upload-item">
          <el-icon><Document /></el-icon>
          <span class="upload-filename">{{ f.name }}</span>
          <span class="upload-size">{{ formatSize(f.size) }}</span>
          <el-tag v-if="f.status === 'done'" type="success" size="small">Done</el-tag>
          <el-tag v-else-if="f.status === 'error'" type="danger" size="small">Error</el-tag>
        </div>
        <el-progress v-if="uploadProgress > 0" :percentage="uploadProgress" class="upload-progress" />
      </div>

      <template #footer>
        <el-button @click="showUploadDialog = false">Cancel</el-button>
        <el-button
          type="primary"
          :disabled="!uploadFiles.length || uploading"
          :loading="uploading"
          @click="startUpload"
        >Upload {{ uploadFiles.length > 0 ? `(${uploadFiles.length} file${uploadFiles.length > 1 ? 's' : ''})` : '' }}</el-button>
      </template>
    </el-dialog>

    <!-- ===== Create Bucket Dialog ===== -->
    <el-dialog v-model="showCreateDialog" title="Create Bucket" width="400px">
      <el-form :model="newBucket" label-width="80px" @submit.prevent="createBucket">
        <el-form-item label="Name">
          <el-input v-model="newBucket.name" placeholder="my-bucket" autofocus />
        </el-form-item>
        <el-form-item label="Region">
          <el-input v-model="newBucket.region" placeholder="us-east-1" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">Cancel</el-button>
        <el-button type="primary" @click="createBucket">Create</el-button>
      </template>
    </el-dialog>

    <!-- ===== Download Link Dialog ===== -->
    <el-dialog v-model="showDownloadLinkDialog" title="Generate Download Link" width="540px">
      <el-form label-width="100px">
        <el-form-item label="File">
          <span class="link-meta">{{ downloadLinkTarget?.key }}</span>
        </el-form-item>
        <el-form-item label="Expires in">
          <el-select v-model="downloadLinkExpiry" style="width:100%">
            <el-option label="1 hour" :value="3600" />
            <el-option label="6 hours" :value="21600" />
            <el-option label="24 hours (default)" :value="86400" />
            <el-option label="3 days" :value="259200" />
            <el-option label="7 days" :value="604800" />
          </el-select>
        </el-form-item>
      </el-form>
      <div v-if="downloadLinkUrl" class="generated-link">
        <el-input v-model="downloadLinkUrl" readonly>
          <template #append>
            <el-button :icon="CopyDocument" @click="copyToClipboard(downloadLinkUrl)">Copy</el-button>
          </template>
        </el-input>
      </div>
      <template #footer>
        <el-button @click="showDownloadLinkDialog = false">Close</el-button>
        <el-button type="primary" :loading="generatingDownloadLink" @click="generateDownloadLinkAction">
          Generate Link
        </el-button>
      </template>
    </el-dialog>

    <!-- ===== Upload Link Dialog ===== -->
    <el-dialog v-model="showUploadLinkDialog" title="Generate Upload Link" width="540px" @closed="resetUploadLink">
      <el-form label-width="100px">
        <el-form-item label="Destination">
          <el-input v-model="uploadLinkKey" placeholder="folder/filename.ext" />
          <div class="field-hint">Full object key (path + filename) for the upload destination.</div>
        </el-form-item>
        <el-form-item label="Expires in">
          <el-select v-model="uploadLinkExpiry" style="width:100%">
            <el-option label="1 hour" :value="3600" />
            <el-option label="6 hours" :value="21600" />
            <el-option label="24 hours (default)" :value="86400" />
            <el-option label="3 days" :value="259200" />
            <el-option label="7 days" :value="604800" />
          </el-select>
        </el-form-item>
      </el-form>
      <div v-if="uploadLinkUrl" class="generated-link">
        <p class="link-label">Upload page link (share with the uploader):</p>
        <el-input v-model="uploadPageUrl" readonly>
          <template #append>
            <el-button :icon="CopyDocument" @click="copyToClipboard(uploadPageUrl)">Copy</el-button>
          </template>
        </el-input>
      </div>
      <template #footer>
        <el-button @click="showUploadLinkDialog = false">Close</el-button>
        <el-button type="primary" :loading="generatingUploadLink" @click="generateUploadLinkAction">
          Generate Link
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, UploadFilled, Delete, Download, Folder, Document, Coin, Share, CopyDocument } from '@element-plus/icons-vue'
import {
  listBuckets,
  createBucket as apiCreateBucket,
  deleteBucket as apiDeleteBucket,
  listObjects,
  downloadUrl,
  uploadObjects,
  deleteObject,
  generateDownloadLink,
  generateUploadLink
} from '../api'

// ---- routing ----
const route = useRoute()
const router = useRouter()

// ---- reactive state ----
const buckets = ref([])
const objects = ref([])
const loading = ref(false)

// Currently selected bucket and "folder" prefix derived from the URL
const currentBucket = computed(() => route.params.bucket || '')
const currentPrefix = computed(() => {
  const match = route.params.pathMatch
  if (!match) return ''
  const raw = Array.isArray(match) ? match.join('/') : match
  return raw ? raw + '/' : ''
})

const prefixParts = computed(() => {
  const p = currentPrefix.value
  if (!p) return []
  return p.split('/').filter(Boolean)
})

// Upload state
const showUploadDialog = ref(false)
const uploadFiles = ref([])
const uploading = ref(false)
const uploadProgress = ref(0)
const isDragging = ref(false)
const fileInputRef = ref(null)

// Create bucket state
const showCreateDialog = ref(false)
const newBucket = ref({ name: '', region: 'us-east-1' })

// Download link state
const showDownloadLinkDialog = ref(false)
const downloadLinkTarget = ref(null)
const downloadLinkExpiry = ref(86400)
const downloadLinkUrl = ref('')
const generatingDownloadLink = ref(false)

// Upload link state
const showUploadLinkDialog = ref(false)
const uploadLinkKey = ref('')
const uploadLinkExpiry = ref(86400)
const uploadLinkUrl = ref('')
const uploadPageUrl = ref('')
const generatingUploadLink = ref(false)

// ---- lifecycle ----
onMounted(() => {
  fetchBuckets()
})

watch(
  () => [currentBucket.value, currentPrefix.value],
  ([bucket]) => {
    if (bucket) fetchObjects()
    else objects.value = []
  },
  { immediate: true }
)

// ---- bucket helpers ----
async function fetchBuckets() {
  try {
    const { data } = await listBuckets()
    buckets.value = data || []
  } catch (e) {
    ElMessage.error('Failed to load buckets: ' + (e.response?.data?.error || e.message))
  }
}

function selectBucket(name) {
  router.push({ name: 'bucket', params: { bucket: name } })
}

function goBucketRoot() {
  if (currentBucket.value) {
    router.push({ name: 'bucket', params: { bucket: currentBucket.value } })
  } else {
    router.push({ name: 'browser' })
  }
}

function navigateToDepth(index) {
  const parts = prefixParts.value.slice(0, index + 1)
  router.push({
    name: 'folder',
    params: { bucket: currentBucket.value, pathMatch: parts.join('/') }
  })
}

function openCreateBucket() {
  newBucket.value = { name: '', region: 'us-east-1' }
  showCreateDialog.value = true
}

async function createBucket() {
  const name = newBucket.value.name.trim()
  if (!name) return ElMessage.warning('Bucket name is required')
  try {
    await apiCreateBucket(name, newBucket.value.region || 'us-east-1')
    ElMessage.success(`Bucket "${name}" created`)
    showCreateDialog.value = false
    await fetchBuckets()
  } catch (e) {
    ElMessage.error('Failed to create bucket: ' + (e.response?.data?.error || e.message))
  }
}

async function confirmDeleteBucket() {
  try {
    await ElMessageBox.confirm(
      `Delete bucket "${currentBucket.value}"? All objects must be removed first.`,
      'Delete Bucket',
      { type: 'warning' }
    )
    await apiDeleteBucket(currentBucket.value)
    ElMessage.success('Bucket deleted')
    router.push({ name: 'browser' })
    await fetchBuckets()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('Failed: ' + (e.response?.data?.error || e.message))
  }
}

// ---- object helpers ----
async function fetchObjects() {
  loading.value = true
  try {
    const { data } = await listObjects(currentBucket.value, currentPrefix.value)
    objects.value = data || []
  } catch (e) {
    ElMessage.error('Failed to list objects: ' + (e.response?.data?.error || e.message))
    objects.value = []
  } finally {
    loading.value = false
  }
}

function handleRowClick(row) {
  if (!row.isDir) return
  // Navigate into the sub-folder; strip leading bucket name from key
  const key = row.key.replace(/\/$/, '')
  router.push({
    name: 'folder',
    params: { bucket: currentBucket.value, pathMatch: key }
  })
}

function downloadFile(row) {
  const url = downloadUrl(currentBucket.value, row.key)
  const a = document.createElement('a')
  a.href = url
  a.download = row.name
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

async function confirmDeleteObject(row) {
  const label = row.isDir ? `folder "${row.name}" (recursive)` : `file "${row.name}"`
  try {
    await ElMessageBox.confirm(`Delete ${label}?`, 'Confirm Delete', { type: 'warning' })
    await deleteObject(currentBucket.value, row.key)
    ElMessage.success('Deleted')
    await fetchObjects()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('Failed: ' + (e.response?.data?.error || e.message))
  }
}

// ---- upload helpers ----
function triggerFileInput() {
  fileInputRef.value?.click()
}

function onFileInputChange(e) {
  addFiles(Array.from(e.target.files))
  e.target.value = ''
}

function onDrop(e) {
  isDragging.value = false
  addFiles(Array.from(e.dataTransfer.files))
}

function addFiles(files) {
  for (const f of files) {
    if (!uploadFiles.value.find((u) => u.name === f.name && u.size === f.size)) {
      uploadFiles.value.push(Object.assign(f, { status: 'pending' }))
    }
  }
}

function resetUpload() {
  uploadFiles.value = []
  uploadProgress.value = 0
  uploading.value = false
}

async function startUpload() {
  if (!uploadFiles.value.length) return
  uploading.value = true
  uploadProgress.value = 0
  try {
    await uploadObjects(
      currentBucket.value,
      uploadFiles.value,
      // strip trailing slash from prefix
      currentPrefix.value.replace(/\/$/, ''),
      (e) => {
        if (e.total) {
          uploadProgress.value = Math.round((e.loaded / e.total) * 100)
        }
      }
    )
    uploadFiles.value.forEach((f) => (f.status = 'done'))
    ElMessage.success(`${uploadFiles.value.length} file(s) uploaded`)
    showUploadDialog.value = false
    await fetchObjects()
  } catch (e) {
    uploadFiles.value.forEach((f) => (f.status = 'error'))
    ElMessage.error('Upload failed: ' + (e.response?.data?.error || e.message))
  } finally {
    uploading.value = false
  }
}

// ---- presign link helpers ----
function openDownloadLinkDialog(row) {
  downloadLinkTarget.value = row
  downloadLinkExpiry.value = 86400
  downloadLinkUrl.value = ''
  showDownloadLinkDialog.value = true
}

async function generateDownloadLinkAction() {
  if (!downloadLinkTarget.value) return
  generatingDownloadLink.value = true
  try {
    const { data } = await generateDownloadLink(
      currentBucket.value,
      downloadLinkTarget.value.key,
      downloadLinkExpiry.value
    )
    downloadLinkUrl.value = data.url
  } catch (e) {
    ElMessage.error('Failed to generate link: ' + (e.response?.data?.error || e.message))
  } finally {
    generatingDownloadLink.value = false
  }
}

function openUploadLinkDialog() {
  uploadLinkKey.value = currentPrefix.value
  uploadLinkExpiry.value = 86400
  uploadLinkUrl.value = ''
  uploadPageUrl.value = ''
  showUploadLinkDialog.value = true
}

async function generateUploadLinkAction() {
  const key = uploadLinkKey.value.trim()
  if (!key) return ElMessage.warning('Destination key is required')
  generatingUploadLink.value = true
  try {
    const { data } = await generateUploadLink(currentBucket.value, key, uploadLinkExpiry.value)
    uploadLinkUrl.value = data.url
    const filename = key.split('/').pop()
    const params = new URLSearchParams({ url: data.url, filename })
    uploadPageUrl.value = `${window.location.origin}/upload?${params.toString()}`
  } catch (e) {
    ElMessage.error('Failed to generate link: ' + (e.response?.data?.error || e.message))
  } finally {
    generatingUploadLink.value = false
  }
}

function resetUploadLink() {
  uploadLinkUrl.value = ''
  uploadPageUrl.value = ''
}

async function copyToClipboard(text) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('Copied to clipboard')
  } catch {
    ElMessage.error('Failed to copy')
  }
}

// ---- formatting helpers ----
function formatSize(bytes) {
  if (bytes == null) return ''
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let n = bytes
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024
    i++
  }
  return `${n.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

function formatDate(ts) {
  if (!ts) return ''
  return new Date(ts).toLocaleString()
}
</script>

<style scoped>
.browser-layout {
  display: flex;
  height: calc(100vh - 60px);
  overflow: hidden;
}

/* ----- Sidebar ----- */
.sidebar {
  width: 220px;
  min-width: 220px;
  border-right: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
  background: #fafafa;
}

.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  border-bottom: 1px solid #e4e7ed;
}

.sidebar-title {
  font-weight: 600;
  font-size: 13px;
  color: #606266;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.sidebar-scroll {
  flex: 1;
}

.bucket-list {
  list-style: none;
  margin: 0;
  padding: 6px 0;
}

.bucket-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  cursor: pointer;
  font-size: 14px;
  color: #303133;
  transition: background 0.15s;
  user-select: none;
}

.bucket-item:hover {
  background: #ecf5ff;
}

.bucket-item.active {
  background: #409eff;
  color: #fff;
}

.bucket-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* ----- Main area ----- */
.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  border-bottom: 1px solid #e4e7ed;
  gap: 12px;
  flex-wrap: wrap;
}

.toolbar-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.breadcrumb {
  flex: 1;
}

.breadcrumb-link {
  cursor: pointer;
  color: #409eff;
}

.breadcrumb-link:hover {
  text-decoration: underline;
}

/* ----- File rows ----- */
.file-row {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.file-icon {
  flex-shrink: 0;
}

.folder-name {
  font-weight: 500;
}

/* ----- Upload dialog ----- */
.drop-zone {
  border: 2px dashed #c0c4cc;
  border-radius: 8px;
  padding: 32px 24px;
  text-align: center;
  cursor: pointer;
  transition: border-color 0.2s, background 0.2s;
}

.drop-zone:hover,
.drop-zone--over {
  border-color: #409eff;
  background: #ecf5ff;
}

.drop-zone p {
  margin: 8px 0 0;
  color: #606266;
}

.hint {
  font-size: 12px;
  color: #909399 !important;
}

.upload-list {
  margin-top: 16px;
}

.upload-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  border-bottom: 1px solid #f0f0f0;
  font-size: 13px;
}

.upload-filename {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.upload-size {
  color: #909399;
  flex-shrink: 0;
}

.upload-progress {
  margin-top: 10px;
}

.generated-link {
  margin-top: 16px;
}

.link-label {
  margin: 0 0 8px;
  font-size: 13px;
  color: #606266;
}

.link-meta {
  font-size: 13px;
  color: #303133;
  word-break: break-all;
}

.field-hint {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
</style>
