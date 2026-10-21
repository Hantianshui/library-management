import type {
  Book,
  BookInput,
  Dashboard,
  Loan,
  Member,
  MemberInput,
  Page,
  User,
} from "@/types"

type Envelope<T> = { data: T; requestId: string }
type ErrorEnvelope = { error: { code: string; message: string }; requestId: string }
type QueryValue = string | number | boolean | null | undefined

export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly code: string,
    readonly requestId?: string,
  ) {
    super(message)
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    credentials: "same-origin",
    ...init,
    headers: {
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
  })
  const payload = (await response.json().catch(() => null)) as Envelope<T> | ErrorEnvelope | null
  if (!response.ok) {
    const error = payload && "error" in payload ? payload : null
    throw new ApiError(
      error?.error.message || "请求失败，请稍后重试",
      response.status,
      error?.error.code || "REQUEST_FAILED",
      error?.requestId,
    )
  }
  if (!payload || !("data" in payload)) {
    throw new ApiError("服务响应格式不正确", response.status, "INVALID_RESPONSE")
  }
  return payload.data
}

function query(values: Record<string, QueryValue>) {
  const params = new URLSearchParams()
  Object.entries(values).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== "") {
      params.set(key, String(value))
    }
  })
  const serialized = params.toString()
  return serialized ? `?${serialized}` : ""
}

const json = (value: unknown) => JSON.stringify(value)

export const api = {
  login: (username: string, password: string) =>
    request<User>("/api/v1/auth/login", {
      method: "POST",
      body: json({ username, password }),
    }),
  logout: () => request<{ loggedOut: boolean }>("/api/v1/auth/logout", { method: "POST" }),
  me: () => request<User>("/api/v1/auth/me"),
  dashboard: () => request<Dashboard>("/api/v1/dashboard"),
  books: (params: { page: number; pageSize?: number; search?: string; status?: string; category?: string }) =>
    request<Page<Book>>(`/api/v1/books${query(params)}`),
  book: (id: string) =>
  request<Book>(`/api/v1/books/${id}`),
  categories: () => request<string[]>("/api/v1/books/categories"),
  createBook: (input: BookInput) =>
    request<Book>("/api/v1/books", { method: "POST", body: json(input) }),
  updateBook: (id: string, input: BookInput) =>
    request<Book>(`/api/v1/books/${id}`, { method: "PUT", body: json(input) }),
  deleteBook: (id: string) => request<{ deleted: boolean }>(`/api/v1/books/${id}`, { method: "DELETE" }),
  members: (params: { page: number; pageSize?: number; search?: string; status?: string }) =>
    request<Page<Member>>(`/api/v1/members${query(params)}`),
  createMember: (input: MemberInput) =>
    request<Member>("/api/v1/members", { method: "POST", body: json(input) }),
  updateMember: (id: string, input: MemberInput) =>
    request<Member>(`/api/v1/members/${id}`, { method: "PUT", body: json(input) }),
  deleteMember: (id: string) =>
    request<{ deleted: boolean }>(`/api/v1/members/${id}`, { method: "DELETE" }),
  loans: (params: { page: number; pageSize?: number; search?: string; status?: string }) =>
    request<Page<Loan>>(`/api/v1/loans${query(params)}`),
  createLoan: (input: { bookId: string; memberId: string; dueDate: string }) =>
    request<Loan>("/api/v1/loans", { method: "POST", body: json(input) }),
  returnLoan: (id: string) => request<Loan>(`/api/v1/loans/${id}/return`, { method: "POST" }),
}

export function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : "操作失败，请稍后重试"
}
