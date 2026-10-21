# 知藏图书管理系统

面向小型图书馆和团队资料室的前后端分离管理系统，提供馆藏、读者、借阅、归还和逾期管理。

## 技术栈

- Web：React 18、TypeScript、Vite、TanStack Query、Tailwind CSS、Nginx
- API：Go 1.24、Chi、pgx、JWT Cookie、bcrypt
- 数据库：PostgreSQL 16
- 部署：Docker Compose

## Docker 启动

```bash
cp .env.example .env
# 修改 .env 中的数据库口令、JWT 密钥和管理员密码
./scripts/build-release.sh
docker compose up -d --build
```

默认访问地址：`http://localhost:8090`。

查看状态和日志：

```bash
docker compose ps
docker compose logs -f api
```

停止服务不会删除数据：

```bash
docker compose down
```

只有执行 `docker compose down -v` 才会删除 PostgreSQL 数据卷。

## 本地开发

前端：

```bash
npm install
npm run dev
```

后端需要可用的 PostgreSQL，并设置环境变量：

```bash
cd api
export DATABASE_URL='postgres://library:library@localhost:5432/library?sslmode=disable'
export JWT_SECRET='replace-with-at-least-32-random-characters'
export ADMIN_PASSWORD='replace-with-at-least-8-characters'
go run ./cmd/server
```

开发时可在 `vite.config.ts` 中配置 `/api` 代理到 `http://localhost:8080`。

## 验证

```bash
npm run check
npm run lint
npm run build
cd api && go test ./... && go vet ./...
```

`build-release.sh` 会生成 Linux amd64 API 二进制和前端静态资源，再由 Docker 组装轻量运行镜像。生产部署会在 API 启动时自动执行 `api/migrations` 下的 SQL；管理员账号和密码以 `.env` 为准，修改后重启 API 即可轮换密码。
