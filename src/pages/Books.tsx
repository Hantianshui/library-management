import { type FormEvent, useEffect, useState } from "react"
import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Pencil, Plus, Search, Trash2 } from "lucide-react"
import { api, errorMessage } from "@/lib/api"
import type { Book, BookInput } from "@/types"
import { useDebouncedValue } from "@/hooks/useDebouncedValue"
import { FormField, IconButton, Modal, PageState, Pagination, PrimaryButton, StatusBadge } from "@/components/ui"
import { useNavigate } from "react-router-dom"
const emptyBook: BookInput = {
  isbn: "",
  title: "",
  author: "",
  category: "",
  publisher: "",
  publishedYear: null,
  totalCopies: 1,
  active: true,
}
export default function Books() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState("")
  const [status, setStatus] = useState("")
  const [category, setCategory] = useState("")
  const [editing, setEditing] = useState<Book | null>(null)
  const [form, setForm] = useState<BookInput>(emptyBook)
  const [modalOpen, setModalOpen] = useState(false)
  const [notice, setNotice] = useState("")
  const debouncedSearch = useDebouncedValue(search)
  const queryClient = useQueryClient()
  const books = useQuery({
    queryKey: ["books", page, debouncedSearch, status, category],
    queryFn: () => api.books({ page, search: debouncedSearch, status, category }),
    placeholderData: keepPreviousData,
  })
  const categories = useQuery({ queryKey: ["book-categories"], queryFn: api.categories })

  useEffect(() => setPage(1), [debouncedSearch, status, category])

  const save = useMutation({
    mutationFn: () => editing ? api.updateBook(editing.id, form) : api.createBook(form),
    onSuccess: () => {
      setModalOpen(false)
      setNotice(editing ? "图书信息已更新" : "图书已入库")
      queryClient.invalidateQueries({ queryKey: ["books"] })
      queryClient.invalidateQueries({ queryKey: ["book-categories"] })
      queryClient.invalidateQueries({ queryKey: ["dashboard"] })
    },
  })
  const remove = useMutation({
    mutationFn: api.deleteBook,
    onSuccess: () => {
      setNotice("图书已删除")
      queryClient.invalidateQueries({ queryKey: ["books"] })
      queryClient.invalidateQueries({ queryKey: ["dashboard"] })
    },
  })

  const openCreate = () => {
    setEditing(null)
    setForm(emptyBook)
    setModalOpen(true)
  }
  const openEdit = (book: Book) => {
    setEditing(book)
    setForm({
      isbn: book.isbn,
      title: book.title,
      author: book.author,
      category: book.category,
      publisher: book.publisher,
      publishedYear: book.publishedYear,
      totalCopies: book.totalCopies,
      active: book.active,
    })
    setModalOpen(true)
  }
  const submit = (event: FormEvent) => {
    event.preventDefault()
    save.mutate()
  }
  const confirmDelete = (book: Book) => {
    if (window.confirm(`确认删除《${book.title}》？此操作不可撤销。`)) remove.mutate(book.id)
  }

  return (
    <div className="content-section">
      <div className="toolbar">
        <div className="search-box">
          <Search size={17} />
          <input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="搜索书名、作者或 ISBN" />
        </div>
        <select value={category} onChange={(event) => setCategory(event.target.value)} aria-label="分类筛选">
          <option value="">全部分类</option>
          {categories.data?.map((item) => <option key={item}>{item}</option>)}
        </select>
        <select value={status} onChange={(event) => setStatus(event.target.value)} aria-label="库存状态">
          <option value="">全部状态</option>
          <option value="available">可借</option>
          <option value="unavailable">无库存</option>
          <option value="inactive">已停用</option>
        </select>
        <PrimaryButton onClick={openCreate}><Plus size={17} />新增图书</PrimaryButton>
      </div>
      {notice && <div className="inline-notice" role="status">{notice}<button onClick={() => setNotice("")}>关闭</button></div>}
      {remove.isError && <div className="inline-notice is-error">{errorMessage(remove.error)}</div>}

      {books.isLoading ? <PageState loading /> : books.isError ? (
        <PageState error={errorMessage(books.error)} />
      ) : books.data!.items.length === 0 ? (
        <PageState empty="没有找到符合条件的图书" />
      ) : (
        <>
          <div className="table-wrap">
            <table>
              <thead><tr><th>图书</th><th>分类</th><th>出版社 / 年份</th><th>库存</th><th>状态</th><th className="actions">操作</th></tr></thead>
              <tbody>
                {books.data!.items.map((book) => {
                  const stockStatus = !book.active ? "inactive" : book.availableCopies > 0 ? "available" : "unavailable"
                  return (
                    <tr key={book.id}>
                      <td><button
  className="text-action"
  onClick={() => navigate(`/books/${book.id}`)}
>
  {book.title}
</button><small>{book.author} · <span className="mono">{book.isbn}</span></small></td>
                      <td>{book.category || "未分类"}</td>
                      <td>{book.publisher || "-"}<small>{book.publishedYear || "年份未知"}</small></td>
                      <td><strong>{book.availableCopies}</strong> / {book.totalCopies}<small>可借 / 总册</small></td>
                      <td><StatusBadge status={stockStatus} /></td>
                      <td className="actions">
                        <IconButton label="编辑图书" onClick={() => openEdit(book)}><Pencil size={16} /></IconButton>
                        <IconButton label="删除图书" className="danger" onClick={() => confirmDelete(book)}><Trash2 size={16} /></IconButton>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
          <Pagination page={page} pageSize={books.data!.pageSize} total={books.data!.total} onChange={setPage} />
        </>
      )}

      <Modal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title={editing ? "编辑图书" : "新增图书"}
        description={editing ? "调整基本信息、馆藏数量或启用状态" : "录入一本新图书及初始馆藏数量"}
        footer={
          <>
            <button className="button button-secondary" onClick={() => setModalOpen(false)}>取消</button>
            <PrimaryButton type="submit" form="book-form" loading={save.isPending}>保存</PrimaryButton>
          </>
        }
      >
        <form id="book-form" className="form-grid" onSubmit={submit}>
          <FormField label="书名" required><input required value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} /></FormField>
          <FormField label="作者" required><input required value={form.author} onChange={(e) => setForm({ ...form, author: e.target.value })} /></FormField>
          <FormField label="ISBN" required><input required className="mono" value={form.isbn} onChange={(e) => setForm({ ...form, isbn: e.target.value })} /></FormField>
          <FormField label="分类"><input value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value })} /></FormField>
          <FormField label="出版社"><input value={form.publisher} onChange={(e) => setForm({ ...form, publisher: e.target.value })} /></FormField>
          <FormField label="出版年份"><input type="number" min="1000" max="9999" value={form.publishedYear ?? ""} onChange={(e) => setForm({ ...form, publishedYear: e.target.value ? Number(e.target.value) : null })} /></FormField>
          <FormField label="馆藏总册" required><input required type="number" min="0" value={form.totalCopies} onChange={(e) => setForm({ ...form, totalCopies: Number(e.target.value) })} /></FormField>
          <label className="toggle-field"><input type="checkbox" checked={form.active} onChange={(e) => setForm({ ...form, active: e.target.checked })} /><span>启用图书</span></label>
          {save.isError && <p className="form-error form-wide">{errorMessage(save.error)}</p>}
        </form>
      </Modal>
    </div>
  )
}
