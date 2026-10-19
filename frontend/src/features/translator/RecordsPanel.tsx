import { cn } from '@/lib/utils'
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
      className="records-panel min-h-0 gap-2 pt-2.5 px-3 pb-2 bg-card border border-border rounded-[14px] [@media(max-width:760px)]:min-h-[210px]"
      aria-label="记录与收藏"
    >
      <div className="h-[30px] shrink-0 flex items-center justify-between gap-3">
        <TabsList
          className={cn(
            "h-[30px]! bg-transparent p-0 gap-[3px] [&_[data-slot='tabs-trigger']]:py-0 [&_[data-slot='tabs-trigger']]:px-3",
            "[&_[data-slot='tabs-trigger']]:h-[30px] [&_[data-slot='tabs-trigger']]:text-[12px] [&_[data-slot='tabs-trigger']]:font-medium",
            "[&_[data-slot='tabs-trigger']]:rounded-[12px] [&_[data-active]]:bg-muted [&_[data-active]]:text-foreground",
            "[&_[data-active]]:shadow-none [@media(max-width:760px)]:[&_[data-slot='tabs-trigger']]:py-0",
            "[@media(max-width:760px)]:[&_[data-slot='tabs-trigger']]:px-2",
          )}
          aria-label="记录分类"
        >
          <TabsTrigger value="history">最近记录</TabsTrigger>
          <TabsTrigger value="favorites">收藏</TabsTrigger>
        </TabsList>
        <div className="flex gap-[7px]">
          <div
            className={cn(
              'relative w-[174px] [&_svg]:absolute [&_svg]:[left:10px] [&_svg]:[top:9px] [&_svg]:w-[12px] [&_svg]:h-[12px]',
              '[&_svg]:text-muted-foreground [&_svg]:pointer-events-none [&_input]:w-full [&_input]:h-[30px] [&_input]:pt-0 [&_input]:pr-3',
              '[&_input]:pb-0 [&_input]:pl-[29px] [&_input]:border [&_input]:border-border [&_input]:rounded-[12px] [&_input]:bg-background',
              '[&_input]:shadow-none [&_input]:text-[11px] [@media(max-width:760px)]:w-[140px]',
            )}
          >
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
                <Button
                  variant="ghost"
                  size="icon"
                  className="w-[30px] h-[30px] border border-border rounded-[12px] bg-background text-[color:var(--subtle-foreground)] [&_svg]:w-[13px] [&_svg]:h-[13px] [&[data-active='true']]:bg-primary [&[data-active='true']]:text-primary-foreground [&[data-active='true']]:border-foreground"
                />
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
      <TabsContent value={collection} className="min-h-0 overflow-y-auto">
        <Table
          className="[table-layout:fixed] text-[12px] [&_tr]:h-[34px] [&_tr]:border-border [&_tr:last-child]:border-0 [&_td]:py-[3px] [&_td]:px-2 [@media(max-width:1000px)]:[&_tr]:h-[30px] [@media(max-width:1000px)]:text-[11px]"
          aria-label={collection === 'history' ? '最近记录' : '收藏记录'}
        >
          <colgroup>
            <col className="w-[36%] [@media(max-width:760px)]:w-[42%]" />
            <col className="w-[44%] text-[color:var(--subtle-foreground)] [&_span]:block [&_span]:overflow-hidden [&_span]:text-ellipsis [@media(max-width:760px)]:w-[42%]" />
            <col className="w-[15%] text-right text-muted-foreground text-[10px] [@media(max-width:760px)]:hidden" />
            <col className="w-[5%] text-right [&_button]:w-[22px] [&_button]:h-[22px] [&_button]:text-[color:var(--subtle-foreground)] [&_svg]:w-[14px] [&_svg]:h-[14px] [@media(max-width:760px)]:w-[16%]" />
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
                  <TableCell className="w-[36%] [@media(max-width:760px)]:w-[42%]">
                    <Button
                      variant="ghost"
                      className="w-full h-auto p-0 block border-0 rounded-[2px] text-left text-[length:inherit] font-normal whitespace-nowrap overflow-hidden text-ellipsis"
                      onClick={() => onRestore(entry)}
                      title={entry.text}
                    >
                      {entry.text}
                    </Button>
                  </TableCell>
                  <TableCell
                    className="w-[44%] text-[color:var(--subtle-foreground)] [&_span]:block [&_span]:overflow-hidden [&_span]:text-ellipsis [@media(max-width:760px)]:w-[42%]"
                    title={entry.result ?? RESULT_TEXT}
                  >
                    <span>{entry.result ?? RESULT_TEXT}</span>
                  </TableCell>
                  <TableCell className="w-[15%] text-right text-muted-foreground text-[10px] [@media(max-width:760px)]:hidden">
                    <time dateTime={entry.createdAt}>
                      {timeLabel(entry.createdAt)}
                    </time>
                  </TableCell>
                  <TableCell className="w-[5%] text-right [&_button]:w-[22px] [&_button]:h-[22px] [&_button]:text-[color:var(--subtle-foreground)] [&_svg]:w-[14px] [&_svg]:h-[14px] [@media(max-width:760px)]:w-[16%]">
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
                <TableCell
                  colSpan={4}
                  className="h-[120px] text-center text-muted-foreground"
                >
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
