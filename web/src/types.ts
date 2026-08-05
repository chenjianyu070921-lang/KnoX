/** 文档列表项 */
export interface DocItem {
  docId: string
  title: string
  docType: string
  fileUrl: string
  createdAt: string
  version: number
}

/** 文档列表响应 */
export interface DocListResp {
  items: DocItem[]
  total: number
  page: number
  pageSize: number
}

/** 文档详情（含全文） */
export interface DocDetailResp extends DocItem {
  content: string
}
