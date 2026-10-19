import { cn } from '@/lib/utils'
import { useState, type CSSProperties } from 'react'
import { ArrowRightLeft, Check, Copy } from 'lucide-react'
import { AnimatedCounter } from '@/components/ui/animated-counter'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { LikeButton } from '@/components/ui/like-button'
import { useToastManager } from '@/components/ui/toast'
import { copyText } from '@/lib/desktop'
import { LanguageSelect } from './LanguageSelect'
import { WorkspaceHeader } from './WorkspaceHeader'
import { MAX_LENGTH, type Page } from './model'
import { useTranslator } from './useTranslator'
import { useTranslation } from './useTranslation'
import { useTranslationPanelHeight } from './useTranslationPanelHeight'
import { SubtitleWorkspace } from '@/features/subtitle/SubtitleWorkspace'
import { useAudioCapture } from '@/features/subtitle/useAudioCapture'
import { InterpretWorkspace } from '@/features/interpret/InterpretWorkspace'

export function TranslatorWorkspace() {
  const translator = useTranslator()
  const audio = useAudioCapture()
  const [page, setPage] = useState<Page>('translate')
  const [composing, setComposing] = useState(false)
  const { panelsRef, sourceRef } = useTranslationPanelHeight(
    translator.text,
    page,
    translator.preferences.textSize,
  )
  const translation = useTranslation(
    {
      text: translator.text,
      source_language: translator.source,
      target_language: translator.target,
    },
    page === 'translate',
    composing,
  )
  const resultText = {
    idle: '',
    waiting: '等待输入完成…',
    loading: '正在翻译…',
    success: translation.text,
    error: translation.error,
  }[translation.status]
  const toastManager = useToastManager()
  const copied = toastManager.toasts.some(
    (toast) => toast.title === '已复制' && toast.transitionStatus !== 'ending',
  )
  const [liked, setLiked] = useState(false)
  const notify = (title: string) =>
    toastManager.add({ id: 'workspace-notice', title })
  async function copy(text: string) {
    try {
      await copyText(text)
      notify('已复制')
    } catch {
      notify('复制未成功，请稍后重试')
    }
  }
  return (
    <Tabs
      value={page}
      onValueChange={(value) => {
        setPage(value as Page)
        setComposing(false)
      }}
      className="w-full h-svh min-h-[600px] pt-0 px-3.5 pb-3 gap-0 relative overflow-hidden rounded-[14px] bg-background [@media(max-width:760px)]:h-auto [@media(max-width:760px)]:min-h-svh [@media(max-width:760px)]:pt-0 [@media(max-width:760px)]:px-2.5 [@media(max-width:760px)]:pb-2.5"
      aria-label="荔枝翻译工作区"
      style={
        {
          '--editor-font-size': `${translator.preferences.textSize}px`,
        } as CSSProperties
      }
    >
      <WorkspaceHeader />
      <TabsContent
        value={page}
        className="min-h-0 grid grid-rows-[minmax(0,_1fr)] [@media(max-width:760px)]:grid-rows-[auto] [@media(max-width:760px)]:[&[data-page='history']_.records-panel]:min-h-[calc(100svh_-_100px)] [&[data-page='interpret']]:overflow-y-auto [&[data-page='interpret']]:[scrollbar-gutter:stable]"
        data-page={page}
      >
        {page === 'translate' && (
          <div
            ref={panelsRef}
            className="relative mt-3 grid grid-cols-[1fr_1fr] h-[var(--translation-panel-height,_50%)] gap-4 min-h-0 [@media(max-width:760px)]:grid-cols-[1fr] [@media(max-width:760px)]:h-auto [@media(max-width:760px)]:gap-[9px]"
          >
            <section
              className="source-panel min-w-0 min-h-0 flex flex-col border border-border rounded-[14px] bg-card [@media(max-width:760px)]:min-h-[265px]"
              aria-label="原文"
            >
              <div className="min-h-[52px] flex items-center gap-3 pt-3 px-6 pb-2.5 rounded-[13px_13px_0_0] bg-[var(--panel-toolbar)]">
                <LanguageSelect
                  label="原语言"
                  autoDetect
                  value={translator.source}
                  onChange={(source) => {
                    translator.patch({ source })
                    setLiked(false)
                  }}
                />
              </div>
              <div className="relative flex-1 min-h-0 flex flex-col">
                <Textarea
                  ref={sourceRef}
                  className={cn(
                    'flex-1 min-h-0 w-full [resize:none] [field-sizing:fixed] md:text-[length:var(--editor-font-size)] pt-4.5 px-6 pb-3 border-0',
                    'rounded-[0] text-[length:var(--editor-font-size)] leading-[1.7] bg-transparent dark:bg-transparent shadow-none',
                    'focus-visible:border-0 focus-visible:[outline:none] focus-visible:shadow-none',
                    '[@media(max-width:1000px)]:text-[length:calc(var(--editor-font-size)_*_0.875)]',
                  )}
                  aria-label="输入原文"
                  spellCheck={false}
                  maxLength={MAX_LENGTH}
                  value={translator.text}
                  placeholder="输入你想翻译的内容…"
                  onChange={(event) => {
                    translator.patch({ text: event.target.value })
                    setLiked(false)
                  }}
                  onCompositionStart={() => setComposing(true)}
                  onCompositionEnd={() => setComposing(false)}
                />
                <div className="h-[38px] shrink-0 flex items-end justify-end pt-0 px-6 pb-4.5">
                  <span
                    className="flex items-center gap-[3px] text-[11px] text-muted-foreground tabular-nums"
                    aria-label={`${translator.text.length} 字，最多 ${MAX_LENGTH} 字`}
                  >
                    <AnimatedCounter
                      value={translator.text.length}
                      separator=""
                      duration={0.2}
                    />{' '}
                    / {MAX_LENGTH}
                  </span>
                </div>
              </div>
            </section>
            <section
              className="result-panel min-w-0 min-h-0 flex flex-col border border-border rounded-[14px] bg-card [@media(max-width:760px)]:min-h-[265px]"
              aria-label="翻译结果"
            >
              <div className="min-h-[52px] flex items-center gap-3 pt-3 px-6 pb-2.5 rounded-[13px_13px_0_0] bg-[var(--panel-toolbar)]">
                <LanguageSelect
                  label="目标语言"
                  value={translator.target}
                  onChange={(target) => {
                    translator.patch({ target })
                    setLiked(false)
                  }}
                />
              </div>
              <div
                className={cn(
                  'flex-1 min-h-0 overflow-y-auto my-4.5 mx-6 text-[length:var(--editor-font-size)] leading-[1.6] [&_p]:m-0',
                  "[&_p]:whitespace-pre-wrap [&_p]:[overflow-wrap:anywhere] [&[data-status='waiting']]:text-muted-foreground",
                  "[&[data-status='loading']]:text-muted-foreground",
                  '[@media(max-width:1000px)]:text-[length:calc(var(--editor-font-size)_*_0.875)]',
                )}
                data-status={translation.status}
                aria-live="polite"
                aria-busy={translation.status === 'loading'}
              >
                {resultText && <p>{resultText}</p>}
                {translation.status === 'error' && (
                  <Button
                    variant="outline"
                    size="sm"
                    className="mt-3"
                    onClick={translation.retry}
                  >
                    重试
                  </Button>
                )}
              </div>
              <div
                className={cn(
                  'h-[50px] shrink-0 pt-0 px-6 pb-4.5 flex items-center gap-2 [&_button]:w-[32px] [&_button]:h-[30px] [&_button]:p-0',
                  '[&_button]:border [&_button]:border-border [&_button]:rounded-[12px] [&_button]:bg-background [&_button]:shadow-none',
                  '[&_button]:text-[color:var(--subtle-foreground)] [&_svg]:w-[14px] [&_svg]:h-[14px] [&_svg]:[stroke-width:1.65]',
                  '[&_.t-like]:relative',
                )}
              >
                <Button
                  variant="outline"
                  size="icon"
                  aria-label="复制翻译结果"
                  title="复制翻译结果"
                  disabled={
                    translation.status !== 'success' || !translation.text
                  }
                  onClick={() => void copy(translation.text)}
                >
                  {copied ? (
                    <Check className="size-3.5" />
                  ) : (
                    <Copy className="size-3.5" />
                  )}
                </Button>
                <LikeButton
                  liked={translation.status === 'success' && liked}
                  onLikedChange={setLiked}
                  disabled={
                    translation.status !== 'success' || !translation.text
                  }
                />
              </div>
            </section>
            <Button
              variant="outline"
              size="icon"
              className="text-muted-foreground hover:text-foreground absolute [top:calc(50%_+_26px)] [left:50%] [transform:translate(-50%,_-50%)] z-[1] rounded-[12px] [@media(max-width:760px)]:[top:50%] [@media(max-width:760px)]:[&_svg]:[transform:rotate(90deg)]"
              aria-label="交换原文与译文语言"
              title="交换原文与译文语言"
              disabled={translator.source === 'auto'}
              onClick={() => {
                if (translator.source === 'auto') return
                translator.patch({
                  source: translator.target,
                  target: translator.source,
                })
                setLiked(false)
              }}
            >
              <ArrowRightLeft strokeWidth={1.5} />
            </Button>
          </div>
        )}
        {page === 'interpret' && <InterpretWorkspace />}
        {page === 'subtitle' && <SubtitleWorkspace audio={audio} />}
      </TabsContent>
    </Tabs>
  )
}
