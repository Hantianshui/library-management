import { useQuery } from "@tanstack/react-query"
import { ArrowLeft } from "lucide-react"
import { useNavigate, useParams } from "react-router-dom"
import { PageState } from "@/components/ui"
import { api, errorMessage } from "@/lib/api"

export default function BookDetails() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const book = useQuery({
    queryKey: ["book", id],
    queryFn: () => api.book(id!),
    enabled: Boolean(id),
    retry: false,
  })

  if (book.isLoading) {
    return <PageState loading />
  }

  if (book.isError) {
    return <PageState error={errorMessage(book.error)} />
  }

  return (
    <div className="content-section">
      <div className="toolbar">
        <button
          className="text-action"
          onClick={() => navigate("/books")}
        >
          <ArrowLeft size={16} />
          返回图书列表
        </button>
      </div>

      <pre style={{ padding: 20, overflow: "auto" }}>
        {JSON.stringify(book.data, null, 2)}
      </pre>
    </div>
  )
}