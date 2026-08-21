/**
 * 将飞牛媒体接口返回的图片地址转换为后端代理地址。
 *
 * 飞牛的图片 URL 需要携带登录 Cookie 才能访问，浏览器无法直接获取，
 * 因此图片统一走 Go 后端 `/api/trimmedia/img` 代理，由后端复用登录会话拉取。
 *
 * 入参可能是飞牛相对路径（如 `/api/v1/sys/img/xxx`）或完整 URL
 * （如 `http://host/api/v1/sys/img/xxx?w=256`），这里统一提取路径部分
 * 交给后端代理。
 */
export function proxyImage(src?: string | null): string {
  if (!src) return ''

  let path = src
  if (/^https?:\/\//i.test(src)) {
    try {
      const u = new URL(src)
      path = u.pathname + u.search
    } catch {
      path = src
    }
  }

  // 与 request.ts 的 baseURL('./api') 保持一致，用相对路径适配子路径部署
  return `./api/trimmedia/img?path=${encodeURIComponent(path)}`
}