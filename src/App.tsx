import { useQuery } from "@tanstack/react-query"
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom"
import { api } from "@/lib/api"
import AppShell from "@/components/AppShell"
import { PageState } from "@/components/ui"
import Books from "@/pages/Books"
import Dashboard from "@/pages/Dashboard"
import Loans from "@/pages/Loans"
import Login from "@/pages/Login"
import Members from "@/pages/Members"
import BookDetails from "@/pages/BookDetails"

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route element={<ProtectedArea />}>
          <Route index element={<Dashboard />} />
          <Route path="books" element={<Books />} />
          <Route path="books/:id" element={<BookDetails />} />
          <Route path="members" element={<Members />} />
          <Route path="loans" element={<Loans />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}

function ProtectedArea() {
  const me = useQuery({
    queryKey: ["me"],
    queryFn: api.me,
    retry: false,
    staleTime: 5 * 60 * 1000,
  })
  if (me.isLoading) {
    return <div className="full-page-state"><PageState loading /></div>
  }
  if (me.isError) {
    return <Navigate to="/login" replace />
  }
  return <AppShell user={me.data} />
}
