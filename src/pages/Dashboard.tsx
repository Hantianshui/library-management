import { useQuery } from "@tanstack/react-query"
import { BookCopy, BookOpenCheck, Clock3, Users } from "lucide-react"
import { api, errorMessage } from "@/lib/api"
import { PageState, StatusBadge } from "@/components/ui"

const dateTime = new Intl.DateTimeFormat("zh-CN", {
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
})

export default function Dashboard() {
  const dashboard = useQuery({ queryKey: ["dashboard"], queryFn: api.dashboard })

  if (dashboard.isLoading) return <PageState loading />
  if (dashboard.isError) return <PageState error={errorMessage(dashboard.error)} />
  const data = dashboard.data!
  const borrowedRate = data.totalCopies ? Math.round((data.activeLoans / data.totalCopies) * 100) : 0

  return (
    <div className="dashboard">
      <section className="metric-strip" aria-label="核心指标">
        <article>
          <span className="metric-icon green"><BookCopy size={19} /></span>
          <div><span>馆藏总册</span><strong>{data.totalCopies}</strong><small>{data.bookTitles} 种图书</small></div>
        </article>
        <article>
          <span className="metric-icon blue"><BookOpenCheck size={19} /></span>
          <div><span>当前可借</span><strong>{data.availableCopies}</strong><small>借出率 {borrowedRate}%</small></div>
        </article>
        <article>
          <span className="metric-icon amber"><Clock3 size={19} /></span>
          <div><span>借阅中</span><strong>{data.activeLoans}</strong><small className={data.overdueLoans ? "danger-text" : ""}>{data.overdueLoans} 条逾期</small></div>
        </article>
        <article>
          <span className="metric-icon gray"><Users size={19} /></span>
          <div><span>启用读者</span><strong>{data.activeMembers}</strong><small>当前有效账号</small></div>
        </article>
      </section>

      <section className="dashboard-grid">
        <div className="content-section recent-loans">
          <div className="section-heading">
            <div><h2>最近借阅</h2><p>最新的借阅和归还状态</p></div>
          </div>
          {data.recentLoans.length === 0 ? (
            <PageState empty="暂无借阅记录" />
          ) : (
            <div className="table-wrap">
              <table>
                <thead><tr><th>图书</th><th>读者</th><th>应还日期</th><th>状态</th></tr></thead>
                <tbody>
                  {data.recentLoans.map((loan) => (
                    <tr key={loan.id}>
                      <td><strong>{loan.bookTitle}</strong><small>{loan.bookIsbn}</small></td>
                      <td>{loan.memberName}<small>{loan.memberNo}</small></td>
                      <td className="mono">{loan.dueDate}</td>
                      <td><StatusBadge status={loan.status} /></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        <div className="content-section activity-list">
          <div className="section-heading">
            <div><h2>最近动态</h2><p>关键数据变更记录</p></div>
          </div>
          {data.recentActivity.length === 0 ? (
            <PageState empty="暂无操作记录" />
          ) : (
            <ul>
              {data.recentActivity.map((item) => (
                <li key={item.id}>
                  <span className={`activity-dot type-${item.entityType}`} />
                  <div><strong>{item.summary}</strong><time>{dateTime.format(new Date(item.createdAt))}</time></div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </section>
    </div>
  )
}
