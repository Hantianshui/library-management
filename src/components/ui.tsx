import type { ButtonHTMLAttributes, ReactNode } from "react"
import { ChevronLeft, ChevronRight, LoaderCircle, X } from "lucide-react"
import { cn } from "@/lib/utils"

export function IconButton({
  label,
  className,
  children,
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { label: string; children: ReactNode }) {
  return (
    <button className={cn("icon-button", className)} title={label} aria-label={label} {...props}>
      {children}
    </button>
  )
}

export function PrimaryButton({
  loading,
  children,
  className,
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { loading?: boolean }) {
  return (
    <button className={cn("button button-primary", className)} disabled={loading || props.disabled} {...props}>
      {loading && <LoaderCircle size={16} className="animate-spin" />}
      {children}
    </button>
  )
}

export function Modal({
  title,
  description,
  open,
  onClose,
  children,
  footer,
}: {
  title: string
  description?: string
  open: boolean
  onClose: () => void
  children: ReactNode
  footer?: ReactNode
}) {
  if (!open) return null
  return (
    <div className="modal-backdrop" role="presentation" onMouseDown={onClose}>
      <section
        className="modal-panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby="modal-title"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <header className="modal-header">
          <div>
            <h2 id="modal-title">{title}</h2>
            {description && <p>{description}</p>}
          </div>
          <IconButton label="关闭" onClick={onClose}>
            <X size={18} />
          </IconButton>
        </header>
        <div className="modal-content">{children}</div>
        {footer && <footer className="modal-footer">{footer}</footer>}
      </section>
    </div>
  )
}

export function FormField({
  label,
  required,
  children,
}: {
  label: string
  required?: boolean
  children: ReactNode
}) {
  return (
    <label className="form-field">
      <span>
        {label}
        {required && <b aria-hidden="true"> *</b>}
      </span>
      {children}
    </label>
  )
}

export function Pagination({
  page,
  pageSize,
  total,
  onChange,
}: {
  page: number
  pageSize: number
  total: number
  onChange: (page: number) => void
}) {
  const pages = Math.max(1, Math.ceil(total / pageSize))
  return (
    <div className="pagination">
      <span>
        共 {total} 条，第 {page} / {pages} 页
      </span>
      <div>
        <IconButton label="上一页" disabled={page <= 1} onClick={() => onChange(page - 1)}>
          <ChevronLeft size={17} />
        </IconButton>
        <IconButton label="下一页" disabled={page >= pages} onClick={() => onChange(page + 1)}>
          <ChevronRight size={17} />
        </IconButton>
      </div>
    </div>
  )
}

export function PageState({
  loading,
  error,
  empty,
}: {
  loading?: boolean
  error?: string
  empty?: string
}) {
  if (loading) {
    return (
      <div className="page-state">
        <LoaderCircle size={22} className="animate-spin" />
        <span>正在加载</span>
      </div>
    )
  }
  return <div className={cn("page-state", error && "is-error")}>{error || empty}</div>
}

export function StatusBadge({ status }: { status: string }) {
  const labels: Record<string, string> = {
    borrowed: "借阅中",
    returned: "已归还",
    overdue: "已逾期",
    active: "启用",
    inactive: "停用",
    available: "可借",
    unavailable: "无库存",
  }
  return <span className={`status-badge status-${status}`}>{labels[status] || status}</span>
}
