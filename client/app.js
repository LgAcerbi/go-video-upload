(function () {
  const form = document.getElementById('uploadForm');
  const submitBtn = document.getElementById('submitBtn');
  const messageEl = document.getElementById('message');

  function showMessage(text, isError) {
    messageEl.textContent = text;
    messageEl.className = 'message ' + (isError ? 'error' : 'success');
    messageEl.hidden = false;
  }

  function hideMessage() {
    messageEl.hidden = true;
  }

  function getApiUrl() {
    const url = document.getElementById('apiUrl').value.trim();
    return url ? url.replace(/\/$/, '') : 'http://localhost:8080';
  }

  form.addEventListener('submit', async function (e) {
    e.preventDefault();
    hideMessage();

    const apiUrl = getApiUrl();
    const userId = document.getElementById('userId').value.trim();
    const title = document.getElementById('title').value.trim();
    const fileInput = document.getElementById('file');
    const file = fileInput.files[0];

    if (!userId || !title || !file) {
      showMessage('Please fill User ID, Title and choose a video file.', true);
      return;
    }

    submitBtn.disabled = true;

    try {
      const presignRes = await fetch(apiUrl + '/videos/upload/presign', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ user_id: userId, title: title })
      });

      if (!presignRes.ok) {
        const errText = await presignRes.text();
        throw new Error('Presign failed: ' + (errText || presignRes.status));
      }

      const { upload_url: uploadUrl, video_id: videoId } = await presignRes.json();
      if (!uploadUrl || !videoId) {
        throw new Error('Invalid presign response');
      }

      const putRes = await fetch(uploadUrl, {
        method: 'PUT',
        body: file,
        headers: {
          'Content-Type': file.type || 'video/mp4'
        }
      });

      if (!putRes.ok) {
        throw new Error('Upload to storage failed: ' + putRes.status);
      }

      const finalizeRes = await fetch(apiUrl + '/videos/' + encodeURIComponent(videoId) + '/upload/finalize', {
        method: 'POST'
      });

      if (!finalizeRes.ok) {
        const errText = await finalizeRes.text();
        throw new Error('Finalize failed: ' + (errText || finalizeRes.status));
      }

      showMessage('Upload completed. Video ID: ' + videoId, false);
      fileInput.value = '';
    } catch (err) {
      showMessage(err.message || 'Upload failed', true);
    } finally {
      submitBtn.disabled = false;
    }
  });
})();
