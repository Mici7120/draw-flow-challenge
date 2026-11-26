import type { DrawflowExport } from '../ast/types'

export interface ProgramSummary {
  id: string
  name: string
  createdAt: string
  updatedAt: string
}

export interface ProgramRecord extends ProgramSummary {
  graph: DrawflowExport
  code: string
}

export interface RunResult {
  stdout: string
  stderr: string
  exitCode: number
  timedOut: boolean
  durationMs: number
}

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? '/api'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
    ...init,
  })

  if (res.status === 204) {
    return undefined as T
  }

  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    const message =
      typeof data?.error === 'string' ? data.error : `Request failed (${res.status})`
    throw new Error(message)
  }
  return data as T
}

export async function listPrograms(): Promise<ProgramSummary[]> {
  const res = await request<{ programs: ProgramSummary[] }>('/programs')
  return res.programs ?? []
}

export async function getProgram(id: string): Promise<ProgramRecord> {
  return request<ProgramRecord>(`/programs/${encodeURIComponent(id)}`)
}

export async function saveProgram(payload: {
  name: string
  graph: DrawflowExport
  code: string
}): Promise<ProgramRecord> {
  return request<ProgramRecord>('/programs', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export async function updateProgram(
  id: string,
  payload: {
    name: string
    graph: DrawflowExport
    code: string
  },
): Promise<ProgramRecord> {
  return request<ProgramRecord>(`/programs/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
  })
}

export async function deleteProgram(id: string): Promise<void> {
  await request<void>(`/programs/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

export async function runCode(code: string): Promise<RunResult> {
  return request<RunResult>('/programs/run', {
    method: 'POST',
    body: JSON.stringify({ code }),
  })
}

export async function checkHealth(): Promise<boolean> {
  try {
    await request<{ status: string }>('/health')
    return true
  } catch {
    return false
  }
}
