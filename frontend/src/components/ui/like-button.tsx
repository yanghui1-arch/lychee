import { cn } from '@/lib/utils'
import { motion, useReducedMotion } from 'motion/react'
import { Heart } from 'lucide-react'
import { Toggle } from './toggle'

// Transitions.dev timing and particle vectors, rendered with Motion.
// Attribution: frontend/licenses/transitions-LICENSE.txt
const particles = Array.from({ length: 8 }, (_, index) => {
  const angle = (index * Math.PI) / 4
  return {
    x: 20 * Number(Math.cos(angle).toFixed(3)),
    y: 20 * Number(Math.sin(angle).toFixed(3)),
  }
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
  const reducedMotion = useReducedMotion()
  return (
    <Toggle
      variant="outline"
      className="t-like group/like"
      data-liked={liked}
      pressed={liked}
      onPressedChange={onLikedChange}
      disabled={disabled}
      aria-label="喜欢译文"
      title={liked ? '取消喜欢' : '喜欢译文'}
    >
      <motion.span
        aria-hidden="true"
        initial={false}
        animate={{ scale: liked && !reducedMotion ? [1, 0.82, 1] : 1 }}
        transition={{
          duration: reducedMotion ? 0 : 0.35,
          times: [0, 0.3, 1],
          ease: [0.34, 1.96, 0.64, 1],
        }}
      >
        <Heart
          className={cn(
            'size-3.5 text-current transition-colors duration-150 ease-[cubic-bezier(0.22,1,0.36,1)]',
            'group-data-[liked=true]/like:text-[#f40051] [&_path]:fill-transparent [&_path]:stroke-current',
            '[&_path]:transition-[fill,stroke] [&_path]:duration-150 [&_path]:ease-[cubic-bezier(0.22,1,0.36,1)]',
            'group-data-[liked=true]/like:[&_path]:fill-current',
          )}
        />
      </motion.span>
      <span
        className="pointer-events-none absolute top-1/2 left-1/2 size-0 text-[#f40051]"
        aria-hidden="true"
      >
        {particles.map(({ x, y }, index) => (
          <motion.i
            key={index}
            className="absolute -top-[1.25px] -left-[1.25px] size-[2.5px] rounded-[50%] bg-current opacity-0"
            initial={false}
            animate={
              liked && !reducedMotion
                ? {
                    opacity: [0, 1, 0],
                    x: [0, x * 0.25, x],
                    y: [0, y * 0.25, y],
                    scale: [0.4, 1, 0.6],
                  }
                : { opacity: 0, x: 0, y: 0, scale: 1 }
            }
            transition={{
              duration: reducedMotion ? 0 : 0.6,
              times: [0, 0.2, 1],
              ease: 'easeOut',
            }}
          />
        ))}
      </span>
    </Toggle>
  )
}
