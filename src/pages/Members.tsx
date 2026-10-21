import { type FormEvent, useEffect, useState } from "react"
import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Mail, Pencil, Phone, Plus, Search, Trash2 } from "lucide-react"
import { api, errorMessage } from "@/lib/api"
import type { Member, MemberInput } from "@/types"
import { useDebouncedValue } from "@/hooks/useDebouncedValue"
import { FormField, IconButton, Modal, PageState, Pagination, PrimaryButton, StatusBadge } from "@/components/ui"

const emptyMember: MemberInput = { memberNo: "", name: "", phone: "", email: "", active: true }

export default function Members() {
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState("")
  const [status, setStatus] = useState("")
  const [editing, setEditing] = useState<Member | null>(null)
  const [form, setForm] = useState<MemberInput>(emptyMember)
  const [modalOpen, setModalOpen] = useState(false)
  const [notice, setNotice] = useState("")
  const debouncedSearch = useDebouncedValue(search)
  const queryClient = useQueryClient()
  const members = useQuery({
    queryKey: ["members", page, debouncedSearch, status],
    queryFn: () => api.members({ page, search: debouncedSearch, status }),
    placeholderData: keepPreviousData,
  })
  useEffect(() => setPage(1), [debouncedSearch, status])

  const save = useMutation({
    mutationFn: () => editing ? api.updateMember(editing.id, form) : api.createMember(form),
    onSuccess: () => {
      setModalOpen(false)
      setNotice(editing ? "读者信息已更新" : "读者已添加")
      queryClient.invalidateQueries({ queryKey: ["members"] })
      queryClient.invalidateQueries({ queryKey: ["dashboard"] })
    },
  })
  const remove = useMutation({
    mutationFn: api.deleteMember,
    onSuccess: () => {
      setNotice("读者已删除")
      queryClient.invalidateQueries({ queryKey: ["members"] })
      queryClient.invalidateQueries({ queryKey: ["dashboard"] })
    },
  })

  const openCreate = () => {
    setEditing(null)
    setForm(emptyMember)
    setModalOpen(true)
  }
  const openEdit = (member: Member) => {
    setEditing(member)
    setForm({
      memberNo: member.memberNo,
      name: member.name,
      phone: member.phone,
      email: member.email,
      active: member.active,
    })
    setModalOpen(true)
  }
  const submit = (event: FormEvent) => {
    event.preventDefault()
    save.mutate()
  }
  const confirmDelete = (member: Member) => {
    if (window.confirm(`确认删除读者“${member.name}”？此操作不可撤销。`)) remove.mutate(member.id)
  }

  return (
    <div className="content-section">
      <div className="toolbar">
        <div className="search-box">
          <Search size={17} />
          <input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="搜索姓名、编号或联系方式" />
        </div>
        <select value={status} onChange={(event) => setStatus(event.target.value)} aria-label="读者状态">
          <option value="">全部状态</option>
          <option value="active">启用</option>
          <option value="inactive">停用</option>
        </select>
        <span className="toolbar-spacer" />
        <PrimaryButton onClick={openCreate}><Plus size={17} />新增读者</PrimaryButton>
      </div>
      {notice && <div className="inline-notice" role="status">{notice}<button onClick={() => setNotice("")}>关闭</button></div>}
      {remove.isError && <div className="inline-notice is-error">{errorMessage(remove.error)}</div>}

      {members.isLoading ? <PageState loading /> : members.isError ? (
        <PageState error={errorMessage(members.error)} />
      ) : members.data!.items.length === 0 ? (
        <PageState empty="没有找到符合条件的读者" />
      ) : (
        <>
          <div className="table-wrap">
            <table>
              <thead><tr><th>读者</th><th>联系方式</th><th>当前借阅</th><th>状态</th><th className="actions">操作</th></tr></thead>
              <tbody>
                {members.data!.items.map((member) => (
                  <tr key={member.id}>
                    <td><strong>{member.name}</strong><small className="mono">{member.memberNo}</small></td>
                    <td>
                      <span className="contact-line"><Phone size={14} />{member.phone || "-"}</span>
                      <small className="contact-line"><Mail size={14} />{member.email || "-"}</small>
                    </td>
                    <td><strong>{member.activeLoans}</strong> / 5<small>在借上限</small></td>
                    <td><StatusBadge status={member.active ? "active" : "inactive"} /></td>
                    <td className="actions">
                      <IconButton label="编辑读者" onClick={() => openEdit(member)}><Pencil size={16} /></IconButton>
                      <IconButton label="删除读者" className="danger" onClick={() => confirmDelete(member)}><Trash2 size={16} /></IconButton>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <Pagination page={page} pageSize={members.data!.pageSize} total={members.data!.total} onChange={setPage} />
        </>
      )}

      <Modal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title={editing ? "编辑读者" : "新增读者"}
        description={editing ? "调整读者资料或借阅资格" : "录入读者资料并设置借阅资格"}
        footer={
          <>
            <button className="button button-secondary" onClick={() => setModalOpen(false)}>取消</button>
            <PrimaryButton type="submit" form="member-form" loading={save.isPending}>保存</PrimaryButton>
          </>
        }
      >
        <form id="member-form" className="form-grid" onSubmit={submit}>
          <FormField label="读者姓名" required><input required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></FormField>
          <FormField label="读者编号" required><input required className="mono" value={form.memberNo} onChange={(e) => setForm({ ...form, memberNo: e.target.value })} /></FormField>
          <FormField label="手机号码"><input type="tel" value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} /></FormField>
          <FormField label="电子邮箱"><input type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} /></FormField>
          <label className="toggle-field form-wide"><input type="checkbox" checked={form.active} onChange={(e) => setForm({ ...form, active: e.target.checked })} /><span>允许该读者借阅图书</span></label>
          {save.isError && <p className="form-error form-wide">{errorMessage(save.error)}</p>}
        </form>
      </Modal>
    </div>
  )
}
