import { useState, useEffect, useCallback } from 'react'

const DEFAULT_API_URL = import.meta.env.VITE_UPLOAD_API_URL || 'http://localhost:8080'
const DEFAULT_API_KEY = (import.meta.env.VITE_UPLOAD_API_KEY || '').trim()

export default function App() {
  const [apiUrl, setApiUrl] = useState(DEFAULT_API_URL)
  const [userId, setUserId] = useState('')
  const [title, setTitle] = useState('')
  const [file, setFile] = useState(null)
  const [loading, setLoading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState(0)
  const [message, setMessage] = useState({ text: '', error: false })
  const [uploads, setUploads] = useState([])
  const [videos, setVideos] = useState([])
  const [uploadsLoading, setUploadsLoading] = useState(false)
  const [videosLoading, setVideosLoading] = useState(false)
  const [uploadsError, setUploadsError] = useState('')
  const [videosError, setVideosError] = useState('')
  const [apiKeyOverride, setApiKeyOverride] = useState('')

  const baseUrl = (apiUrl || DEFAULT_API_URL).trim().replace(/\/$/, '')
  const effectiveApiKey = apiKeyOverride.trim() || DEFAULT_API_KEY

  const apiHeaders = useCallback((includeJSONContentType = false) => {
    const headers = {}
    if (includeJSONContentType) headers['Content-Type'] = 'application/json'
    if (effectiveApiKey) headers['X-Api-Key'] = effectiveApiKey
    return headers
  }, [effectiveApiKey])

  const fetchUploads = useCallback(async () => {
    setUploadsLoading(true)
    setUploadsError('')
    try {
      const res = await fetch(`${baseUrl}/uploads`, { headers: apiHeaders() })
      if (!res.ok) throw new Error('Failed to load uploads')
      const data = await res.json()
      setUploads(data.uploads ?? [])
    } catch (e) {
      setUploadsError(e.message || 'Error loading uploads')
      setUploads([])
    } finally {
      setUploadsLoading(false)
    }
  }, [apiHeaders, baseUrl])

  const fetchVideos = useCallback(async () => {
    setVideosLoading(true)
    setVideosError('')
    try {
      const res = await fetch(`${baseUrl}/videos`, { headers: apiHeaders() })
      if (!res.ok) throw new Error('Failed to load videos')
      const data = await res.json()
      setVideos(data.videos ?? [])
    } catch (e) {
      setVideosError(e.message || 'Error loading videos')
      setVideos([])
    } finally {
      setVideosLoading(false)
    }
  }, [apiHeaders, baseUrl])

  useEffect(() => {
    fetchUploads()
    fetchVideos()
  }, [fetchUploads, fetchVideos])

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
        headers: apiHeaders(true),
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
        headers: apiHeaders(),
      })

      if (!finalizeRes.ok) {
        const errText = await finalizeRes.text()
        throw new Error('Finalize failed: ' + (errText || finalizeRes.status))
      }

      setMessage({ text: `Upload completed. Video ID: ${videoId}`, error: false })
      setFile(null)
      const fileInput = document.getElementById('file')
      if (fileInput) fileInput.value = ''
      fetchUploads()
      fetchVideos()
    } catch (err) {
      setMessage({ text: err?.message || 'Upload failed', error: true })
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
        <div className="formGroup">
          <label htmlFor="apiKey">API key (optional override)</label>
          <input
            id="apiKey"
            type="password"
            value={apiKeyOverride}
            onChange={(e) => setApiKeyOverride(e.target.value)}
            placeholder={DEFAULT_API_KEY ? 'Using key from VITE_UPLOAD_API_KEY' : 'Set VITE_UPLOAD_API_KEY or enter key'}
            autoComplete="off"
          />
        </div>
        {!effectiveApiKey && (
          <p className="tableStatus error">
            No API key configured. Upload API calls requiring <code>X-Api-Key</code> will return 401.
          </p>
        )}
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

      <section className="tablesSection">
        <div className="tableBlock">
          <h2>Uploads</h2>
          {uploadsLoading && <p className="tableStatus">Loading…</p>}
          {uploadsError && <p className="tableStatus error">{uploadsError}</p>}
          {!uploadsLoading && !uploadsError && (
            <div className="tableWrap">
              <table className="dataTable" aria-label="Uploads">
                <thead>
                  <tr>
                    <th>ID</th>
                    <th>Video ID</th>
                    <th>Storage path</th>
                    <th>Status</th>
                    <th>Created</th>
                    <th>Updated</th>
                  </tr>
                </thead>
                <tbody>
                  {uploads.length === 0 ? (
                    <tr><td colSpan={6}>No uploads</td></tr>
                  ) : (
                    uploads.map((u) => (
                      <tr key={u.id}>
                        <td className="cellId">{u.id}</td>
                        <td className="cellId">{u.video_id}</td>
                        <td>{u.storage_path || '—'}</td>
                        <td>{u.status}</td>
                        <td>{u.created_at}</td>
                        <td>{u.updated_at}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          )}
        </div>

        <div className="tableBlock">
          <h2>Videos</h2>
          {videosLoading && <p className="tableStatus">Loading…</p>}
          {videosError && <p className="tableStatus error">{videosError}</p>}
          {!videosLoading && !videosError && (
            <div className="tableWrap">
              <table className="dataTable" aria-label="Videos">
                <thead>
                  <tr>
                    <th>ID</th>
                    <th>User ID</th>
                    <th>Title</th>
                    <th>Format</th>
                    <th>Status</th>
                    <th>Duration (s)</th>
                    <th>Created</th>
                    <th>Updated</th>
                  </tr>
                </thead>
                <tbody>
                  {videos.length === 0 ? (
                    <tr><td colSpan={8}>No videos</td></tr>
                  ) : (
                    videos.map((v) => (
                      <tr key={v.id}>
                        <td className="cellId">{v.id}</td>
                        <td className="cellId">{v.user_id}</td>
                        <td>{v.title}</td>
                        <td>{v.format || '—'}</td>
                        <td>{v.status}</td>
                        <td>{v.duration_sec != null ? Number(v.duration_sec).toFixed(1) : '—'}</td>
                        <td>{v.created_at}</td>
                        <td>{v.updated_at}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </section>
    </div>
  )
}
