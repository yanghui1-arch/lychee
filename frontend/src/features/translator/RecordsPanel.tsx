import { useState } from 'react'
import {
  Bookmark,
  Copy,
  Funnel,
  MoreHorizontal,
  Search,
  Trash2,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { LANGUAGES, RESULT_TEXT, type Collection, type Entry } from './model'

function timeLabel(timestamp: string) {
  const date = new Date(timestamp)
  const today = new Date()
  const yesterday = new Date(today)
  yesterday.setDate(today.getDate() - 1)
  const day = date.toDateString()
  const prefix =
    day === today.toDateString()
      ? '今天'
      : day === yesterday.toDateString()
        ? '昨天'
        : date.toLocaleDateString('zh-CN', { month: 'numeric', day: 'numeric' })
  return `${prefix} ${date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false })}`
}

type Props = {
  history: Entry[]
  favorites: Entry[]
  onRestore: (entry: Entry) => void
  onCopy: (text: string) => void
  onFavorite: (entry: Entry) => void
  isFavorite: (entry: Entry) => boolean
  onRemove: (id: string, collection: Collection) => void
}

export function RecordsPanel({
  history,
  favorites,
  onRestore,
  onCopy,
  onFavorite,
  isFavorite,
  onRemove,
}: Props) {
  const [collection, setCollection] = useState<Collection>('history')
  const [query, setQuery] = useState('')
  const [language, setLanguage] = useState('all')
  const entries = (collection === 'history' ? history : favorites).filter(
    (entry) =>
      (language === 'all' || entry.source === language) &&
      `${entry.text} ${entry.result ?? RESULT_TEXT}`
        .toLocaleLowerCase()
        .includes(query.trim().toLocaleLowerCase()),
  )
  return (
    <Tabs
      value={collection}
      onValueChange={(value) => setCollection(value as Collection)}
      className="records-panel"
      aria-label="记录与收藏"
    >
      <div className="records-toolbar">
        <TabsList className="records-tabs" aria-label="记录分类">
          <TabsTrigger value="history">最近记录</TabsTrigger>
          <TabsTrigger value="favorites">收藏</TabsTrigger>
        </TabsList>
        <div className="records-tools">
          <div className="records-search">
            <Search aria-hidden="true" />
            <Input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="搜索历史记录…"
              aria-label="搜索记录"
            />
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button variant="ghost" size="icon" className="filter-button" />
              }
              aria-label="筛选原文语言"
              title="筛选原文语言"
              data-active={language !== 'all'}
            >
              <Funnel />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuRadioGroup
                value={language}
                onValueChange={setLanguage}
              >
                <DropdownMenuRadioItem value="all">
                  全部语言
                </DropdownMenuRadioItem>
                {LANGUAGES.map((item) => (
                  <DropdownMenuRadioItem value={item.value} key={item.value}>
                    {item.label}
                  </DropdownMenuRadioItem>
                ))}
              </DropdownMenuRadioGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
      <TabsContent value={collection} className="records-content">
        <Table
          className="records-table"
          aria-label={collection === 'history' ? '最近记录' : '收藏记录'}
        >
          <colgroup>
            <col className="record-source" />
            <col className="record-result" />
            <col className="record-time" />
            <col className="record-actions" />
          </colgroup>
          <TableHeader className="sr-only">
            <TableRow>
              <TableHead>原文</TableHead>
              <TableHead>译文</TableHead>
              <TableHead>时间</TableHead>
              <TableHead>操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {entries.length ? (
              entries.map((entry) => (
                <TableRow key={entry.id}>
                  <TableCell className="record-source">
                    <Button
                      variant="ghost"
                      className="record-restore"
                      onClick={() => onRestore(entry)}
                      title={entry.text}
                    >
                      {entry.text}
                    </Button>
                  </TableCell>
                  <TableCell
                    className="record-result"
                    title={entry.result ?? RESULT_TEXT}
                  >
                    <span>{entry.result ?? RESULT_TEXT}</span>
                  </TableCell>
                  <TableCell className="record-time">
                    <time dateTime={entry.createdAt}>
                      {timeLabel(entry.createdAt)}
                    </time>
                  </TableCell>
                  <TableCell className="record-actions">
                    <DropdownMenu>
                      <DropdownMenuTrigger
                        render={<Button variant="ghost" size="icon" />}
                        aria-label={`记录选项：${entry.text}`}
                      >
                        <MoreHorizontal />
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem onClick={() => onRestore(entry)}>
                          恢复原文
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => onCopy(entry.text)}>
                          <Copy />
                          复制原文
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => onFavorite(entry)}>
                          <Bookmark />
                          {isFavorite(entry) ? '取消收藏' : '收藏记录'}
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          onClick={() => onRemove(entry.id, collection)}
                          variant="destructive"
                        >
                          <Trash2 />
                          {collection === 'favorites' ? '移除收藏' : '删除记录'}
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={4} className="records-empty">
                  {query || language !== 'all'
                    ? '未找到匹配的记录'
                    : collection === 'favorites'
                      ? '还没有收藏的记录'
                      : '还没有历史记录'}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TabsContent>
    </Tabs>
  )
}
