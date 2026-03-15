import { useState } from 'react'

const DEFAULT_API_URL = import.meta.env.VITE_UPLOAD_API_URL || 'http://localhost:8080'

export default function App() {
  const [apiUrl, setApiUrl] = useState(DEFAULT_API_URL)
  const [userId, setUserId] = useState('')
  const [title, setTitle] = useState('')
  const [file, setFile] = useState(null)
  const [loading, setLoading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState(0)
  const [message, setMessage] = useState({ text: '', error: false })

  const baseUrl = (apiUrl || DEFAULT_API_URL).trim().replace(/\/$/, '')

  async function handleSubmit(e) {
    e.preventDefault()
    setMessage({ text: '', error: false })

    if (!userId.trim() || !title.trim() || !file) {
      setMessage({ text: 'Please fill User ID, Title and choose a video file.', error: true })
      return
    }

    setLoading(true)
    setUploadProgress(0)
    try {
      const presignRes = await fetch(`${baseUrl}/videos/upload/presign`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_id: userId.trim(), title: title.trim() }),
      })

      if (!presignRes.ok) {
        const errText = await presignRes.text()
        throw new Error('Presign failed: ' + (errText || presignRes.status))
      }

      const { upload_url: uploadUrl, video_id: videoId } = await presignRes.json()
      if (!uploadUrl || !videoId) throw new Error('Invalid presign response')

      await new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest()
        xhr.upload.addEventListener('progress', (e) => {
          if (e.lengthComputable) {
            setUploadProgress(Math.round((e.loaded / e.total) * 100))
          }
        })
        xhr.addEventListener('load', () => {
          if (xhr.status >= 200 && xhr.status < 300) resolve()
          else reject(new Error('Upload to storage failed: ' + xhr.status))
        })
        xhr.addEventListener('error', () => reject(new Error('Upload failed')))
        xhr.open('PUT', uploadUrl)
        xhr.setRequestHeader('Content-Type', file.type || 'video/mp4')
        xhr.send(file)
      })

      const finalizeRes = await fetch(`${baseUrl}/videos/${encodeURIComponent(videoId)}/upload/finalize`, {
        method: 'POST',
      })

      if (!finalizeRes.ok) {
        const errText = await finalizeRes.text()
        throw new Error('Finalize failed: ' + (errText || finalizeRes.status))
      }

      setMessage({ text: `Upload completed. Video ID: ${videoId}`, error: false })
      setFile(null)
      const fileInput = document.getElementById('file')
      if (fileInput) fileInput.value = ''
    } catch (err) {
      setMessage({ text: err.message || 'Upload failed', error: true })
    } finally {
      setLoading(false)
      setUploadProgress(0)
    }
  }

  return (
    <div className="app">
      <h1>E2E Upload (presigned URL)</h1>
      <form onSubmit={handleSubmit}>
        <div className="formGroup">
          <label htmlFor="apiUrl">API URL</label>
          <input
            id="apiUrl"
            type="url"
            value={apiUrl}
            onChange={(e) => setApiUrl(e.target.value)}
            placeholder="http://localhost:8080"
          />
        </div>
        <div className="formGroup">
          <label htmlFor="userId">User ID</label>
          <input
            id="userId"
            type="text"
            value={userId}
            onChange={(e) => setUserId(e.target.value)}
            placeholder="e.g. user-123"
            required
          />
        </div>
        <div className="formGroup">
          <label htmlFor="title">Title</label>
          <input
            id="title"
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Video title"
            required
          />
        </div>
        <div className="formGroup">
          <label htmlFor="file">Video file</label>
          <input
            id="file"
            type="file"
            accept="video/mp4,.mp4"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            required
          />
        </div>
        <button type="submit" className="submit" disabled={loading}>
          {loading ? 'Uploading…' : 'Upload'}
        </button>
        {loading && (
          <div className="progressWrap">
            <progress className="progressBar" value={uploadProgress} max={100} aria-label="Upload progress" />
            <span className="progressLabel">{uploadProgress}%</span>
          </div>
        )}
      </form>
      {message.text && (
        <div className={`message ${message.error ? 'error' : 'success'}`} role="alert">
          {message.text}
        </div>
      )}
    </div>
  )
}
