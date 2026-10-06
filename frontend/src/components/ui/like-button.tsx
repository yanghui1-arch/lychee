import type { CSSProperties } from 'react'
import { Heart } from 'lucide-react'
import { Toggle } from './toggle'

// Transitions.dev markup and motion; shadcn/Base UI handles button semantics.
const particles = Array.from({ length: 8 }, (_, index) => {
  const angle = (index * Math.PI) / 4
  return {
    '--px': `calc(var(--like-particle-dist) * ${Math.cos(angle).toFixed(3)})`,
    '--py': `calc(var(--like-particle-dist) * ${Math.sin(angle).toFixed(3)})`,
  } as CSSProperties
})

export function LikeButton({
  liked,
  onLikedChange,
  disabled = false,
}: {
  liked: boolean
  onLikedChange: (liked: boolean) => void
  disabled?: boolean
}) {
  return (
    <Toggle
      variant="outline"
      className={`t-like${liked ? ' is-bursting' : ''}`}
      data-liked={liked}
      pressed={liked}
      onPressedChange={onLikedChange}
      disabled={disabled}
      aria-label="喜欢译文"
      title={liked ? '取消喜欢' : '喜欢译文'}
    >
      <span className="t-like-icon" aria-hidden="true">
        <Heart className="t-like-heart" />
      </span>
      <span className="t-like-particles" aria-hidden="true">
        {particles.map((style, index) => (
          <i key={index} style={style} />
        ))}
      </span>
    </Toggle>
  )
}
