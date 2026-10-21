import { type FormEvent, useEffect, useState } from "react"
import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { CheckCircle2, Plus, Search } from "lucide-react"
import { api, errorMessage } from "@/lib/api"
import { useDebouncedValue } from "@/hooks/useDebouncedValue"
import { FormField, Modal, PageState, Pagination, PrimaryButton, StatusBadge } from "@/components/ui"

function defaultDueDate() {
  const date = new Date()
  date.setDate(date.getDate() + 14)
  return date.toISOString().slice(0, 10)
}

export default function Loans() {
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState("")
  const [status, setStatus] = useState("")
  const [modalOpen, setModalOpen] = useState(false)
  const [form, setForm] = useState({ bookId: "", memberId: "", dueDate: defaultDueDate() })
  const [notice, setNotice] = useState("")
  const debouncedSearch = useDebouncedValue(search)
  const queryClient = useQueryClient()
  const loans = useQuery({
    queryKey: ["loans", page, debouncedSearch, status],
    queryFn: () => api.loans({ page, search: debouncedSearch, status }),
    placeholderData: keepPreviousData,
  })
  const availableBooks = useQuery({
    queryKey: ["books", "loan-options"],
    queryFn: () => api.books({ page: 1, pageSize: 100, status: "available" }),
    enabled: modalOpen,
  })
  const activeMembers = useQuery({
    queryKey: ["members", "loan-options"],
    queryFn: () => api.members({ page: 1, pageSize: 100, status: "active" }),
    enabled: modalOpen,
  })
  useEffect(() => setPage(1), [debouncedSearch, status])

  const refresh = () => {
    queryClient.invalidateQueries({ queryKey: ["loans"] })
    queryClient.invalidateQueries({ queryKey: ["books"] })
    queryClient.invalidateQueries({ queryKey: ["members"] })
    queryClient.invalidateQueries({ queryKey: ["dashboard"] })
  }
  const create = useMutation({
    mutationFn: () => api.createLoan(form),
    onSuccess: () => {
      setModalOpen(false)
      setNotice("借阅办理成功，库存已同步扣减")
      refresh()
    },
  })
  const returnBook = useMutation({
    mutationFn: api.returnLoan,
    onSuccess: () => {
      setNotice("归还办理成功，库存已恢复")
      refresh()
    },
  })

  const openCreate = () => {
    setForm({ bookId: "", memberId: "", dueDate: defaultDueDate() })
    setModalOpen(true)
  }
  const submit = (event: FormEvent) => {
    event.preventDefault()
    create.mutate()
  }
  const confirmReturn = (id: string, title: string) => {
    if (window.confirm(`确认归还《${title}》？`)) returnBook.mutate(id)
  }

  return (
    <div className="content-section">
      <div className="toolbar">
        <div className="search-box">
          <Search size={17} />
          <input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="搜索图书或读者" />
        </div>
        <select value={status} onChange={(event) => setStatus(event.target.value)} aria-label="借阅状态">
          <option value="">全部状态</option>
          <option value="borrowed">借阅中</option>
          <option value="overdue">已逾期</option>
          <option value="returned">已归还</option>
        </select>
        <span className="toolbar-spacer" />
        <PrimaryButton onClick={openCreate}><Plus size={17} />办理借阅</PrimaryButton>
      </div>
      {notice && <div className="inline-notice" role="status">{notice}<button onClick={() => setNotice("")}>关闭</button></div>}
      {returnBook.isError && <div className="inline-notice is-error">{errorMessage(returnBook.error)}</div>}

      {loans.isLoading ? <PageState loading /> : loans.isError ? (
        <PageState error={errorMessage(loans.error)} />
      ) : loans.data!.items.length === 0 ? (
        <PageState empty="没有找到符合条件的借阅记录" />
      ) : (
        <>
          <div className="table-wrap">
            <table>
              <thead><tr><th>图书</th><th>读者</th><th>借出时间</th><th>应还日期</th><th>状态</th><th className="actions">操作</th></tr></thead>
              <tbody>
                {loans.data!.items.map((loan) => (
                  <tr key={loan.id}>
                    <td><strong>{loan.bookTitle}</strong><small className="mono">{loan.bookIsbn}</small></td>
                    <td>{loan.memberName}<small className="mono">{loan.memberNo}</small></td>
                    <td>{new Date(loan.borrowedAt).toLocaleDateString("zh-CN")}</td>
                    <td className="mono">{loan.dueDate}</td>
                    <td><StatusBadge status={loan.status} /></td>
                    <td className="actions">
                      {loan.status !== "returned" ? (
                        <button className="text-action" onClick={() => confirmReturn(loan.id, loan.bookTitle)}>
                          <CheckCircle2 size={16} />归还
                        </button>
                      ) : <span className="muted">已完成</span>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <Pagination page={page} pageSize={loans.data!.pageSize} total={loans.data!.total} onChange={setPage} />
        </>
      )}

      <Modal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title="办理借阅"
        description="选择可借图书、有效读者和应还日期"
        footer={
          <>
            <button className="button button-secondary" onClick={() => setModalOpen(false)}>取消</button>
            <PrimaryButton type="submit" form="loan-form" loading={create.isPending} disabled={!form.bookId || !form.memberId}>确认借出</PrimaryButton>
          </>
        }
      >
        <form id="loan-form" className="loan-form" onSubmit={submit}>
          <FormField label="图书" required>
            <select required value={form.bookId} onChange={(e) => setForm({ ...form, bookId: e.target.value })}>
              <option value="">请选择可借图书</option>
              {availableBooks.data?.items.map((book) => (
                <option key={book.id} value={book.id}>{book.title} · {book.author}（余 {book.availableCopies}）</option>
              ))}
            </select>
          </FormField>
          <FormField label="读者" required>
            <select required value={form.memberId} onChange={(e) => setForm({ ...form, memberId: e.target.value })}>
              <option value="">请选择有效读者</option>
              {activeMembers.data?.items.map((member) => (
                <option key={member.id} value={member.id}>{member.name} · {member.memberNo}（在借 {member.activeLoans}）</option>
              ))}
            </select>
          </FormField>
          <FormField label="应还日期" required>
            <input required type="date" min={new Date().toISOString().slice(0, 10)} value={form.dueDate} onChange={(e) => setForm({ ...form, dueDate: e.target.value })} />
          </FormField>
          {availableBooks.data?.items.length === 0 && <p className="form-hint">当前没有可借图书，请先补充库存。</p>}
          {activeMembers.data?.items.length === 0 && <p className="form-hint">当前没有启用读者，请先新增或启用读者。</p>}
          {create.isError && <p className="form-error">{errorMessage(create.error)}</p>}
        </form>
      </Modal>
    </div>
  )
}
