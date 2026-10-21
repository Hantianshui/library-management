import { type FormEvent, useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Eye, EyeOff, Library } from "lucide-react"
import { useNavigate } from "react-router-dom"
import { api, errorMessage } from "@/lib/api"
import { IconButton, PrimaryButton } from "@/components/ui"

export default function Login() {
  const [username, setUsername] = useState("admin")
  const [password, setPassword] = useState("")
  const [showPassword, setShowPassword] = useState(false)
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const login = useMutation({
    mutationFn: () => api.login(username, password),
    onSuccess: (user) => {
      queryClient.setQueryData(["me"], user)
      navigate("/", { replace: true })
    },
  })

  const submit = (event: FormEvent) => {
    event.preventDefault()
    if (username.trim() && password) login.mutate()
  }

  return (
    <main className="login-page">
      <section className="login-panel">
        <div className="login-brand">
          <span><Library size={25} /></span>
          <div>
            <strong>知藏</strong>
            <p>图书管理系统</p>
          </div>
        </div>
        <div className="login-heading">
          <h1>管理员登录</h1>
          <p>登录后管理馆藏、读者与借阅记录</p>
        </div>
        <form onSubmit={submit}>
          <div className="form-field">
            <label htmlFor="username">用户名</label>
            <input
              id="username"
              autoComplete="username"
              value={username}
              onChange={(event) => setUsername(event.target.value)}
              placeholder="请输入用户名"
            />
          </div>
          <div className="form-field">
            <label htmlFor="password">密码</label>
            <div className="password-input">
              <input
                id="password"
                type={showPassword ? "text" : "password"}
                autoComplete="current-password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                placeholder="请输入密码"
              />
              <IconButton label={showPassword ? "隐藏密码" : "显示密码"} type="button" onClick={() => setShowPassword(!showPassword)}>
                {showPassword ? <EyeOff size={17} /> : <Eye size={17} />}
              </IconButton>
            </div>
          </div>
          {login.isError && <p className="form-error" role="alert">{errorMessage(login.error)}</p>}
          <PrimaryButton type="submit" loading={login.isPending} disabled={!username.trim() || !password}>
            登录
          </PrimaryButton>
        </form>
        <p className="login-footer">内部资料管理 · 数据仅保存在当前部署环境</p>
      </section>
      <aside className="login-context">
        <div>
          <span className="context-kicker">LIBRARY OPERATIONS</span>
          <h2>让每一本书的去向清晰可查</h2>
          <p>集中维护馆藏库存、读者状态和借阅流转，及时发现逾期记录。</p>
        </div>
        <dl>
          <div><dt>01</dt><dd>馆藏信息统一管理</dd></div>
          <div><dt>02</dt><dd>借阅库存自动联动</dd></div>
          <div><dt>03</dt><dd>逾期状态实时识别</dd></div>
        </dl>
      </aside>
    </main>
  )
}
