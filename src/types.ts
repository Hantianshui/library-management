export type User = {
  id: string
  username: string
}

export type Book = {
  id: string
  isbn: string
  title: string
  author: string
  category: string
  publisher: string
  publishedYear: number | null
  totalCopies: number
  availableCopies: number
  active: boolean
  createdAt: string
  updatedAt: string
}

export type BookInput = Omit<Book, "id" | "availableCopies" | "createdAt" | "updatedAt">

export type Member = {
  id: string
  memberNo: string
  name: string
  phone: string
  email: string
  active: boolean
  activeLoans: number
  createdAt: string
  updatedAt: string
}

export type MemberInput = Omit<Member, "id" | "activeLoans" | "createdAt" | "updatedAt">

export type LoanStatus = "borrowed" | "returned" | "overdue"

export type Loan = {
  id: string
  bookId: string
  bookTitle: string
  bookIsbn: string
  memberId: string
  memberName: string
  memberNo: string
  borrowedAt: string
  dueDate: string
  returnedAt: string | null
  status: LoanStatus
}

export type Activity = {
  id: number
  action: string
  entityType: string
  summary: string
  createdAt: string
}

export type Dashboard = {
  bookTitles: number
  totalCopies: number
  availableCopies: number
  activeLoans: number
  overdueLoans: number
  activeMembers: number
  recentLoans: Loan[]
  recentActivity: Activity[]
}

export type Page<T> = {
  items: T[]
  page: number
  pageSize: number
  total: number
}
