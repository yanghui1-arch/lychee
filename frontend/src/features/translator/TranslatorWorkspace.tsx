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

export function TranslatorWorkspace() {
  const translator = useTranslator()
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
      className="workspace"
      aria-label="荔枝翻译工作区"
      style={
        {
          '--editor-font-size': `${translator.preferences.textSize}px`,
        } as CSSProperties
      }
    >
      <WorkspaceHeader />
      <TabsContent value={page} className="workspace-body" data-page={page}>
        {(page === 'translate' || page === 'interpret') && (
          <div ref={panelsRef} className="translation-panels">
            <section
              className="translation-panel source-panel"
              aria-label="原文"
            >
              <div className="panel-toolbar">
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
              <div className="source-editor">
                <Textarea
                  ref={sourceRef}
                  className="source-field"
                  aria-label="输入原文"
                  spellCheck={false}
                  maxLength={MAX_LENGTH}
                  value={translator.text}
                  placeholder={
                    page === 'interpret'
                      ? '同传内容将在这里显示…'
                      : '输入你想翻译的内容…'
                  }
                  onChange={(event) => {
                    translator.patch({ text: event.target.value })
                    setLiked(false)
                  }}
                  onCompositionStart={() => setComposing(true)}
                  onCompositionEnd={() => setComposing(false)}
                />
                <div className="source-footer">
                  <span
                    className="character-count"
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
              className="translation-panel result-panel"
              aria-label="翻译结果"
            >
              <div className="panel-toolbar">
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
                className="result-text"
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
              <div className="result-footer">
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
                  {copied ? <Check /> : <Copy />}
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
              className="language-swap text-muted-foreground hover:text-foreground"
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
        {page === 'image' && (
          <section className="page-placeholder" aria-label="图译">
            <h1>图译</h1>
            <p>图片翻译内容将在这里显示。</p>
          </section>
        )}
        {page === 'subtitle' && (
          <section className="page-placeholder" aria-label="字幕">
            <h1>字幕</h1>
            <p>字幕内容将在这里显示。</p>
          </section>
        )}
      </TabsContent>
    </Tabs>
  )
}
