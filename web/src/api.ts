export class ApiError extends Error {
  constructor(readonly status: number, message: string) { super(message) }
}

export async function request<T>(path: string, method = 'GET', body?: unknown): Promise<T> {
  const response = await fetch('/api/' + path, {
    method,
    credentials: 'same-origin',
    signal: AbortSignal.timeout(path.split('?')[0]==='watch'?5000:30000),
    headers: body === undefined ? undefined : {'Content-Type': 'application/json'},
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  if (!response.ok) {
    const result = await response.json().catch(() => ({}))
    throw new ApiError(response.status, result.error || 'FamilyTV could not complete the request.')
  }
  return response.json()
}

export function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : 'Could not reach FamilyTV.'
}
