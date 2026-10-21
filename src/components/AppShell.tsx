import { useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom"
import {
  BookOpen,
  LayoutDashboard,
  Library,
  LogOut,
  Menu,
  Repeat2,
  Users,
  X,
} from "lucide-react"
import { api } from "@/lib/api"
import type { User } from "@/types"
import { IconButton } from "@/components/ui"

const navItems = [
  { to: "/", label: "数据看板", icon: LayoutDashboard },
  { to: "/books", label: "图书管理", icon: BookOpen },
  { to: "/members", label: "读者管理", icon: Users },
  { to: "/loans", label: "借阅管理", icon: Repeat2 },
]

const titles: Record<string, { title: string; description: string }> = {
  "/": { title: "数据看板", description: "馆藏与借阅状态概览" },
  "/books": { title: "图书管理", description: "维护馆藏信息和可借库存" },
  "/members": { title: "读者管理", description: "维护读者资料和借阅资格" },
  "/loans": { title: "借阅管理", description: "办理借阅、归还与逾期跟踪" },
}

export default function AppShell({ user }: { user: User }) {
  const [menuOpen, setMenuOpen] = useState(false)
  const location = useLocation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
const heading = location.pathname.startsWith("/books/")
  ? { title: "图书详情", description: "查看图书与馆藏信息" }
  : titles[location.pathname] || titles["/"]
  const logout = useMutation({
    mutationFn: api.logout,
    onSettled: () => {
      queryClient.clear()
      navigate("/login", { replace: true })
    },
  })

  return (
    <div className="app-shell">
      {menuOpen && <button className="nav-scrim" aria-label="关闭导航" onClick={() => setMenuOpen(false)} />}
      <aside className={`sidebar ${menuOpen ? "is-open" : ""}`}>
        <div className="brand">
          <span className="brand-mark"><Library size={22} /></span>
          <div>
            <strong>知藏</strong>
            <span>图书管理系统</span>
          </div>
          <IconButton label="关闭导航" className="sidebar-close" onClick={() => setMenuOpen(false)}>
            <X size={19} />
          </IconButton>
        </div>
        <nav>
          {navItems.map(({ to, label, icon: Icon }) => (
            <NavLink key={to} to={to} end={to === "/"} onClick={() => setMenuOpen(false)}>
              <Icon size={18} />
              <span>{label}</span>
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-account">
          <span className="avatar">{user.username.slice(0, 1).toUpperCase()}</span>
          <div>
            <strong>{user.username}</strong>
            <span>系统管理员</span>
          </div>
          <IconButton label="退出登录" onClick={() => logout.mutate()}>
            <LogOut size={17} />
          </IconButton>
        </div>
      </aside>
      <main className="main-area">
        <header className="topbar">
          <IconButton label="打开导航" className="menu-button" onClick={() => setMenuOpen(true)}>
            <Menu size={20} />
          </IconButton>
          <div>
            <h1>{heading.title}</h1>
            <p>{heading.description}</p>
          </div>
          <span className="date-chip">
            {new Intl.DateTimeFormat("zh-CN", { month: "long", day: "numeric", weekday: "short" }).format(new Date())}
          </span>
        </header>
        <div className="page-content">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
