export type ConnectionState = 'connected' | 'reconnecting' | 'disconnected';

const API_BASE = (import.meta.env.VITE_API_BASE as string) || 'http://localhost:8081';

const wait = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

type SubmitJobResponse = {
  job_id?: string;
  status?: string;
  response?: string;
};

type JobStatusResponse = {
  job_id?: string;
  status?: 'processing' | 'success' | 'fail';
  response?: string;
};

export async function sendMessage(
  message: string,
  sessionId = 'default',
  onStatusChange?: (status: ConnectionState) => void
): Promise<string> {
  const delays = [2000, 4000, 6000];
  let lastError: unknown = null;
  let jobId: string | undefined;

  for (let attempt = 0; attempt < delays.length; attempt++) {
    try {
      if (attempt > 0) {
        onStatusChange?.('reconnecting');
      }

      const response = await fetch(`${API_BASE}/api/chat`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message, session_id: sessionId })
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      const data = (await response.json()) as SubmitJobResponse;
      jobId = data.job_id;
      if (!jobId) {
        onStatusChange?.('connected');
        return data.response ?? "I'm having trouble right now. Please try again in a moment.";
      }
      break;
    } catch (error) {
      lastError = error;
      if (attempt < delays.length - 1) {
        onStatusChange?.('reconnecting');
        await wait(delays[attempt]);
        continue;
      }
    }
  }

  if (!jobId) {
    console.error('chat submit failed after retries', lastError);
    onStatusChange?.('disconnected');
    return 'Connection lost. Please try again.';
  }

  onStatusChange?.('reconnecting');
  const pollAttempts = 45;
  for (let i = 0; i < pollAttempts; i++) {
    try {
      const resp = await fetch(`${API_BASE}/api/chat?job_id=${encodeURIComponent(jobId)}`);
      if (!resp.ok) {
        throw new Error(`poll failed with ${resp.status}`);
      }

      const data = (await resp.json()) as JobStatusResponse;
      if (data.status === 'processing') {
        await wait(1200);
        continue;
      }

      onStatusChange?.('connected');
      return data.response ?? "I'm having trouble right now. Please try again in a moment.";
    } catch (error) {
      lastError = error;
      onStatusChange?.('reconnecting');
      await wait(1500);
    }
  }

  console.error('chat job polling timed out/failed', lastError);
  onStatusChange?.('disconnected');
  return 'Connection lost. Please try again.';
}

export async function checkHealth(): Promise<boolean> {
  try {
    const resp = await fetch(`${API_BASE}/health`);
    return resp.ok;
  } catch {
    return false;
  }
}
